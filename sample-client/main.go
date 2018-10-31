package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/lucas-clemente/quic-go/h2quic"
	"github.com/miekg/dns"
)

func main() {
	baseURL := flag.String("url", "", "Full URL to DOH HTTP endpoint for example https://localhost/dns-query")
	flag.Parse()
	if *baseURL == "" {
		log.Fatal("Base url isn't defined")
	}

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
	googleReqResp(*baseURL, hclient, name)
	googleReqIETFResp(*baseURL, hclient, name)
	ietfGetReqResp(*baseURL, hclient, name)
	ietfPostReqResp(*baseURL, hclient, name)
}

func googleReqResp(baseURL string, client *http.Client, name string) {
	fmt.Println("\n\n=====================\ngoogleReqResp\n=====================")
	addr := baseURL + "?name=" + name
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

func googleReqIETFResp(baseURL string, client *http.Client, name string) {
	fmt.Println("\n\n=====================\ngoogleReqIETFResp\n=====================")
	addr := baseURL + "?name=yandex.ru"
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

func ietfGetReqResp(baseURL string, client *http.Client, name string) {
	fmt.Println("\n\n=====================\nietfGetReqResp\n=====================")
	dnsQuestion := new(dns.Msg)
	dnsQuestion.SetQuestion(dns.Fqdn("reg.ru"), dns.TypeA)
	questionBody, err := dnsQuestion.Pack()
	addr := baseURL + "?dns=" + base64.RawURLEncoding.EncodeToString(questionBody)
	log.Printf("GET %s", addr)

	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Accept", "application/dns-message")
	req.Header.Add("DNT", "1")

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

func ietfPostReqResp(baseURL string, client *http.Client, name string) {
	fmt.Println("\n\n=====================\nietfPostReqResp\n=====================")
	dnsQuestion := new(dns.Msg)
	dnsQuestion.SetQuestion(dns.Fqdn(name), dns.TypeA)
	questionBody, err := dnsQuestion.Pack()
	log.Printf("POST %s", baseURL)
	reader := bytes.NewBuffer(questionBody)
	req, err := http.NewRequest("POST", baseURL, reader)
	req.Header.Add("Accept", "application/dns-message")
	req.Header.Add("Content-Type", "application/dns-message")
	if err != nil {
		panic(err)
	}

	rsp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	log.Printf("Got response for %s: %#v", baseURL, rsp)

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
