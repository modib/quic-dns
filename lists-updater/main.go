package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"

	"github.com/gocarina/gocsv"
	"github.com/gofrs/flock"
)

func main() {
	workDir := flag.String("wd", "", "path to working directory")
	flag.Parse()
	err := os.Chdir(*workDir)
	if err != nil {
		log.Fatalf("Unable to open %s: %v", *workDir, err)
	}

	fileLock := flock.New("lockfile")
	lockSuccess, err := fileLock.TryLock()
	if err != nil {
		log.Fatalf("Unable to take a lock: %v", err)
	}
	if !lockSuccess {
		log.Fatalf("Unable to take a lock")
	}
	defer fileLock.Unlock()
	log.Print("Lock file successfully taken")

	err = writeWhitelist()
	if err != nil {
		log.Fatalf("Unable to update whitelist: %v", err)
	}

	err = writeBlacklist()
	if err != nil {
		log.Fatalf("Unable to update blacklist: %v", err)
	}

	////////////

	whitelistRE, err := regexp.Compile("^whitelist\\.(\\d+)\\.txt$")
	maxWhitelistNumber, err := maxNumberForRE(*workDir, whitelistRE)
	if err != nil {
		log.Fatalf("Unable to find latest whitelist number: %v", err)
	}
	err = os.Rename("whitelist.new.txt", fmt.Sprintf("whitelist.%d.txt", maxWhitelistNumber+1))
	if err != nil {
		log.Fatalf("Unable to rename whitelist: %v", err)
	}

	blacklistRE, err := regexp.Compile("^blacklist\\.(\\d+)\\.txt$")
	maxBlacklistNumber, err := maxNumberForRE(*workDir, blacklistRE)
	if err != nil {
		log.Fatalf("Unable to find latest blacklist number: %v", err)
	}
	err = os.Rename("blacklist.new.txt", fmt.Sprintf("blacklist.%d.txt", maxBlacklistNumber+1))
	if err != nil {
		log.Fatalf("Unable to rename blacklist: %v", err)
	}
}

func maxNumberForRE(workDir string, selector *regexp.Regexp) (int, error) {
	wd, err := os.Open(workDir)
	if err != nil {
		return 0, err
	}
	names, err := wd.Readdirnames(0)

	maxFileNumber := -1

	for _, filename := range names {
		match := selector.FindStringSubmatch(filename)
		if match == nil {
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err != nil {
			log.Printf("Warning: unable to parse filename: %q", filename)
			continue
		}
		if maxFileNumber < number {
			maxFileNumber = number
		}
	}

	return maxFileNumber, nil
}

func writeWhitelist() error {
	whiteFile, err := os.OpenFile("whitelist.new.txt", os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer whiteFile.Close()

	majestic1M, err := majestic1MFetch()
	if err != nil {
		return err
	}
	log.Printf("Majestic domains count is %d", len(majestic1M))
	for _, domain := range majestic1M {
		_, err := fmt.Fprintln(whiteFile, domain)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeBlacklist() error {
	blackFile, err := os.OpenFile("blacklist.new.txt", os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer blackFile.Close()

	phishTankDomains, err := phishTankFetch()
	if err != nil {
		return err
	}
	log.Printf("PhishTank domains count = %d", len(phishTankDomains))

	blackDomains := map[string]bool{}
	for _, domain := range phishTankDomains {
		if blackDomains[domain] {
			continue
		}
		blackDomains[domain] = true
		_, err := fmt.Fprintln(blackFile, domain)
		if err != nil {
			return err
		}
	}

	openPhishDomains, err := openPhishFetch()
	log.Printf("OpenPhish domains count = %d", len(openPhishDomains))
	for _, domain := range openPhishDomains {
		if blackDomains[domain] {
			continue
		}
		blackDomains[domain] = true
		_, err := fmt.Fprintln(blackFile, domain)
		if err != nil {
			return err
		}
	}
	return nil
}

func urlToDomain(input string) (string, error) {
	parsed, err := url.Parse(input)
	if err != nil {
		return "", err
	}

	// TODO: in the future we can filter A and AAAA record which points to IP
	hostname := parsed.Hostname()
	if net.ParseIP(hostname) != nil {
		return "", fmt.Errorf("host name is IP (%s)", hostname)
	}

	return hostname, nil
}

type PhishTankEntry struct {
	URL string `csv:"url"`
}

func phishTankFetch() ([]string, error) {
	resp, err := http.Get("http://data.phishtank.com/data/online-valid.csv")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Bad HTTP code %d", resp.StatusCode)
	}

	data := []PhishTankEntry{}
	err = gocsv.Unmarshal(resp.Body, &data)
	if err != nil {
		return nil, err
	}

	looked := map[string]bool{}
	result := []string{}
	for _, ptEntry := range data {
		domain, err := urlToDomain(ptEntry.URL)
		if err != nil {
			log.Printf("Unable to extract domain: %v", err)
			continue
		}
		if looked[domain] {
			continue
		}

		looked[domain] = true
		result = append(result, domain)
	}
	return result, nil
}

type MajesticEntry struct {
	Domain string `csv:"Domain"`
}

func majestic1MFetch() ([]string, error) {
	resp, err := http.Get("http://downloads.majestic.com/majestic_million.csv")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	log.Printf("Got response from Majestic")

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Bad HTTP code %d", resp.StatusCode)
	}

	data := []MajesticEntry{}
	err = gocsv.Unmarshal(resp.Body, &data)
	if err != nil {
		return nil, err
	}

	result := []string{}
	for _, entry := range data {
		result = append(result, entry.Domain)
	}

	return result, nil
}

func openPhishFetch() ([]string, error) {
	resp, err := http.Get("http://openphish.com/feed.txt")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	looked := map[string]bool{}
	result := []string{}
	for scanner.Scan() {
		domain, err := urlToDomain(scanner.Text())
		if err != nil {
			log.Printf("Unable to extract domain: %v", err)
			continue
		}
		if looked[domain] {
			continue
		}

		looked[domain] = true
		result = append(result, domain)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
