package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"path"
	"runtime"
	"time"

	"golang.org/x/net/http2"
)

var certPath string

func init() {
	setupCertPath()
}

func setupCertPath() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current frame")
	}

	certPath = path.Dir(filename)
}

func main() {
	req, err := http.NewRequest("GET", "https://localhost:3443", nil)
	if err != nil {
		log.Fatalf("unable to create request: %v", err)
	}

	var t0, t1, t2, t3, t4, t5, t6 time.Time

	trace := &httptrace.ClientTrace{
		ConnectStart: func(_, _ string) {
			if t1.IsZero() {
				// connecting to IP
				t1 = time.Now()
			}
			fmt.Println("Connecting to IP...")
		},
		ConnectDone: func(net, addr string, err error) {
			if err != nil {
				log.Fatalf("unable to connect to host %v: %v", addr, err)
			}
			t2 = time.Now()

			// printf("\n%s%s\n", color.GreenString("Connected to "), color.CyanString(addr))
			fmt.Printf("\nConnected to %s\n", addr)
		},
		DNSStart:             func(_ httptrace.DNSStartInfo) { t0 = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { t1 = time.Now() },
		GotConn:              func(_ httptrace.GotConnInfo) { t3 = time.Now() },
		GotFirstResponseByte: func() { t4 = time.Now() },
		TLSHandshakeStart:    func() { t5 = time.Now() },
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			t6 = time.Now()
			if err != nil {
				log.Fatalf("failed to perform TLS handshake: %v", err)
			}
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	client := &http.Client{
		Transport: transportHttp2(),
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to read response: %v", err)
	}

	// res, err := client.Get("https://localhost:3443")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	body := getBody(res)

	res.Body.Close()

	t7 := time.Now() // after read body

	dnsLookup := t1.Sub(t0)        // dns lookup
	tcpConnection := t2.Sub(t1)    // tcp connection
	tlsHandshake := t6.Sub(t5)     // tls handshake
	serverProcessing := t4.Sub(t3) // server processing
	contentTransfer := t7.Sub(t4)  // content transfer
	connect := t2.Sub(t0)          // connect
	preTransfer := t3.Sub(t0)      // pretransfer
	startTransfer := t4.Sub(t0)    // starttransfer
	// total := t7.Sub(t0)            // total

	fmt.Printf("Protocol: %s\n", res.Proto)
	fmt.Printf("Code: %d\n", res.StatusCode)
	fmt.Printf("Body: %s\n", body)

	fmt.Printf("\nDNS Lookup: %s\n", dnsLookup)
	fmt.Printf("Connection time: %s\n", tcpConnection)
	fmt.Printf("TLS handshake: %s\n", tlsHandshake)
	fmt.Printf("Server processing: %s\n", serverProcessing)
	fmt.Printf("Content transfer: %s\n", contentTransfer)
	fmt.Printf("Connection: %s\n", connect)
	fmt.Printf("Pretransfer: %s\n", preTransfer)
	fmt.Printf("Starttransfer: %s\n", startTransfer)
	fmt.Printf("Total: %s\n", t7.Sub(t0))
}

func transportHttp2() *http2.Transport {
	return &http2.Transport{
		TLSClientConfig: tlsConfig(),
	}
}

func tlsConfig() *tls.Config {
	crt, err := os.ReadFile(path.Join(certPath, "cert.pem"))
	if err != nil {
		log.Fatal(err)
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(crt)

	return &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: false,
		ServerName:         "localhost",
	}
}

func getBody(response *http.Response) []byte {
	body := &bytes.Buffer{}

	_, err := io.Copy(body, response.Body)

	if err != nil {
		log.Fatal(err)
	}

	return body.Bytes()
}
