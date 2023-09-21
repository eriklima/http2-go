package main

import (
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"golang.org/x/net/http2"
)

// var randomResponse *[]byte
var responses [4]*[]byte

func main() {
	addr := flag.String("addr", "localhost:3443", "Server listening to IP:PORT")
	nbytes := flag.Int("bytes", 1_000_000, "Number of bytes to send to the server")
	flag.Parse()

	fmt.Println("Creating buffer...")

	// randomResponse = createBuf(1000 * 1000 * 1000)
	// randomResponse = createBuf(*nbytes)

	responses[0] = createBuf(*nbytes)
	responses[1] = createBuf(*nbytes * 2)
	responses[2] = createBuf(*nbytes * 4)
	responses[3] = createBuf(*nbytes * 8)

	// fmt.Printf("Buffer created with %d bytes\n", len(*randomResponse))

	// http.HandleFunc("/test", baseHandler)
	// log.Fatal(http.ListenAndServeTLS(":3443", "keys/cert.pem", "keys/priv.key", nil))

	server := &http.Server{
		Addr:      *addr,
		TLSConfig: tlsConfig(),
	}

	if err := http2.ConfigureServer(server, nil); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", baseHandler)

	fmt.Printf("HTTP2 Server listening on %s\n", *addr)

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

	paramString := r.URL.Path[1:]
	paramNumber, err := strconv.ParseInt(paramString, 10, 0)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Param: %d\n", paramNumber)

	response := *responses[paramNumber-1]

	w.Header().Add("X-Body-Size", fmt.Sprintf("%d", len(response)))
	w.Write(response)
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
