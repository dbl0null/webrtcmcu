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
	DataChanPort int    `json:"datachanport"`
}

// WSConnWrap 伪装成Browser的PeerConnection对端
type WSConnWrap struct {
	// 真实WebSocket连接的指针
	signalConn *websocket.Conn

	// 与终端之间的数据发送和接收队列
	dataSendQueue chan []byte
	dataRecvQueue chan []byte

	// 指向UserInfo的指针
	userInfo *room.AVUser

	// 房间管理器指针
	roomgr *room.AVRoomMgr

	// 客户端IP和端口
	clientAddr *net.TCPAddr
}

func (wswrap *WSConnWrap) msgHandler(recognizer webrtcRecognizer, jsonMessage []byte) {
	switch recognizer.Type {
	case "offer":
		reqMsg := messageSDP{}
		//根据消息类型，解析不同的消息结构
		jsonErr := json.Unmarshal(jsonMessage, &reqMsg)
		if jsonErr != nil {
			log.Printf("msgHandler| Can't parse msg content. %v", jsonErr)
			return
		}

		var offer SdpInfo
		offer.InitWithSdp(reqMsg.Sdp, "")

		//值拷贝，offer还要保留
		answer := offer

		//@TODO H264和Opus要排在前面
		reqMsg.Sdp = answer.CreateSdp()
		reqMsg.Type = "answer"

		//生成一个新的Offer消息返回给浏览器
		jsonBytes, jsonErr := json.Marshal(reqMsg)
		if jsonErr != nil {
			log.Printf("msgHandler| Can't create anser msg. %v", jsonErr)
			return
		}
		wswrap.dataSendQueue <- jsonBytes

	case "candidate":
		reqMsg := messageSDP{}
		//根据消息类型，解析不同的消息结构
		jsonErr := json.Unmarshal(jsonMessage, &reqMsg)
		if jsonErr != nil {
			log.Printf("msgHandler| Can't parse msg content. %v", jsonErr)
		}
		/*
					type: 'candidate',
			      label: event.candidate.sdpMLineIndex,
			      id: event.candidate.sdpMid,
			      candidate: event.candidate.candidate,
			      userid : userIDVal,
			      roomid : roomIDVal
		*/
		log.Println("msgHandler| receive candidate:")

	case "enteroom":
		reqMsg := messageEnteroom{}
		//根据消息类型，解析不同的消息结构
		jsonErr := json.Unmarshal(jsonMessage, &reqMsg)
		if jsonErr != nil {
			log.Printf("msgHandler| Can't parse enteroom msg. err=%v, msg=%v", jsonErr, jsonMessage)
			answerMsg := messageEnteroom{
				Type:  "enteroomres",
				Error: "enteroom msg parse err:" + jsonErr.Error(),
			}
			jsonBytes, jsonErr := json.Marshal(answerMsg)
			if jsonErr != nil {
				log.Printf("msgHandler| Can't create anser msg-1. %v", jsonErr)
				return
			}
			wswrap.dataSendQueue <- jsonBytes
			return
		}

		log.Printf("msgHandler|Enteroom. reqMsg=%v", reqMsg)

		userInfo := room.AVUser{
			RoomID:         reqMsg.RoomID,
			UserID:         reqMsg.UserID,
			UserSignalAddr: wswrap.signalConn.RemoteAddr().(*net.TCPAddr),
		}

		//到房间管理器加入房间（房间不存在就创建一个）
		allocPort, enterErr := wswrap.roomgr.EnterRoom(userInfo)

		//进房间失败？？
		if enterErr != nil {
			log.Printf("msgHandler| Enter room err(%v), msg=%v", enterErr, jsonMessage)
			answerMsg := messageEnteroom{
				Type:  "enteroomres",
				Error: "enteroom msg parse err:" + enterErr.Error(),
			}
			jsonBytes, jsonErr := json.Marshal(answerMsg)
			if jsonErr != nil {
				log.Printf("msgHandler| Can't create anser msg-2. %v", jsonErr)
				return
			}
			wswrap.dataSendQueue <- jsonBytes
			return
		}

		log.Printf("msgHandler|Enteroom. allocPort=%d", allocPort)

		//进房间成功？给客户端回包
		answerMsg := messageEnteroom{
			RoomID:       reqMsg.RoomID,
			UserID:       reqMsg.UserID,
			DataChanPort: allocPort,
			Type:         "enteroomres",
		}

		jsonBytes, jsonErr := json.Marshal(answerMsg)
		if jsonErr != nil {
			log.Printf("msgHandler| Can't create anser msg-3. %v", jsonErr)
			return
		}

		wswrap.dataSendQueue <- jsonBytes
		log.Printf("msgHandler|Enteroom. answer=%v", answerMsg)

		//@TODO send msg to other participants
	case "leaveroom":
		log.Println("msgHandler| leaveroom, 页面直接关闭就退房了，暂时不搞挂断了")
	default:
		log.Println("msgHandler| Unknown msgtype:")
	}
}

