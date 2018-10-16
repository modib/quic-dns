package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

func parseList(filepath string) (map[string]bool, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := map[string]bool{}

	reader := bufio.NewReader(file)
	line := []byte{}
	for ; ; line = line[:0] {
		chunk, isPrefix, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line = append(line, chunk...)
		if isPrefix {
			continue
		}
		domain := strings.ToLower(strings.TrimSpace(string(line)))
		if domain == "" {
			continue
		}
		fqdn := dns.Fqdn(domain)
		result[fqdn] = true
	}

	return result, nil
}

// Prelookup for domains filtering
func (s *Server) preLookup(req *DNSRequest) {
	if len(req.request.Question) != 1 {
		req.errcode = 400
		req.errtext = "Request with many questions is unsupported"
	}

	if req.request.Opcode != dns.OpcodeQuery {
		req.errcode = 400
		req.errtext = "Non-query opcodes are unsupported"
	}
}

// Filtering according to White/Black lists
func (s *Server) postLookup(resp *DNSRequest) {
	// Uncheck authoritative answer because this server is just resolver.
	resp.response.Authoritative = false

	answer := make([]dns.RR, 0)
	for _, rr := range resp.response.Answer {
		if s.isRestricted(rr.Header().Name) {
			// Drop this RR from answer
			// fmt.Printf("Dropping RR: %v\n", rr)
			continue
		}
		answer = append(answer, rr)
	}
	resp.response.Answer = answer

	additional := make([]dns.RR, 0)
	for _, rr := range resp.response.Extra {
		if s.isRestricted(rr.Header().Name) {
			// Drop this RR from additional
			// fmt.Printf("Dropping RR: %v\n", rr)
			continue
		}
		additional = append(additional, rr)
	}
	resp.response.Extra = additional

	// Clearing "Authoritative section" because this server is just resolver.
	// Also client can try to lookup domains out of our server through servers
	// from this section.
	resp.response.Ns = []dns.RR{}
}

func (s *Server) isRestricted(domain string) bool {
	normalized := strings.ToLower(dns.Fqdn(domain))
	s.listsMu.RLock()
	defer s.listsMu.RUnlock()

	if s.whitelist[normalized] {
		return false
	}
	return s.blacklist[normalized]
}

func (s *Server) readLists() error {
	if s.conf.ListsDirectory == "" {
		log.Print("No filtering lists are used")
		return nil
	}
	whitelistRE := regexp.MustCompile("^whitelist\\.(\\d+)\\.txt$")
	whitelist, err := readList(s.conf.ListsDirectory, whitelistRE)
	if err != nil {
		return fmt.Errorf("can't read whitelist: %v", err)
	}
	s.listsMu.Lock()
	s.whitelist = whitelist
	s.listsMu.Unlock()

	blacklistRE := regexp.MustCompile("^blacklist\\.(\\d+)\\.txt$")
	blacklist, err := readList(s.conf.ListsDirectory, blacklistRE)
	if err != nil {
		return fmt.Errorf("Can't read blacklist: %v", err)
	}
	s.listsMu.Lock()
	s.blacklist = blacklist
	s.listsMu.Unlock()

	s.listsMu.RLock()
	log.Printf("Blacklist size: %v", len(s.blacklist))
	log.Printf("Whitelist size: %v", len(s.whitelist))
	s.listsMu.RUnlock()

	return nil
}

func readList(dir string, selector *regexp.Regexp) (map[string]bool, error) {
	listName, err := getMaxNumberFilenameForRE(dir, selector)
	if err != nil {
		return nil, err
	}
	list, err := parseList(path.Join(dir, listName))
	if err != nil {
		return nil, fmt.Errorf("Can't read whitelist: %v", err)
	}
	return list, nil
}

func getMaxNumberFilenameForRE(workDir string, selector *regexp.Regexp) (string, error) {
	wd, err := os.Open(workDir)
	if err != nil {
		return "", err
	}
	names, err := wd.Readdirnames(0)
	if err != nil {
		return "", err
	}

	maxFileNumber := -1
	numberToFilename := map[int]string{}
	for _, filename := range names {
		match := selector.FindStringSubmatch(filename)
		if match == nil {
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err != nil {
			log.Printf("[Warning] Unable to parse filename: %q", filename)
			continue
		}
		numberToFilename[number] = filename
		if maxFileNumber < number {
			maxFileNumber = number
		}
	}

	if maxFileNumber == -1 {
		return "", fmt.Errorf("no files are found for RE (%v)", selector)
	}

	return numberToFilename[maxFileNumber], nil
}

func (s *Server) startListsUpdateEndpoint() error {
	if s.conf.ListsUpdateEndpoint == "" {
		return nil
	}

	log.Printf("Lists update endpoint address: %v", s.conf.ListsUpdateEndpoint)

	theURL, err := url.Parse(s.conf.ListsUpdateEndpoint)
	if err != nil {
		return err
	}

	http.HandleFunc(theURL.Path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(400)
			fmt.Fprintf(w, "Bad request")
			return
		}

		fmt.Fprintf(w, "Ok")
		go func() {
			log.Printf("Rereading the lists")
			err := s.readLists()
			if err != nil {
				log.Printf("[Warning] Unable to reread lists: %v", err)
			}
		}()
	})

	go func() {
		err := http.ListenAndServe(theURL.Host, nil)
		if err != nil {
			log.Fatalf("Unable to start update lists endpoint: %v", err)
		}
	}()

	return nil
}
