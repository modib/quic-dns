package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/lucas-clemente/quic-go/h2quic"
	"github.com/miekg/dns"
)

func main() {
	roundTripper := &h2quic.RoundTripper{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	defer roundTripper.Close()
	hclient := &http.Client{
		Transport: roundTripper,
	}

	name := "google.com"
	googleReqResp(hclient, name)
	googleReqIETFResp(hclient, name)
	ietfGetReqResp(hclient, name)
	ietfPostReqResp(hclient, name)
}

func googleReqResp(client *http.Client, name string) {
	fmt.Println("\n\n=====================\ngoogleReqResp\n=====================")
	addr := "https://localhost:8053/dns-query?name=" + name
	log.Printf("GET %s", addr)
	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		panic(err)
	}
	rsp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	log.Printf("Got response for %s: %#v", addr, rsp)

	body := &bytes.Buffer{}
	_, err = io.Copy(body, rsp.Body)
	if err != nil {
		panic(err)
	}
	log.Printf("Response Body:")
	log.Printf("%s", body.Bytes())
}

func googleReqIETFResp(client *http.Client, name string) {
	fmt.Println("\n\n=====================\ngoogleReqIETFResp\n=====================")
	addr := "https://localhost:8053/dns-query?name=yandex.ru"
	log.Printf("GET %s", addr)

	req, err := http.NewRequest("GET", addr, nil)
	req.Header.Add("Accept", "application/dns-message")
	if err != nil {
		panic(err)
	}

	rsp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	log.Printf("Got response for %s: %#v", addr, rsp)

	body := &bytes.Buffer{}
	_, err = io.Copy(body, rsp.Body)
	if err != nil {
		panic(err)
	}

	msg := &dns.Msg{}
	err = msg.Unpack(body.Bytes())
	if err != nil {
		panic(err)
	}

	log.Printf("Response Body:")
	log.Printf("%v", base64.StdEncoding.EncodeToString(body.Bytes()))
	log.Printf("%v", msg)
}

func ietfGetReqResp(client *http.Client, name string) {
	fmt.Println("\n\n=====================\nietfGetReqResp\n=====================")
	dnsQuestion := new(dns.Msg)
	dnsQuestion.SetQuestion(dns.Fqdn(name), dns.TypeA)
	questionBody, err := dnsQuestion.Pack()
	addr := "https://localhost:8053/dns-query?dns=" + base64.RawURLEncoding.EncodeToString(questionBody)
	log.Printf("GET %s", addr)

	req, err := http.NewRequest("GET", addr, nil)
	req.Header.Add("Accept", "application/dns-message")
	if err != nil {
		panic(err)
	}

	rsp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	log.Printf("Got response for %s: %#v", addr, rsp)

	body := &bytes.Buffer{}
	_, err = io.Copy(body, rsp.Body)
	if err != nil {
		panic(err)
	}

	msg := &dns.Msg{}
	err = msg.Unpack(body.Bytes())
	if err != nil {
		panic(err)
	}

	log.Printf("Response Body:")
	log.Printf("%v", base64.StdEncoding.EncodeToString(body.Bytes()))
	log.Printf("%v", msg)
}

func ietfPostReqResp(client *http.Client, name string) {
	fmt.Println("\n\n=====================\nietfPostReqResp\n=====================")
	dnsQuestion := new(dns.Msg)
	dnsQuestion.SetQuestion(dns.Fqdn(name), dns.TypeA)
	questionBody, err := dnsQuestion.Pack()
	addr := "https://localhost:8053/dns-query"
	log.Printf("POST %s", addr)
	reader := bytes.NewBuffer(questionBody)
	req, err := http.NewRequest("POST", addr, reader)
	req.Header.Add("Accept", "application/dns-message")
	req.Header.Add("Content-Type", "application/dns-message")
	if err != nil {
		panic(err)
	}

	rsp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	log.Printf("Got response for %s: %#v", addr, rsp)

	body := &bytes.Buffer{}
	_, err = io.Copy(body, rsp.Body)
	if err != nil {
		panic(err)
	}

	msg := &dns.Msg{}
	err = msg.Unpack(body.Bytes())
	if err != nil {
		panic(err)
	}

	log.Printf("Response Body:")
	log.Printf("%v", base64.StdEncoding.EncodeToString(body.Bytes()))
	log.Printf("%v", msg)
}
