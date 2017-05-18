package main

import (
	"log"
	"net/http"
	"os"
	"webrtcmcu/room"
	"webrtcmcu/signaling"
)

func main() {
	log.Println("Starting serivces...")

	roomMgr := room.NewRoomMgr()
	go roomMgr.RoomService()

	//创建websocket服务
	http.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		signaling.ServeWebSocket(roomMgr, w, r)
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
