package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/gocarina/gocsv"
)

func main() {
	phishTankDomains, err := phishTankFetch()
	fmt.Println(err)
	fmt.Println(phishTankDomains)
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
	resp, err := http.Get("https://data.phishtank.com/data/online-valid.csv")
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
