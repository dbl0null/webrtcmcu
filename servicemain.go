package main

import (
	"log"
	"net/http"

	_ "github.com/gorilla/websocket"
)

func main() {
	http.HandleFunc("/websocket", websocketHandler)
	http.Handle("/", http.FileServer(http.Dir("browser")))
	log.Printf("About to listen on 8888. Go to https://127.0.0.1:8888/")

	//// One can use generate_cert.go in crypto/tls to generate cert.pem and key.pem.
	err := http.ListenAndServeTLS(":8888", "signaling/certificate.pem", "signaling/privatekey.pem", nil)
	log.Fatal(err)
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("This is an example server.\n"))
}
