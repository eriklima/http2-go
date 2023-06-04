package main

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/http2"
)

var randomResponse *[]byte

func main() {
	fmt.Println("Creating buffer...")

	randomResponse = createBuf(1000 * 1000 * 1000)

	fmt.Printf("Buffer created with %d bytes\n", len(*randomResponse))

	// http.HandleFunc("/test", baseHandler)
	// log.Fatal(http.ListenAndServeTLS(":3443", "keys/cert.pem", "keys/priv.key", nil))

	serverAddr := ":3443"

	server := &http.Server{
		Addr:      serverAddr,
		TLSConfig: tlsConfig(),
	}

	if err := http2.ConfigureServer(server, nil); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", baseHandler)

	fmt.Printf("HTTP2 Server listening on %s\n", serverAddr)

	// TODO: testar passando certificado e chave aqui (igual ao HTTP3) e não no TLSConfig
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Server running on PORT: 3443")
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

func baseHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Protocol: ", r.Proto, r.Method)
	// fmt.Fprintf(w, "Hello, World!")
	w.Write(*randomResponse)
}

func tlsConfig() *tls.Config {
	crt, err := os.ReadFile("./keys/cert.pem")
	if err != nil {
		log.Fatal(err)
	}

	key, err := os.ReadFile("./keys/priv.key")
	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		log.Fatal(err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "localhost",
	}
}
