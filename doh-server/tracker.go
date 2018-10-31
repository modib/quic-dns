package main

import (
	"bytes"
	"log"
	"os"
	"time"
)

type trackerEntry struct {
	domain string
	date   time.Time
}

type Tracker struct {
	entries chan *trackerEntry
	file    *os.File
}

func NewTracker(filepath string) (*Tracker, error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	return &Tracker{make(chan *trackerEntry, 1000), file}, nil
}

func (t *Tracker) SaveDomain(domain string) {
	t.entries <- &trackerEntry{
		domain,
		time.Now(),
	}
}

func (t *Tracker) Start() {
	go func() {
		log.Printf("Tracker is started")
		buf := bytes.NewBuffer([]byte{})
		ticker := time.Tick(time.Second * 30)

		for {
			select {
			case <-ticker:
				_, err := buf.WriteTo(t.file)
				if err != nil {
					log.Printf("[Warning] Unable to write error to requests log")
				}
				buf.Reset()
			case entry := <-t.entries:
				buf.WriteString(entry.date.UTC().Format("06-01-02 15:04:05 - "))
				buf.WriteString(entry.domain)
				buf.WriteString("\n")
				if buf.Len() >= 2048 {
					_, err := buf.WriteTo(t.file)
					if err != nil {
						log.Printf("[Warning] Unable to write error to requests log")
					}
					buf.Reset()
				}
			}
		}
	}()
}
