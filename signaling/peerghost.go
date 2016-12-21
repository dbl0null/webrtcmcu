package signaling

import (
	"log"
	"net/http"
	"time"

	"encoding/json"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 5 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 10 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  maxMessageSize,
	WriteBufferSize: maxMessageSize,
}

type WebrtcMessage struct {
	Type, Sdp string
}

// PeerGhost 伪装成Browser的PeerConnection对端
type PeerGhost struct {
	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

//peerMsgHandler 处理从Browser那边传过来的数据
func (ghost *PeerGhost) peerMsgHandler(jsonMessage []byte) {
	var msg WebrtcMessage
	json.Unmarshal(jsonMessage, &msg)
	log.Println("peerMsgHandler()", msg.Sdp)
}

//从头Browser->WSClient通道读数据
func (ghost *PeerGhost) readPump() {
	log.Println(">>>readPump()")
	defer func() {
		ghost.conn.Close()
	}()
	ghost.conn.SetReadLimit(maxMessageSize)
	ghost.conn.SetReadDeadline(time.Now().Add(pongWait))
	ghost.conn.SetPongHandler(func(string) error { ghost.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		messageType, message, err := ghost.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("readPump|Websocket UnexpectedCloseError: %v", err)
			} else {
				log.Printf("readPump|Websocket Error: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			ghost.peerMsgHandler(message)
		} else if messageType == websocket.BinaryMessage {
			log.Println("Not support BinaryMessage(2) for now!")
		} else {
			log.Printf("Other message?! %v", message)
		}
	}
}

//writePump 从PeerGhost的send管道里面取数据发给Browser端
func (ghost *PeerGhost) writePump() {
	log.Println(">>>writePump()")
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		ghost.conn.Close()
	}()
	for {
		select {
		case message, ok := <-ghost.send:
			ghost.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				//后台强行关闭连接
				ghost.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := ghost.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(ghost.send)
			for i := 0; i < n; i++ {
				w.Write(<-ghost.send)
			}

			//Close()是为了把数据刷到底层发送出去
			if err := w.Close(); err != nil {
				log.Println("writePump()|io.WriterCloser.Close() error!", err)
				return
			}
		case <-ticker.C:
			ghost.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ghost.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

//ServeWebSocket handles websocket requests from the peer.
func ServeWebSocket(w http.ResponseWriter, r *http.Request) {
	wsConnection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	ghost := &PeerGhost{conn: wsConnection, send: make(chan []byte, 256)}
	go ghost.writePump()
	ghost.readPump()
}
