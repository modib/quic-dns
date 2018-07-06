package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/lucas-clemente/quic-go/h2quic"
)

func main() {
	flag.Parse()
	urls := flag.Args()

	roundTripper := &h2quic.RoundTripper{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	defer roundTripper.Close()
	hclient := &http.Client{
		Transport: roundTripper,
	}

	var wg sync.WaitGroup
	wg.Add(len(urls))
	for _, addr := range urls {
		log.Printf("GET %s", addr)
		go func(addr string) {
			rsp, err := hclient.Get(addr)
			if err != nil {
				panic(err)
			}
			log.Printf("Got response for %s: %#v", addr, rsp)

			body := &bytes.Buffer{}
			_, err = io.Copy(body, rsp.Body)
			if err != nil {
				panic(err)
			}
			log.Printf("Request Body:")
			log.Printf("%s", body.Bytes())
			wg.Done()
		}(addr)
	}
	wg.Wait()
}
