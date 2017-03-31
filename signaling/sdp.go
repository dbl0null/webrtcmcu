package signaling

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// HostType
type HostType int

const (
	HOST HostType = iota
	SRFLX
	PRFLX
	RELAY
)

// MediaType
type MediaType int

const (
	VIDEO_TYPE MediaType = iota
	AUDIO_TYPE
	OTHER
)

// StreamDirection
type StreamDirection int

const (
	SENDRECV StreamDirection = iota
	SENDONLY
	RECVONLY
)

// Profile
type Profile int

const (
	AVPF Profile = iota
	SAVPF
)

// SDP属性定义
const (
	SdpIdentifier    = "webrtcmcu"
	AttrCand         = "a=candidate:"
	AttrCrypto       = "a=crypto:"
	AttrGroup        = "a=group:"
	AttrVideo        = "m=video"
	AttrAudio        = "m=audio"
	AttrMid          = "a=mid"
	AttrSendrecv     = "a=sendrecv"
	AttrRecvonly     = "a=recvonly"
	AttrSendonly     = "a=sendonly"
	AttrIceUser      = "a=ice-ufrag"
	AttrIcePass      = "a=ice-pwd"
	AttrSsrctag      = "a=ssrc"
	AttrSsrcGrouptag = "a=ssrc-group"
	AttrSavpf        = "SAVPF"
	AttrRtpmap       = "a=rtpmap:"
	AttrRtcpmux      = "a=rtcp-mux"
	AttrFp           = "a=fingerprint"
	AttrExtmap       = "a=extmap:"
	AttrRtcpfb       = "a=rtcp-fb:"
	AttrFmtp         = "a=fmtp:"
	AttrBas          = "b=AS:"
	AttrAssociatedPt = "apt"
)

type CryptoInfo struct {
	// * tag number
	tag int

	// * The cipher suite. Only AES_CM_128_HMAC_SHA1_80 is supported as of now.
	cipherSuite string

	// * The key
	keyParams string

	// * The MediaType
	mediaType MediaType
}

type CandidateInfo struct {
	isBundle      bool
	tag           int32
	priority      uint32
	componentID   uint32
	foundation    string
	hostAddress   string
	rAddress      string
	hostPort      uint32
	rPort         uint32
	netProtocol   string
	hostType      HostType
	transProtocol string
	username      string
	password      string
	mediaType     MediaType
}

type BundleTag struct {
	id        string
	mediaType MediaType
}

type RtpMap struct {
	payloadType      uint32
	encodingName     string
	clockRate        uint32
	mediaType        MediaType
	channels         uint32
	feedbackTypes    []string
	formatParameters map[string]string
}

type ExtMap struct {
	value      uint32
	uri        string
	direction  int32
	parameters string
	mediaType  MediaType
}

// https://www.ietf.org/rfc/rfc2327.txt
type SdpInfo struct {

	//lv1-BUNDLE属性相关
	isBundle   bool
	bundleTags []BundleTag

	//lv2--音频和视频属性标识
	hasAudio bool
	hasVideo bool

	//lv3-----ICE Candidate相关属性
	candidateVector []CandidateInfo

	//lv3-----ICE Parameters
	iceVideoUsername string
	iceAudioUsername string
	iceVideoPassword string
	iceAudioPassword string

	isRtcpMux     bool
	isFingerprint bool

	// * RTP Profile type
	profile int

	// * The audio and video SSRCs for this particular SDP.
	audioSsrc, videoSsrc, videoRtxSsrc uint64

	// * MLines for video and audio
	videoCodecs    int
	audioCodecs    int
	videoSdpMLine  int
	audioSdpMLine  int
	videoBandwidth uint

	//其他内部变量
	// * DTLS Fingerprint
	fingerprint string

	// * Mapping from internal PT (key) to external PT (value)
	inOutPTMap map[uint]uint

	// * Mapping from external PT (key) to intermal PT (value)
	outInPTMap map[uint]uint

	// * The negotiated payload list
	payloadVector []RtpMap

	extMapVector []ExtMap
	//---------------------
	cryptoVector []CryptoInfo
}

func (si SdpInfo) initWithSdp(sdp string, media string) {
	si.processSdp(sdp, media)
}

