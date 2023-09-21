package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync"
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
	server := flag.String("server", "localhost:3443", "IP:PORT for HTTP2 server")
	parallel := flag.Int("parallel", 1, "Number of parallel requests")
	experNumber := flag.Int("expernumber", 1, "Number of the experiment")
	flag.Parse()

	loopCount := *parallel
	var wg sync.WaitGroup

	wg.Add(loopCount)

	for loopCount > 0 {
		loopCount -= 1

		go func() {
			// buf := createBuf(1000 * 1000 * 1000)
			buf := createBuf(0)

			var req *http.Request
			var err error
			// host := "https://localhost:3443"
			// host := "https://192.168.1.8:3443"
			// host := "https://193.167.100.100:3443"
			host := "https://" + *server + "/" + strconv.Itoa(*experNumber)

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

			// fmt.Printf("METHOD :::: %s\n", req.Method)

			tInit := time.Now()

			var t0, t1, t2, t3, t4, t5, t6, tGetConn time.Time

			// TODO:: Verificar se os ts foram definidos antes de calcular o tempo

			trace := &httptrace.ClientTrace{
				ConnectStart: func(_, _ string) {
					if t1.IsZero() {
						// connecting to IP
						t1 = time.Now()
					}
					// fmt.Println("EVENT: ConnectStart", t1.Sub(tInit))
				},
				ConnectDone: func(net, addr string, err error) {
					if err != nil {
						log.Fatalf("unable to connect to host %v: %v", addr, err)
					}
					t2 = time.Now()

					// fmt.Println("EVENT: ConnectDone", t2.Sub(tInit))
				},
				DNSStart: func(_ httptrace.DNSStartInfo) { t0 = time.Now() },
				DNSDone:  func(_ httptrace.DNSDoneInfo) { t1 = time.Now() },
				GetConn: func(_ string) {
					tGetConn = time.Now()
					// fmt.Println("EVENT: GetConn", tGetConn.Sub(tInit))
				},
				GotConn: func(_ httptrace.GotConnInfo) {
					t3 = time.Now()
					// fmt.Println("EVENT: GotConn", t3.Sub(tInit))
				},
				GotFirstResponseByte: func() {
					t4 = time.Now()
					// fmt.Println("EVENT: GotFirstResponseByte", t4.Sub(tInit))
				},
				TLSHandshakeStart: func() {
					t5 = time.Now()
					fmt.Println("EVENT: TLSHandshakeStart", t5.Sub(tInit))
				},
				TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
					t6 = time.Now()
					if err != nil {
						log.Fatalf("failed to perform TLS handshake: %v", err)
					}
					fmt.Println("EVENT: TLSHandshakeDone", t6.Sub(tInit))
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

			body := getBody(res)

			t7 := time.Now() // after read body
			// fmt.Println("EVENT: ResponseReady", t7.Sub(tInit))

			// res, err := client.Get("https://localhost:3443")
			// if err != nil {
			// 	log.Fatal(err)
			// }

			bodySize := res.Header.Get("X-Body-Size")

			// responseLength := res.Body.
			// fmt.Printf("Body size: %v\n", body)
			// fmt.Printf("Body size Recalc: %d\n", len(body))

			// saveBody(body)

			res.Body.Close()

			// t7 := time.Now() // after read body

			dnsLookup := t1.Sub(t0) // dns lookup
			// tcpConnection := t2.Sub(t1)    // tcp connection
			tcpConnection := t3.Sub(tGetConn) // tcp connection
			tlsHandshake := t6.Sub(t5)        // tls handshake
			serverProcessing := t4.Sub(t3)    // server processing
			contentTransfer := t7.Sub(t3)     // content transfer
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

			fmt.Printf("\nProtocol: %s %s\n", res.Proto, req.Method)
			fmt.Printf("Code: %d\n", res.StatusCode)
			fmt.Printf("Content-Length: %d\n", res.ContentLength)
			fmt.Printf("Body: %d\n", len(body))
			// fmt.Printf("Body Length: %d\n\n", responseLength)
			fmt.Printf("Body Size Header: %s\n", bodySize)

			// fmt.Printf("\nDNS Lookup: %s\n", dnsLookup)
			fmt.Printf("\nConnection time: %s\n", tcpConnection)
			// fmt.Printf("TLS handshake: %s\n", tlsHandshake)
			// fmt.Printf("Server processing: %s\n", serverProcessing)
			fmt.Printf("Content transfer: %s\n", contentTransfer)
			// fmt.Printf("Connection: %s\n", connect)
			// fmt.Printf("Pretransfer: %s\n", preTransfer)
			// fmt.Printf("Starttransfer: %s\n", startTransfer)
			fmt.Printf("Total: %s\n", total)
			fmt.Print("-----------------------------------------\n")

			wg.Done()
		}()
	}

	wg.Wait()
}

func transportHttp2() *http2.Transport {
	return &http2.Transport{
		TLSClientConfig: tlsConfig(),
	}
}

func tlsConfig() *tls.Config {
	// crt, err := os.ReadFile(path.Join(certPath, "cert.pem"))
	crt, err := os.ReadFile("./cert.pem")
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

// func getBody(response *http.Response) []byte {
// 	body, err := io.ReadAll(response.Body)

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	return body
// }

// func saveBody(body []byte) {
// 	f, err := os.OpenFile("./body.txt", os.O_WRONLY|os.O_CREATE, 0644)
// 	check(err)

// 	defer f.Close()

// 	wBytes, err := f.Write(body)
// 	check(err)

// 	fmt.Printf("Bytes written: %d\n", wBytes)

// 	f.Sync()
// }

// func check(e error) {
// 	if e != nil {
// 		panic(e)
// 	}
// }

func saveMetrics(metrics *Metrics) {
	// f, err := os.OpenFile(path.Join(certPath, "metrics.csv"), os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	f, err := os.OpenFile("/logs/metrics.csv", os.O_WRONLY|os.O_APPEND, os.ModeAppend)
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
