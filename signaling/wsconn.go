package signaling

import (
	"log"
	"net"
	"net/http"
	"time"
	"webrtcmcu/room"

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

type webrtcRecognizer struct {
	Type string `json:"type"`
}

//包括Offer、Answer、Candidate
type messageSDP struct {
	Type   string `json:"type"`
	Sdp    string `json:"sdp"`
	UserID string `json:"userid"`
	RoomID string `json:"roomid"`
	Error  string `json:"error"`
}

type messageEnteroom struct {
	Type         string `json:"type"`
	UserID       string `json:"userid"`
	RoomID       string `json:"roomid"`
	Error        string `json:"error"`
	Participants string `json:"participants"`
}

// WSConnWrap 伪装成Browser的PeerConnection对端
type WSConnWrap struct {
	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	userid string
	roomid string
}

//jsonMsgHandler 处理从Browser那边传过来的数据
func (wswrap *WSConnWrap) jsonMsgHandler(jsonMessage []byte) {
	recognizer := webrtcRecognizer{}
	//先解析一次，把消息类型解析出来
	jsonErr := json.Unmarshal(jsonMessage, &recognizer)
	if jsonErr != nil {
		log.Printf("peerMsgHandler| Can't recognize msg type. %v", jsonErr)
	}

	switch recognizer.Type {
	case "offer":
		msg := messageSDP{}
		//根据消息类型，解析不同的消息结构
		jsonErr = json.Unmarshal(jsonMessage, &msg)
		if jsonErr != nil {
			log.Printf("peerMsgHandler| Can't parse msg content. %v", jsonErr)
		}

		var offer SdpInfo
		offer.InitWithSdp(msg.Sdp, "")
		//值拷贝，offer还要保留
		answer := offer
		//@TODO H264和Opus要拍在前面
		msg.Sdp = answer.CreateSdp()
		msg.Type = "answer"

		//生成一个新的Offer消息返回给浏览器
		jsonBytes, jsonErr := json.Marshal(msg)
		if jsonErr != nil {
			log.Printf("peerMsgHandler| Can't create anser msg. %v", jsonErr)
		} else {
			wswrap.send <- jsonBytes
		}
	case "candidate":
		msg := messageSDP{}
		//根据消息类型，解析不同的消息结构
		jsonErr = json.Unmarshal(jsonMessage, &msg)
		if jsonErr != nil {
			log.Printf("peerMsgHandler| Can't parse msg content. %v", jsonErr)
		}
		/*
					type: 'candidate',
			      label: event.candidate.sdpMLineIndex,
			      id: event.candidate.sdpMid,
			      candidate: event.candidate.candidate,
			      userid : userIDVal,
			      roomid : roomIDVal
		*/
		log.Println("peerMsgHandler| receive candidate:")
	case "enteroom":
		msg := messageEnteroom{}
		//根据消息类型，解析不同的消息结构
		jsonErr = json.Unmarshal(jsonMessage, &msg)
		if jsonErr != nil {
			log.Printf("peerMsgHandler| Can't parse enteroom msg. %v", jsonErr)
		}

		msg.Error = ""
		msg.Type = "enteroomres"
		msg.Participants = "[" + msg.UserID + "]"
		jsonBytes, jsonErr := json.Marshal(msg)
		if jsonErr != nil {
			log.Printf("peerMsgHandler| Can't create anser msg. %v", jsonErr)
		} else {
			wswrap.send <- jsonBytes
		}
		//@TODO create user info and add to room
		//@TODO send msg to other participants
	default:
		log.Println("peerMsgHandler| Unknown msgtype:")
	}
}

//从头Browser->WSClient通道读数据
func (wswrap *WSConnWrap) readPump() {
	defer func() {
		wswrap.conn.Close()
		//TODO 删除用户信息
	}()
	wswrap.conn.SetReadLimit(maxMessageSize)
	wswrap.conn.SetReadDeadline(time.Now().Add(pongWait))
	wswrap.conn.SetPongHandler(func(string) error { wswrap.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		messageType, message, err := wswrap.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("readPump|Connection close(userid=%s, roomid=%s): err=%v", wswrap.userid, wswrap.roomid, err)
			} else {
				log.Printf("readPump|Connection(userid=%s, roomid=%s) Error: %v", wswrap.userid, wswrap.roomid, err)
			}
			return
		}

		if messageType == websocket.TextMessage {
			wswrap.jsonMsgHandler(message)
		} else if messageType == websocket.BinaryMessage {
			log.Println("readPump|Not support BinaryMessage(2) for now!")
			return
		} else {
			log.Printf("readPump|Other message(3)?! %v", message)
			return
		}
	}
}

//writePump 从PeerGhost的send管道里面取数据发给Browser端
func (wswrap *WSConnWrap) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		wswrap.conn.Close()
	}()
	for {
		select {
		case message, ok := <-wswrap.send:
			wswrap.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				//后台强行关闭连接
				wswrap.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := wswrap.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(wswrap.send)
			for i := 0; i < n; i++ {
				w.Write(<-wswrap.send)
			}

			//Close()是为了把数据刷到底层发送出去
			if err := w.Close(); err != nil {
				log.Println("writePump|io.WriterCloser.Close() error!", err)
				return
			}
		case <-ticker.C:
			wswrap.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := wswrap.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		} // end for select
	} // end of for
}

// https://github.com/gorilla/websocket
func ServeWebSocket(rmgr *room.AVRoomMgr, w http.ResponseWriter, r *http.Request) {
	wsConnection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ServeWebSocket:", err)
		return
	}

	wsAddr, ok := wsConnection.RemoteAddr().(*net.TCPAddr)
	if !ok {
		log.Printf("Error WebSocket request from: ", wsAddr.String())
		return
	}

	ghost := &WSConnWrap{conn: wsConnection, send: make(chan []byte, 256)}
	go ghost.writePump()
	ghost.readPump()
}