//关于SDP的说明可以参考这里 -->  https://webrtchacks.com/sdp-anatomy/
func (si SdpInfo) processSdp(sdp string, media string) {
	//用于记录当前是在处理哪个部分的属性，video？audio？还是其他
	mtype := MediaType(OTHER)
	media = strings.ToLower(media)

	switch media {
	case "audio":
		mtype = MediaType(AUDIO_TYPE)
	case "video":
		mtype = MediaType(VIDEO_TYPE)
	default:
		mtype = MediaType(OTHER)
	}

	//作为sdp行的索引，从0开始
	mlineNum := -1
	sdpScanner := bufio.NewScanner(strings.NewReader(sdp))
	logger := log.New(os.Stdout, "[sdp]", log.Ldate|log.Ltime|log.Lshortfile)

	for sdpScanner.Scan() {
		//一行一行读，到文件结尾退出. 注意内容有大小写
		line := sdpScanner.Text()

		//注意这里有意按照SDP的结构排列了顺序
		isGroup := strings.Contains(line, AttrGroup)
		isAudio := strings.Contains(line, AttrAudio)
		isVideo := strings.Contains(line, AttrVideo)

		isCand := strings.Contains(line, AttrCand)
		isIceUser := strings.Contains(line, AttrIceUser)
		isIcePass := strings.Contains(line, AttrIcePass)

		isMid := strings.Contains(line, AttrMid)
		isCrypt := strings.Contains(line, AttrCrypto)

		isSsrc := strings.Contains(line, AttrSsrctag)
		// isSsrcGroup := strings.Contains(line, AttrSsrcGrouptag)
		// isSAVPF := strings.Contains(line, AttrSavpf)
		// isRtpmap := strings.Contains(line, AttrRtpmap)
		// isRtcpMuxchar := strings.Contains(line, AttrRtcpmux)
		// isSendRecv := strings.Contains(line, AttrSendrecv)
		// isRecvOnly := strings.Contains(line, AttrRecvonly)
		// isSendOnly := strings.Contains(line, AttrSendonly)
		// isFP := strings.Contains(line, AttrFp)
		// isExtMap := strings.Contains(line, AttrExtmap)
		// isFeedback := strings.Contains(line, AttrRtcpfb)
		// isFmtp := strings.Contains(line, AttrFmtp)
		// isBandwidth := strings.Contains(line, AttrBas)

		logger.Printf("current line -> %s", line)

		// 具体含义参考这个rfc --> https://tools.ietf.org/html/draft-ietf-mmusic-sdp-bundle-negotiation-36
		// 大致用途是，在同一个RTP会话中合并多个媒体流(音频或视频)，例如，a=group:BUNDLE audio video是指，本行下面带有
		// a=mid:audio 和 a=mid:video 两种媒体流合并在一个RTP会话中传输
		if isGroup {
			parts := strings.Split(strings.Split(line, ":")[1], " ")

			if parts[0] == "BUNDLE" {
				si.isBundle = true
			}

			//看bundle后面的属性
			if len(parts) >= 2 {
				for tagno := 1; tagno < len(parts); tagno++ { //tagno := 1, 忽略bundle标识
					theTag := BundleTag{id: parts[tagno], mediaType: MediaType(OTHER)} //MediaType初始化为Other，下面a=mid标签会补充这里
					si.bundleTags = append(si.bundleTags, theTag)
				}
			}
		} //end of group

		// eg: m=audio 9 UDP/TLS/RTP/SAVPF 111 103 104 9 0 8 126
		if isAudio {
			mlineNum++
			si.audioSdpMLine = mlineNum
			//表示进入音频相关的配置处理的部分
			mtype = MediaType(AUDIO_TYPE)
			si.hasAudio = true
		} //end of audio

		// eg: m=video 9 UDP/TLS/RTP/SAVPF 107 99
		if isVideo {
			mlineNum++
			si.videoSdpMLine = mlineNum
			mtype = MediaType(VIDEO_TYPE) //这一步是缘何？
			si.hasVideo = true
		} //end of isVideo

		// "a=candidate:1467250027 1 udp 2122260223 192.168.0.196 46243 typ host generation 0"
		if isCand {
			parts := strings.Split(strings.Split(line, ":")[1], " ")
			si.processCandidate(parts, mtype)
		} //end of isCand

		// "a=ice-ufrag:Oyef7uvBlwafI3hT"
		if isIceUser {
			parts := strings.Split(line, ":")

			if mtype == VIDEO_TYPE {
				si.iceVideoUsername = parts[1]
			} else if mtype == AUDIO_TYPE {
				si.iceAudioUsername = parts[1]
			} else {
				si.iceVideoUsername = parts[1]
				log.Println("Unknown media type for ICE credentials, looks like Firefox, set VideoUsername as default")
			}
		} //end of isUser

		// "a=ice-pwd:T0teqPLNQQOf+5W+ls+P2p16"
		if isIcePass {
			parts := strings.Split(line, ":")
			if mtype == VIDEO_TYPE {
				si.iceVideoPassword = parts[1]
			} else if mtype == AUDIO_TYPE {
				si.iceAudioPassword = parts[1]
			} else {
				si.iceVideoPassword = parts[1]
				log.Println("Unknown media type for ICE credentials, looks like Firefox, set VideoPassword as default")
			}
		}

		//eg: a=mid:audio或a=mid:video
		if isMid {
			parts := strings.Split(strings.Split(line, ":")[1], " ")

			if len(parts) >= 1 && si.isBundle {
				thisID := parts[0]
				//和之前group=BUNDLE里面设定的标签要一致
				for i := 0; i < len(si.bundleTags); i++ {
					if si.bundleTags[i].id == thisID {
						si.bundleTags[i].mediaType = mtype
					}
				}
			} else {
				log.Printf("Unexpected size of a=mid element:%s, len(parts)=%d, isBundle?%t", line, len(parts), si.isBundle)
			}
		} // end of isMid

		// "a=crypto:" 真实场景下暂时没见到这个属性
		if isCrypt {
			var crypto CryptoInfo
			cryptopiece := strings.Split(strings.Split(line, ":")[1], " ")

			// FIXME: add error checking here
			crypto.cipherSuite = cryptopiece[1]
			crypto.keyParams = cryptopiece[3]
			crypto.mediaType = mtype
			si.cryptoVector = append(si.cryptoVector, crypto)
		} //end of isCrypt

		//a=ssrc:3570614608 cname:4TOk42mSjXCkVIa6
		//a=ssrc:3570614608 msid:lgsCFqt9kN2fVKw5wg3NKqGdATQoltEwOdMS 35429d94-5637-4686-9ecd-7d0622261ce8
		//a=ssrc:3570614608 mslabel:lgsCFqt9kN2fVKw5wg3NKqGdATQoltEwOdMS
		//a=ssrc:3570614608 label:35429d94-5637-4686-9ecd-7d0622261ce8
		if isSsrc {
			parts := strings.Split(strings.Split(line, ":")[1], " ")
			if mtype == VIDEO_TYPE && si.videoSsrc == 0 {
				ssrc, err := strconv.ParseUint(parts[0], 10, 32)
				if err == nil {
					log.Printf("Unexpected ssrc(%s), oringin info=%v", parts[0], parts)
				}
				si.videoSsrc = ssrc
			} else if mtype == AUDIO_TYPE && si.audioSsrc == 0 {
				ssrc, err := strconv.ParseUint(parts[0], 10, 32)
				if err == nil {
					log.Printf("Unexpected ssrc(%s), oringin info=%v", parts[0], parts)
				}
				si.audioSsrc = ssrc
			} else {
				log.Printf("Unexpected condition in SSRC, mtype=%d, si.videoSsrc=%d, si.audioSsrc=%d", mtype, si.videoSsrc, si.audioSsrc)
			}
		}

	} //end of for

}

