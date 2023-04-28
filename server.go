package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/http2"
)

func main() {
	fmt.Println("Hello, World!")

	// http.HandleFunc("/test", baseHandler)
	// log.Fatal(http.ListenAndServeTLS(":3443", "keys/cert.pem", "keys/priv.key", nil))

	server := &http.Server{
		Addr:      ":3443",
		TLSConfig: tlsConfig(),
	}

	if err := http2.ConfigureServer(server, nil); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", baseHandler)

	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Server running on port 3443")
}

func baseHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Protocol: ", r.Proto, r.Method)
	fmt.Fprintf(w, "Hello, World!")
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
