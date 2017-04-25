package room

import (
	"log"
	"net"
)

type AVRoomMgr struct {
	//房间列表
	rooms map[string]*AVRoom

	//端口池子，每个用户需要分一个端口
	portPool map[uint32]bool

	enterRoomChan chan AVUser
	leaveRoomChan chan AVUser
	actionReqChan chan<- RoomActionMsg
}

type AVRoom struct {
	RoomID       string
	Participants *map[string]*AVUser
	//Participants *map[UserChanID]*AVUser
}

type AVUser struct {
	RoomID          string
	UserID          string
	UserClientAddr  net.UDPAddr
	UserAssiginPort uint32
	ActionResChan   <-chan RoomActionMsg
	OfferSDP        string
	AnswerSDP       string
	CandidateSDP    string
}

type RoomActionMsg struct {
	RoomID     string
	UserID     string
	ActionCmd  int
	ReturnCode int
	Content    string
	ActionErr  error
}

// NewRoomMgr 用于创建一个房间管理器
// 在这个简单的程序中，只需要有一个AVRoomMgr单例即可
func NewRoomMgr() *AVRoomMgr {
	var roomgr AVRoomMgr
	roomgr.rooms = make(map[string]*AVRoom)
	roomgr.portPool = make(map[uint32]bool)
	roomgr.enterRoomChan = make(chan AVUser, 612)
	roomgr.leaveRoomChan = make(chan AVUser, 612)
	roomgr.actionReqChan = make(chan RoomActionMsg, 612)

	//保留10000个端口供后续使用
	for i := 50000; i < 60000; i++ {
		roomgr.portPool[uint32(i)] = false
	}

	return &roomgr
}

func (roomgr *AVRoomMgr) RoomService() {
	for {
		select {
		case avuser := <-roomgr.enterRoomChan:
			log.Printf("AVUser:%s enter room(%s)", avuser.UserID, avuser.RoomID)
		case avuser := <-roomgr.leaveRoomChan:
			log.Printf("AVUser:%s leave room(%s)", avuser.UserID, avuser.RoomID)
		case actMsg := <-roomgr.actionReqChan:
			log.Printf("AVUser:%s leave room(%s)", avuser.UserID, avuser.RoomID)
		}
	}
}
