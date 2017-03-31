package main

import (
	"log"
	"net/http"

	"webrtcmcu/signaling"

	"os"

	_ "github.com/gorilla/websocket"
)

func main() {
	log.Println("Starting serivces...")

	//创建websocket服务
	http.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		signaling.ServeWebSocket(w, r)
	})

	//创建https文件服务
	http.Handle("/", http.FileServer(http.Dir("browser")))

	//// One can use generate_cert.go in crypto/tls to generate cert.pem and key.pem.
	err := http.ListenAndServeTLS(":8888", "signaling/cert/certificate.pem", "signaling/cert/privatekey.pem", nil)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	//代码不会执行到这里来！！
}
