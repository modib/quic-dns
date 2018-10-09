package main

import (
	"bufio"
	"io"
	"os"
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
	if s.whitelist[normalized] {
		return false
	}
	return s.blacklist[normalized]
}
