package room

import "net"

type AVRoom struct {
	RoomID       string
	Participants map[string]*AVUser
	//Participants *map[UserChanID]*AVUser
}

type AVUser struct {
	RoomID           string
	UserID           string
	UserSignalAddr   *net.TCPAddr
	UserDataChanPort uint32
	OfferSDP         string
	AnswerSDP        string
	CandidateSDP     string
}
