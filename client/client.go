package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"path"
	"runtime"
	"time"

	"golang.org/x/net/http2"
)

type Metrics struct {
	dnsLookup        time.Duration
	tcpConnection    time.Duration
	tlsHandshake     time.Duration
	serverProcessing time.Duration
	contentTransfer  time.Duration
	connect          time.Duration
	preTransfer      time.Duration
	startTransfer    time.Duration
	total            time.Duration
}

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
	// buf := createBuf(1000 * 1000 * 1000)
	buf := createBuf(0)

	var req *http.Request
	var err error
	// host := "https://192.168.1.8:3443"
	host := "https://localhost:3443"

	if len(*buf) == 0 {
		req, err = http.NewRequest("GET", host, nil)
		if err != nil {
			log.Fatalf("unable to create request: %v", err)
		}
	} else {
		req, err = http.NewRequest("POST", host, bytes.NewReader(*buf))
		if err != nil {
			log.Fatalf("unable to create request: %v", err)
		}
	}

	fmt.Printf("METHOD :::: %s\n", req.Method)

	var t0, t1, t2, t3, t4, t5, t6, tGetConn time.Time

	// TODO:: Verificar se os ts foram definidos antes de calcular o tempo

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

			fmt.Printf("\nConnected to %s\n", addr)
		},
		DNSStart:             func(_ httptrace.DNSStartInfo) { t0 = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { t1 = time.Now() },
		GetConn:              func(_ string) { tGetConn = time.Now() },
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

	t7 := time.Now() // after read body

	// res, err := client.Get("https://localhost:3443")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// body := getBody(res)
	// responseLength := res.Body.

	res.Body.Close()

	// t7 := time.Now() // after read body

	dnsLookup := t1.Sub(t0) // dns lookup
	// tcpConnection := t2.Sub(t1)    // tcp connection
	tcpConnection := t3.Sub(tGetConn) // tcp connection
	tlsHandshake := t6.Sub(t5)        // tls handshake
	serverProcessing := t4.Sub(t3)    // server processing
	contentTransfer := t7.Sub(t4)     // content transfer
	connect := t2.Sub(t1)             // connect
	preTransfer := t3.Sub(t0)         // pretransfer
	startTransfer := t4.Sub(t0)       // starttransfer
	// total := t7.Sub(t0)            // total
	total := t7.Sub(tGetConn) // total
	// total := t7.Sub(t1) // total

	metrics := &Metrics{
		dnsLookup:        dnsLookup,
		tcpConnection:    tcpConnection,
		tlsHandshake:     tlsHandshake,
		serverProcessing: serverProcessing,
		contentTransfer:  contentTransfer,
		connect:          connect,
		preTransfer:      preTransfer,
		startTransfer:    startTransfer,
		total:            total,
	}

	saveMetrics(metrics)

	fmt.Printf("Protocol: %s\n", res.Proto)
	fmt.Printf("Code: %d\n", res.StatusCode)
	// fmt.Printf("Body: %s\n", body)
	// fmt.Printf("Body Length: %d\n\n", responseLength)

	// fmt.Printf("\nDNS Lookup: %s\n", dnsLookup)
	fmt.Printf("Connection time: %s\n", tcpConnection)
	// fmt.Printf("TLS handshake: %s\n", tlsHandshake)
	fmt.Printf("Server processing: %s\n", serverProcessing)
	fmt.Printf("Content transfer: %s\n", contentTransfer)
	// fmt.Printf("Connection: %s\n", connect)
	// fmt.Printf("Pretransfer: %s\n", preTransfer)
	// fmt.Printf("Starttransfer: %s\n", startTransfer)
	fmt.Printf("Total: %s\n", total)
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

// func getBody(response *http.Response) []byte {
// 	body := &bytes.Buffer{}

// 	_, err := io.Copy(body, response.Body)

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	return body.Bytes()
// }

func saveMetrics(metrics *Metrics) {
	f, err := os.OpenFile(path.Join(certPath, "metrics.csv"), os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		// fmt.Println(err)
		// return
		log.Fatal(err)
	}
	w := csv.NewWriter(f)
	// for i := 0; i < 10; i++ {
	// 	w.Write([]string{"a", "b", "c"})
	// }
	row := []string{
		metrics.dnsLookup.String(),
		metrics.tcpConnection.String(),
		metrics.tlsHandshake.String(),
		metrics.serverProcessing.String(),
		metrics.contentTransfer.String(),
		metrics.connect.String(),
		metrics.preTransfer.String(),
		metrics.startTransfer.String(),
		metrics.total.String(),
	}

	w.Write(row)

	w.Flush()
}

func createBuf(size int) *[]byte {
	buf := make([]byte, 0)

	if size > 0 {
		buf = make([]byte, size)

		// Randomize the buffer
		_, err := rand.Read(buf)

		if err != nil {
			log.Fatalf("error while generating random string: %s", err)
		}
	}

	return &buf
}
