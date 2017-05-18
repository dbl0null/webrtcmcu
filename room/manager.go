package room

import (
	"fmt"
	"log"
)

const (
	CmdEnterRoom = iota + 1
	CmdLeaveRoom
	CmdGetUserInfo
	CmdModifyUserInfo
)

type AVRoomMgr struct {
	//房间列表
	rooms map[string]*AVRoom

	//端口池子，每个用户需要分一个端口
	portPool map[uint32]bool

	//用来统一收集针对房间的操作请求
	actionReqChan chan RoomActionMsg
}

type RoomActionMsg struct {
	UserInfo      AVUser
	ActionCmd     int
	ReturnCode    int
	Content       interface{}
	ActionErr     error
	ActionResChan chan RoomActionMsg
}

type RoomServiceErr string

func (rserr RoomServiceErr) Error() string {
	return string(rserr)
}

//定义一些错误码
const (
	RET_CODE_SUCC           = 0
	RET_CODE_ALLOC_PORT_ERR = -iota
)

// NewRoomMgr 用于创建一个房间管理器
// 在这个简单的程序中，只需要有一个AVRoomMgr单例即可
func NewRoomMgr() *AVRoomMgr {
	var roomgr AVRoomMgr
	roomgr.rooms = make(map[string]*AVRoom)
	roomgr.portPool = make(map[uint32]bool)
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
		case actMsg := <-roomgr.actionReqChan:
			log.Printf("AVUser action msg: %v", actMsg)
			user := actMsg.UserInfo

			//开始按命令字处理
			switch actMsg.ActionCmd {
			case CmdEnterRoom:
				log.Printf("roomgr.RoomService().CmdEnterRoom, AVUser:%s enter room(%s)", user.UserID, user.RoomID)
				roomInfo := roomgr.rooms[user.RoomID]

				allocPort := roomgr.allecDataChannelPort()

				//如果Port不够了就返回错误，不加入房间了
				if allocPort == -1 {
					//给User的goroutine回复处理结果
					go func(msg *RoomActionMsg) {
						msg.ActionResChan <- RoomActionMsg{
							UserInfo:   msg.UserInfo,
							ActionCmd:  msg.ActionCmd,
							ReturnCode: RET_CODE_ALLOC_PORT_ERR,
							ActionErr:  RoomServiceErr(fmt.Sprintf("Run out of port resource!")),
						}
					}(&actMsg)
					continue
				}

				//房间不存在就创建一个
				if roomInfo == nil {
					roomgr.rooms[user.RoomID] = &AVRoom{
						RoomID:       user.RoomID,
						Participants: make(map[string]*AVUser),
					}
					roomInfo = roomgr.rooms[user.RoomID]
					roomInfo.Participants[user.UserID] = &user
				}

				//给User的goroutine回复处理结果
				go func(msg *RoomActionMsg) {
					msg.ActionResChan <- RoomActionMsg{
						UserInfo:   msg.UserInfo,
						ActionCmd:  msg.ActionCmd,
						ReturnCode: RET_CODE_SUCC,
						Content:    allocPort,
					}
				}(&actMsg)
			case CmdLeaveRoom:
				log.Printf("AVUser:%s leave room(%s)", user.UserID, user.RoomID)

				//回收分配出去的端口
				if roomgr.portPool[user.UserDataChanPort] {
					roomgr.portPool[user.UserDataChanPort] = false
				}

				roomInfo := roomgr.rooms[user.RoomID]
				if roomInfo == nil {
					log.Printf("AVUser:%s leave room(%s), room not exist!", user.UserID, user.RoomID)
					continue
				}

				userInRoom := roomInfo.Participants[user.UserID]
				if userInRoom == nil {
					log.Printf("AVUser:%s leave room(%s), user not exist!", user.UserID, user.RoomID)
					continue
				}

				//从用户列表里删掉该用户，用户信息进房间检查过，这里不用重复搞
				delete(roomInfo.Participants, user.UserID)

				//房间里所有用户都退了，就把房间删除
				if len(roomInfo.Participants) == 0 {
					delete(roomgr.rooms, user.RoomID)
				}
				//删除用户不用给调用方返回了
			case CmdGetUserInfo:
			case CmdModifyUserInfo:
			default:
				log.Printf("room.mgr.RoomService() Unknown cmd:%d", actMsg.ActionCmd)
			} // end of switch
		} // end of select
	} // end of for
}

func (roomgr *AVRoomMgr) allecDataChannelPort() int {
	retPort := -1
	for port, used := range roomgr.portPool {
		if !used {
			roomgr.portPool[port] = true
			retPort = int(port)
			break
		}
	}

	return retPort
}

//EnterRoom 进房间成功返回分配的数据端口
func (roomgr *AVRoomMgr) EnterRoom(userInfo AVUser) (int, error) {
	if len(userInfo.UserID) == 0 || len(userInfo.RoomID) == 0 {
		return 0, RoomServiceErr(fmt.Sprintf("Enter room error: content=%#v", userInfo))
	}

	//插入到请求队列
	req := RoomActionMsg{
		UserInfo:      userInfo,
		ActionCmd:     CmdEnterRoom,
		ActionResChan: make(chan RoomActionMsg),
	}

	roomgr.actionReqChan <- req

	res := <-req.ActionResChan
	if res.ReturnCode != RET_CODE_SUCC {
		return 0, RoomServiceErr(fmt.Sprintf("Enter room error(%d)", res.ReturnCode))
	}

	return res.Content.(int), nil
}

// 接收请求的入口，这里先做一些检查
func (roomgr *AVRoomMgr) RecvReqMsg(msg RoomActionMsg) error {
	//检查一下有没有异常
	switch msg.ActionCmd {
	case CmdEnterRoom:
		if len(msg.UserInfo.UserID) == 0 || len(msg.UserInfo.RoomID) == 0 || msg.ActionResChan == nil {
			return RoomServiceErr(fmt.Sprintf("Enter room error: content=%#v", msg))
		}
	case CmdLeaveRoom:
		if len(msg.UserInfo.UserID) == 0 || len(msg.UserInfo.RoomID) == 0 || msg.UserInfo.UserDataChanPort == 0 {
			return RoomServiceErr(fmt.Sprintf("Leave room error: content=%#v", msg))
		}
	default:
		if len(msg.UserInfo.UserID) == 0 || len(msg.UserInfo.RoomID) == 0 {
			return RoomServiceErr(fmt.Sprintf("RoomActionReq error: content=%#v", msg))
		}
	}

	//插入到请求队列
	roomgr.actionReqChan <- msg
	return nil
}