//msgLooper 循环处理接收队列里的数据
func (wswrap *WSConnWrap) msgLooper() {
	for jsonMessage := range wswrap.dataRecvQueue {
		recognizer := webrtcRecognizer{}
		jsonErr := json.Unmarshal(jsonMessage, &recognizer)
		if jsonErr != nil {
			log.Printf("msgLooper| Can't recognize msg type. %v", jsonErr)
			continue
		}
		wswrap.msgHandler(recognizer, jsonMessage)
	}
}

//从头Browser->WSClient通道读数据
func (wswrap *WSConnWrap) readPump() {
	defer func() {
		wswrap.signalConn.Close()
		close(wswrap.dataRecvQueue)
		//TODO 删除用户信息
	}()
	wswrap.signalConn.SetReadLimit(maxMessageSize)
	wswrap.signalConn.SetReadDeadline(time.Now().Add(pongWait))
	wswrap.signalConn.SetPongHandler(func(string) error { wswrap.signalConn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		messageType, message, err := wswrap.signalConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("readPump|Connection lost for userinfo=%v: err=%v", wswrap.userInfo, err)
			} else {
				log.Printf("readPump|Connection(userinfo=%v) Error: %v", wswrap.userInfo, err)
			}
			return
		}

		if messageType == websocket.TextMessage {
			wswrap.dataRecvQueue <- message
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
		wswrap.signalConn.Close()
		close(wswrap.dataSendQueue)
	}()
	for {
		select {
		case message, ok := <-wswrap.dataSendQueue:
			wswrap.signalConn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				//后台强行关闭连接
				wswrap.signalConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := wswrap.signalConn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 把队列里积攒的待发送数据一起往外发
			n := len(wswrap.dataSendQueue)
			for i := 0; i < n; i++ {
				w.Write(<-wswrap.dataSendQueue)
			}

			//Close()是为了把数据刷到底层发送出去
			if err := w.Close(); err != nil {
				log.Println("writePump|io.WriterCloser.Close() error!", err)
				return
			}
		case <-ticker.C:
			wswrap.signalConn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := wswrap.signalConn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		} // end for select
	} // end of for
}

// https://github.com/gorilla/websocket
func ServeWebSocket(rmgr *room.AVRoomMgr, w http.ResponseWriter, r *http.Request) {
	log.Printf("ServeWebSocket(), req=%v\n", r.URL)
	wsConnection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ServeWebSocket:", err)
		return
	}

	wsAddr, ok := wsConnection.RemoteAddr().(*net.TCPAddr)
	if !ok {
		log.Printf("Error WebSocket request from: %s", wsAddr.String())
		return
	}

	log.Printf("ServeWebSocket() clientAddr=%v", wsAddr)

	wswrap := &WSConnWrap{
		signalConn:    wsConnection,
		dataSendQueue: make(chan []byte, 612),
		dataRecvQueue: make(chan []byte, 612),
		roomgr:        rmgr,
		clientAddr:    wsAddr,
	}
	go wswrap.writePump()
	go wswrap.msgLooper()
	log.Printf("ServeWebSocket() wsConn=%v", wswrap)
	wswrap.readPump()
}