//eg:  "a=candidate:1467250027 1 udp 2122260223 192.168.0.196 46243 typ host generation 0"
//eg: "a=candidate:1367696781 1 udp 33562367 138. 49462 typ relay raddr 138.4 rport 53531 generation 0"
func (si SdpInfo) processCandidate(parts []string, mtype MediaType) bool {
	var cand CandidateInfo
	var typesStr = []string{"host", "srflx", "prflx", "relay"}
	cand.mediaType = mtype
	cand.foundation = parts[0] //这个数字干啥用的？
	compID, err := strconv.ParseUint(parts[1], 10, 32)
	if err == nil { // componentID=0代表rtp，=1代表rtcp
		log.Printf("Unexpected ComponentID(%s), oringin info=%v", parts[1], parts)
	}
	cand.componentID = uint32(compID)

	cand.netProtocol = parts[2] // tcp or udp

	//暂时不支持TCP candidate
	if strings.ToLower(cand.netProtocol) != "udp" {
		return false
	}

	if priority, err := strconv.ParseUint(parts[3], 10, 32); err == nil {
		cand.priority = uint32(priority)
	} else {
		log.Printf("Unexpected priority(%s), oringin info=%v", parts[3], parts)
	}

	cand.hostAddress = parts[4]

	if hostport, err := strconv.ParseUint(parts[5], 10, 32); err == nil {
		cand.hostPort = uint32(hostport)
	} else {
		log.Printf("Unexpected HostPort(%s), oringin info=%v", parts[5], parts)
	}

	if parts[6] != "typ" {
		return false
	}

	candType := uint32(0)
	for i := uint32(0); i < 4; i++ {
		if parts[7] == typesStr[i] {
			candType = i
		}
	}

	// mcu嘛，默认要走中转方式
	switch candType {
	case 0:
		cand.hostType = HOST
	case 1:
		cand.hostType = SRFLX
	case 2:
		cand.hostType = PRFLX
	case 3:
		cand.hostType = RELAY
	default:
		cand.hostType = RELAY
	}

	if cand.hostType == SRFLX || cand.hostType == RELAY {
		cand.rAddress = parts[9]
		rport, err := strconv.ParseUint(parts[11], 10, 32)
		if err != nil {
			log.Printf("Unexpected rport(%s), oringin info=%v", parts[10], parts)
		}
		cand.rPort = uint32(rport)
	}

	si.candidateVector = append(si.candidateVector, cand)
	return true
}
