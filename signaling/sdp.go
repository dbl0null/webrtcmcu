package signaling

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"
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
	AVPF Profile = iota // webrtc已不支持裸rtp，都要加密
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
	priority      uint64
	componentID   uint64
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
	payloadType      uint64
	encodingName     string
	clockRate        uint64
	mediaType        MediaType
	channels         uint64
	feedbackTypes    []string
	formatParameters map[string]string
}

type ExtMap struct {
	value      uint64
	uri        string
	direction  StreamDirection
	parameters string
	mediaType  MediaType
}

// https://www.ietf.org/rfc/rfc2327.txt
type SdpInfo struct {

	//sdp原始内容
	sdpRawContent string

	//lv1-BUNDLE属性相关
	isBundle   bool
	bundleTags []BundleTag

	//lv2--音频和视频属性标识
	hasAudio bool
	hasVideo bool
	// MLines for video and audio
	videoSdpMLine int
	audioSdpMLine int

	//lv2--RTP Profile type ==> m=xxx/SAVPF，但为啥只有一个变量？会有覆盖隐患
	profile Profile

	//lv3-----ICE Candidate相关属性，audio和video都有
	candidateVector []*CandidateInfo

	//lv3-----ICE Parameters
	iceVideoUsername string
	iceAudioUsername string
	iceVideoPassword string
	iceAudioPassword string

	//lv3-----DTLS Fingerprint，audio和video都有，但为啥只有一个变量？@FIXME
	fingerprint   string
	isFingerprint bool

	//lv3-----Codec Parameter，音视频混在一起，靠value来区分
	extMapVector     []ExtMap
	payloadParsedMap map[uint64]*RtpMap

	videoDirection, audioDirection     StreamDirection
	isRtcpMux                          bool
	audioSsrc, videoSsrc, videoRtxSsrc uint64
	videoBandwidth                     uint64
	cryptoVector                       []*CryptoInfo

	//=============================
	// 记录codec有多少个
	videoCodecs int
	audioCodecs int
}

func (si *SdpInfo) CreateSdp() string {
	var strBuf bytes.Buffer
	//代表一次会话的ID，随机字符串
	msidtemp := si.generateRandom(10)
	strBuf.WriteString("v=0\n")
	strBuf.WriteString("o=- 0 0 IN IP4 127.0.0.1\n")
	strBuf.WriteString(fmt.Sprintf("s=%s\n", SdpIdentifier))
	strBuf.WriteString("t=0 0\n")

	if si.isBundle {
		strBuf.WriteString("a=group:BUNDLE")
		for i := 0; i < len(si.bundleTags); i++ {
			strBuf.WriteString(fmt.Sprintf(" %s", si.bundleTags[i].id))
		}
		strBuf.WriteString("\n")
		strBuf.WriteString(fmt.Sprintf("a=msid-semantic: WMS %s\n", msidtemp))
	}

	if si.hasAudio {
		strBuf.WriteString("m=audio 1")
		if si.profile == SAVPF {
			strBuf.WriteString(" UDP/TLS/RTP/SAVPF ")
		} else {
			strBuf.WriteString(" RTP/AVPF ")
		}

		codecCounter := 0
		for _, v := range si.payloadParsedMap {
			if v.mediaType == AUDIO_TYPE {
				codecCounter++
				strBuf.WriteString(fmt.Sprintf("%d", v.payloadType))
				if codecCounter < si.audioCodecs {
					strBuf.WriteString(" ")
				}
			}
		} //end of for

		strBuf.WriteString("\nc=IN IP4 0.0.0.0\n")

		if si.isRtcpMux {
			strBuf.WriteString("a=rtcp:1 IN IP4 0.0.0.0\n")
		}

		//audio里面就包含了所有candidate，video里面却只有一部分。。。
		for i := 0; i < len(si.candidateVector); i++ {
			if si.candidateVector[i].mediaType == AUDIO_TYPE || si.isBundle {
				strBuf.WriteString(si.StringfyCandidate(si.candidateVector[i]))
				strBuf.WriteString("\n")
			}
		}

		if len(si.iceAudioUsername) > 0 {
			strBuf.WriteString(fmt.Sprintln("a=ice-ufrag:", si.iceAudioUsername))
			strBuf.WriteString(fmt.Sprintln("a=ice-pwd:", si.iceAudioPassword))
		} else {
			strBuf.WriteString(fmt.Sprintln("a=ice-ufrag:", si.iceVideoUsername))
			strBuf.WriteString(fmt.Sprintln("a=ice-pwd:", si.iceVideoPassword))
		}

		if si.isFingerprint {
			strBuf.WriteString(fmt.Sprintln("a=fingerprint:sha-256", si.fingerprint))
		}

		switch si.audioDirection {
		case SENDONLY:
			strBuf.WriteString("a=sendonly\n")
		case SENDRECV:
			strBuf.WriteString("a=sendrecv\n")
		case RECVONLY:
			strBuf.WriteString("a=recvonly\n")
		}

		for i := 0; i < len(si.extMapVector); i++ {
			if si.extMapVector[i].mediaType == AUDIO_TYPE {
				strBuf.WriteString(fmt.Sprintf("a=extmap:%d %s\n", si.extMapVector[i].value, si.extMapVector[i].uri))
			}
		}

		if len(si.bundleTags) > 2 {
			log.Printf("Too much bundleTags(%d) than support", len(si.bundleTags))
		}

		for i := 0; i < len(si.bundleTags); i++ {
			if si.bundleTags[i].mediaType == AUDIO_TYPE {
				strBuf.WriteString(fmt.Sprintf("a=mid:%s\n", si.bundleTags[i].id))
			}
		}

		if si.isRtcpMux {
			strBuf.WriteString("a=rtcp-mux\n")
		}

		for i := 0; i < len(si.cryptoVector); i++ {
			if si.cryptoVector[i].mediaType == AUDIO_TYPE {
				strBuf.WriteString(fmt.Sprintf("a=crypto:%d %s inline:%s\n", si.cryptoVector[i].tag, si.cryptoVector[i].cipherSuite, si.cryptoVector[i].keyParams))
			}
		}

		for _, aRtpMap := range si.payloadParsedMap {
			if aRtpMap.mediaType == AUDIO_TYPE {
				if aRtpMap.channels > 1 {
					strBuf.WriteString(fmt.Sprintf("a=rtpmap:%d %s/%d/%d\n", aRtpMap.payloadType, aRtpMap.encodingName, aRtpMap.clockRate, aRtpMap.channels))
				} else {
					strBuf.WriteString(fmt.Sprintf("a=rtpmap:%d %s/%d\n", aRtpMap.payloadType, aRtpMap.encodingName, aRtpMap.clockRate))
				}

				if len(aRtpMap.feedbackTypes) > 0 {
					for i := 0; i < len(aRtpMap.feedbackTypes); i++ {
						strBuf.WriteString(fmt.Sprintf("a=rtcp-fb:%d %s\n", aRtpMap.payloadType, aRtpMap.feedbackTypes[i]))
					}
				}

				for fmtKey, fmtVal := range aRtpMap.formatParameters {
					if fmtKey != "none" {
						strBuf.WriteString(fmt.Sprintf("a=fmtp:%d %s=%s\n", aRtpMap.payloadType, fmtKey, fmtVal))
					} else {
						strBuf.WriteString(fmt.Sprintf("a=fmtp:%d %s\n", aRtpMap.payloadType, fmtVal))
					}
				}
			}
		} //end of for

		if si.audioSsrc == 0 {
			si.audioSsrc = 44444
		}

		strBuf.WriteString("a=maxptime:60\n")
		strBuf.WriteString(fmt.Sprintf("a=ssrc:%d cname:o/i14u9pJrxRKAsu\n", si.audioSsrc))
		strBuf.WriteString(fmt.Sprintf("a=ssrc:%d msid:%s a0\n", si.audioSsrc, msidtemp))
		strBuf.WriteString(fmt.Sprintf("a=ssrc:%d mslabel:%s\n", si.audioSsrc, msidtemp))
		strBuf.WriteString(fmt.Sprintf("a=ssrc:%d label:a0\n", si.audioSsrc))
	} //end of hasAudio

	if si.hasVideo {
		strBuf.WriteString("m=video 1")
		if si.profile == SAVPF {
			strBuf.WriteString(" UDP/TLS/RTP/SAVPF ")
		} else {
			strBuf.WriteString(" RTP/AVPF ")
		}

		codecCounter := 0
		for _, v := range si.payloadParsedMap {
			if v.mediaType == VIDEO_TYPE {
				codecCounter++
				strBuf.WriteString(fmt.Sprintf("%d", v.payloadType))
				if codecCounter < si.videoCodecs {
					strBuf.WriteString(" ")
				}
			}
		} //end of for

		strBuf.WriteString("\nc=IN IP4 0.0.0.0\n")

		if si.isRtcpMux {
			strBuf.WriteString("a=rtcp:1 IN IP4 0.0.0.0\n")
		}

		for i := 0; i < len(si.candidateVector); i++ {
			if si.candidateVector[i].mediaType == VIDEO_TYPE {
				strBuf.WriteString(si.StringfyCandidate(si.candidateVector[i]))
				strBuf.WriteString("\n")
			}
		}

		strBuf.WriteString(fmt.Sprintln("a=ice-ufrag:", si.iceVideoUsername))
		strBuf.WriteString(fmt.Sprintln("a=ice-pwd:", si.iceVideoPassword))

		if si.isFingerprint {
			strBuf.WriteString(fmt.Sprintln("a=fingerprint:sha-256", si.fingerprint))
		}

		switch si.videoDirection {
		case SENDONLY:
			strBuf.WriteString("a=sendonly\n")
		case SENDRECV:
			strBuf.WriteString("a=sendrecv\n")
		case RECVONLY:
			strBuf.WriteString("a=recvonly\n")
		}

		for i := 0; i < len(si.extMapVector); i++ {
			if si.extMapVector[i].mediaType == VIDEO_TYPE {
				strBuf.WriteString(fmt.Sprintf("a=extmap:%d %s\n", si.extMapVector[i].value, si.extMapVector[i].uri))
			}
		}

		for i := 0; i < len(si.bundleTags); i++ {
			if si.bundleTags[i].mediaType == VIDEO_TYPE {
				strBuf.WriteString("a=mid:")
				strBuf.WriteString(si.bundleTags[i].id)
				strBuf.WriteString("\n")
			}
		}

		if si.isRtcpMux {
			strBuf.WriteString("a=rtcp-mux\n")
		}

		for i := 0; i < len(si.cryptoVector); i++ {
			if si.cryptoVector[i].mediaType == VIDEO_TYPE {
				strBuf.WriteString(fmt.Sprintf("a=crypto:%d %s inline:%s\n", si.cryptoVector[i].tag, si.cryptoVector[i].cipherSuite, si.cryptoVector[i].keyParams))
			}
		}

		for _, aRtpMap := range si.payloadParsedMap {
			if aRtpMap.mediaType == VIDEO_TYPE {
				strBuf.WriteString(fmt.Sprintf("a=rtpmap:%d %s/%d\n", aRtpMap.payloadType, aRtpMap.encodingName, aRtpMap.clockRate))

				if len(aRtpMap.feedbackTypes) > 0 {
					for i := 0; i < len(aRtpMap.feedbackTypes); i++ {
						strBuf.WriteString(fmt.Sprintf("a=rtcp-fb:%d %s\n", aRtpMap.payloadType, aRtpMap.feedbackTypes[i]))
					}
				}

				for fmtKey, fmtVal := range aRtpMap.formatParameters {
					if fmtKey != "none" {
						strBuf.WriteString(fmt.Sprintf("a=fmtp:%d %s=%s\n", aRtpMap.payloadType, fmtKey, fmtVal))
					} else {
						strBuf.WriteString(fmt.Sprintf("a=fmtp:%d %s\n", aRtpMap.payloadType, fmtVal))
					}
				}
			}
		} //end of for

		if si.videoSsrc == 0 {
			si.videoSsrc = 55543
		}

		//与原始sdp相比，没生成这玩意，不知道行不行~
		//a=ssrc-group:FID 2231627014 632943048
		strBuf.WriteString(fmt.Sprintf("a=ssrc:%d cname:o/i14u9pJrxRKAsu\n", si.videoSsrc))
		strBuf.WriteString(fmt.Sprintf("a=ssrc:%d msid:%s v0\n", si.videoSsrc, msidtemp))
		strBuf.WriteString(fmt.Sprintf("a=ssrc:%d mslabel:%s\n", si.videoSsrc, msidtemp))
		strBuf.WriteString(fmt.Sprintf("a=ssrc:%d label:v0\n", si.videoSsrc))

		if si.videoRtxSsrc != 0 {
			strBuf.WriteString(fmt.Sprintf("a=ssrc:%d cname:o/i14u9pJrxRKAsu\n", si.videoRtxSsrc))
			strBuf.WriteString(fmt.Sprintf("a=ssrc:%d msid:%s v0\n", si.videoRtxSsrc, msidtemp))
			strBuf.WriteString(fmt.Sprintf("a=ssrc:%d mslabel:%s\n", si.videoRtxSsrc, msidtemp))
			strBuf.WriteString(fmt.Sprintf("a=ssrc:%d label:v0\n", si.videoRtxSsrc))
		}
	}

	return strBuf.String()
}

func (si *SdpInfo) generateRandom(randLen int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	rand.Seed(int64(time.Now().Second()))

	var strBuf bytes.Buffer
	for i := 0; i < randLen; i++ {
		strBuf.WriteByte(alphanum[rand.Intn(len(alphanum))])
	}

	return strBuf.String()
}

func (si *SdpInfo) GetCodecByName(codecname string, clockrate uint64) *RtpMap {
	for _, v := range si.payloadParsedMap {
		if v.encodingName == codecname && v.clockRate == clockrate {
			return v
		}
	}
	return nil
}

func (si *SdpInfo) GetUsername(media MediaType) string {
	if media == AUDIO_TYPE {
		return si.iceAudioUsername
	}
	return si.iceVideoUsername
}

func (si *SdpInfo) GetPassword(media MediaType) string {
	if media == AUDIO_TYPE {
		return si.iceAudioPassword
	}
	return si.iceVideoPassword
}

func (si *SdpInfo) SetCredentials(username string, password string, media MediaType) {
	switch media {
	case VIDEO_TYPE:
		si.iceVideoUsername = username
		si.iceVideoPassword = password
	case AUDIO_TYPE:
		si.iceAudioUsername = username
		si.iceAudioPassword = password
	default:
		si.iceVideoUsername = username
		si.iceVideoPassword = password
		si.iceAudioUsername = username
		si.iceAudioPassword = password
	}
}

func (si *SdpInfo) AddCrypto(cinfo *CryptoInfo) {
	si.cryptoVector = append(si.cryptoVector, cinfo)
}

func (si *SdpInfo) StringfyCandidate(candInfo *CandidateInfo) string {
	var strBuf bytes.Buffer

	strBuf.WriteString("a=candidate:")
	strBuf.WriteString(candInfo.foundation)
	strBuf.WriteString(" ")

	strBuf.WriteString(fmt.Sprintf("%d %s %s %d tpy ", candInfo.componentID, candInfo.netProtocol, candInfo.hostAddress, candInfo.hostPort))
	switch candInfo.hostType {
	case HOST:
		strBuf.WriteString("host")
	case SRFLX:
		strBuf.WriteString("srflx")
	case PRFLX:
		strBuf.WriteString("prflx")
	case RELAY:
		strBuf.WriteString("relay")
	default:
		strBuf.WriteString("host")
	}

	if candInfo.hostType == SRFLX || candInfo.hostType == RELAY {
		// raddr 192.168.0.12 rport 50483
		strBuf.WriteString(" raddr ")
		strBuf.WriteString(candInfo.rAddress)
		strBuf.WriteString(" rport ")
		strBuf.WriteString(fmt.Sprintf("%d", candInfo.rPort))
	}

	strBuf.WriteString(" generation 0")
	return strBuf.String()
}

func (si *SdpInfo) AddCandidate(candInfo *CandidateInfo) bool {
	if candInfo != nil {
		si.candidateVector = append(si.candidateVector, candInfo)
		return true
	}
	return false
}

//InitWithSdp 用字符串初始化SDP对象，processCandidate和processSdp是初始化动作的辅助函数
func (si *SdpInfo) InitWithSdp(sdp string, media string) {
	si.audioSdpMLine = -1
	si.videoSdpMLine = -1
	si.profile = SAVPF //现在Chrome都强制加密了
	si.sdpRawContent = sdp
	si.payloadParsedMap = make(map[uint64]*RtpMap)
	si.candidateVector = []*CandidateInfo{}
	si.extMapVector = []ExtMap{}
	si.cryptoVector = []*CryptoInfo{}
	si.processSdp(sdp, media)
}

//关于SDP的说明可以参考这里 -->  https://webrtchacks.com/sdp-anatomy/
func (si *SdpInfo) processSdp(sdp string, media string) bool {
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
	//logger := log.New(os.Stdout, "[sdp]", log.Ldate|log.Ltime|log.Lshortfile)

	for sdpScanner.Scan() {
		//一行一行读，到文件结尾退出. 注意内容有大小写
		line := sdpScanner.Text()

		//注意这里有意按照SDP的结构排列了顺序
		isGroup := strings.Contains(line, AttrGroup)
		isAudio := strings.Contains(line, AttrAudio)
		isVideo := strings.Contains(line, AttrVideo)
		isSAVPF := strings.Contains(line, AttrSavpf)

		isCand := strings.Contains(line, AttrCand)
		isIceUser := strings.Contains(line, AttrIceUser)
		isIcePass := strings.Contains(line, AttrIcePass)

		isFp := strings.Contains(line, AttrFp)
		isMid := strings.Contains(line, AttrMid)
		isCrypt := strings.Contains(line, AttrCrypto)

		isExtMap := strings.Contains(line, AttrExtmap)
		isSendRecv := strings.Contains(line, AttrSendrecv)
		isRecvOnly := strings.Contains(line, AttrRecvonly)
		isSendOnly := strings.Contains(line, AttrSendonly)
		isRtcpMuxchar := strings.Contains(line, AttrRtcpmux)

		isRtpmap := strings.Contains(line, AttrRtpmap)
		isFmtp := strings.Contains(line, AttrFmtp)
		isFeedback := strings.Contains(line, AttrRtcpfb)

		isSsrc := strings.Contains(line, AttrSsrctag)
		isSsrcGroup := strings.Contains(line, AttrSsrcGrouptag)
		isBandwidth := strings.Contains(line, AttrBas)

		//logger.Printf("current line -> %s", line)

		// 具体含义参考这个rfc --> https://tools.ietf.org/html/draft-ietf-mmusic-sdp-bundle-negotiation-36
		// 大致用途是，在同一个RTP会话中合并多个媒体流(音频或视频)
		// "a=group:BUNDLE audio video是指，本行下面带有"
		// "a=mid:audio 和 a=mid:video" 两种媒体流合并在一个RTP会话中传输
		if isGroup {
			parts := strings.Split(strings.Split(line, ":")[1], " ")

			if parts[0] == "BUNDLE" {
				si.isBundle = true
			}

			//看bundle后面的属性
			if len(parts) >= 2 {
				for tagno := 1; tagno < len(parts); tagno++ {
					theTag := BundleTag{id: parts[tagno], mediaType: MediaType(OTHER)} //MediaType初始化为Other，下面a=mid标签会补充这里
					si.bundleTags = append(si.bundleTags, theTag)
				}
			}
		} //end of group

		// "m=audio 9 UDP/TLS/RTP/SAVPF 111 103 104 9 0 8 126"
		if isAudio {
			mlineNum++
			si.audioSdpMLine = mlineNum
			mtype = MediaType(AUDIO_TYPE)
			si.hasAudio = true
		} //end of audio

		// "m=video 9 UDP/TLS/RTP/SAVPF 107 99"
		if isVideo {
			mlineNum++
			si.videoSdpMLine = mlineNum
			mtype = MediaType(VIDEO_TYPE)
			si.hasVideo = true
		} //end of isVideo

		// "m=video 60372 UDP/TLS/RTP/SAVPF 100 101 116 117 96"
		// "m=audio 58779 UDP/TLS/RTP/SAVPF 111 103 104 9 0 8 106 105 13 126"
		if isSAVPF {
			si.profile = SAVPF
		} // end of isSAVPF

		// "a=candidate:1467250027 1 udp 2122260223 192.168.0.196 46243 typ host generation 0"
		if isCand {
			parts := strings.Split(strings.Split(line, ":")[1], " ")
			ok := si.processCandidate(parts, mtype)
			if !ok {
				log.Println("Process candidate fail: %v", parts)
			}
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
		} // end of isIcePas

		// "a=fingerprint:sha-256 49:66:12:17:0D:1C:91:AE:57:4C:C6:36:DD:D5:97:D2:7D:62:C9:9A:7F:B9:A3:F4:70:03:E7:43:91:73:23:5E"
		if isFp {
			parts := strings.Split(strings.Split(line, "a=fingerprint:")[1], " ")
			si.fingerprint = parts[1]
			si.isFingerprint = true
		}

		//eg: a=mid:audio或a=mid:video
		if isMid {
			parts := strings.Split(line, ":")

			if len(parts) >= 2 && si.isBundle {
				thisID := parts[1]
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

		// "a=extmap:1 urn:ietf:params:rtp-hdrext:ssrc-audio-level"
		// "a=extmap:3 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time"
		if isExtMap {
			parts := strings.Split(strings.Split(line, ":")[1], " ")
			if len(parts) >= 2 {
				extmapID, err := strconv.ParseUint(parts[0], 10, 32)
				if err != nil {
					log.Printf("Unexpected extmapID(%s), oringin info=%v", parts[0], parts)
					continue
				}
				var theExtmap ExtMap
				theExtmap.mediaType = mtype
				theExtmap.value = extmapID
				theExtmap.uri = strings.Split(line, " ")[1]
				si.extMapVector = append(si.extMapVector, theExtmap)
			}
		} // end of isExtMap

		// "a=sendrecv"
		if isSendRecv {
			if mtype == AUDIO_TYPE {
				si.audioDirection = SENDRECV
			} else {
				si.videoDirection = SENDRECV
			}
		}

		if isRecvOnly {
			if mtype == AUDIO_TYPE {
				si.audioDirection = RECVONLY
			} else {
				si.videoDirection = RECVONLY
			}
		}

		if isSendOnly {
			if mtype == AUDIO_TYPE {
				si.audioDirection = SENDONLY
			} else {
				si.videoDirection = SENDONLY
			}
		} //end of isSendRecv

		//"a=rtcp-mux"
		if isRtcpMuxchar {
			si.isRtcpMux = true
		} //end of isRtcpMuxchar

		// "a=rtpmap:103 ISAC/16000"
		// "a=rtpmap:111 opus/48000/2"
		if isRtpmap {
			parts := strings.Split(strings.Split(line, ":")[1], " ")
			payloadType, err := strconv.ParseUint(parts[0], 10, 32)
			if err != nil {
				log.Printf("Unexpected rtpmap.payloadType(%s), oringin info=%v", parts[0], parts)
				continue
			}

			codecOptions := strings.Split(parts[1], "/")
			codecName := codecOptions[0]
			clockRate, err := strconv.ParseUint(codecOptions[1], 10, 32)
			if err != nil {
				log.Printf("Unexpected rtpmap.ClockRate(%s), oringin info=%v", parts[1], parts)
				continue
			}

			channel := uint64(0)
			if len(codecOptions) > 2 {
				channel, err = strconv.ParseUint(codecOptions[2], 10, 32)
				if err != nil {
					channel = uint64(0)
				}
			}

			mapElement, find := si.payloadParsedMap[payloadType]
			if find {
				mapElement.payloadType = payloadType
				mapElement.encodingName = codecName
				mapElement.clockRate = clockRate
				mapElement.mediaType = mtype
				mapElement.channels = channel
			} else {
				var theRTPMap RtpMap
				theRTPMap.payloadType = payloadType
				theRTPMap.encodingName = codecName
				theRTPMap.clockRate = clockRate
				theRTPMap.mediaType = mtype
				theRTPMap.channels = channel
				theRTPMap.formatParameters = make(map[string]string)
				si.payloadParsedMap[payloadType] = &theRTPMap
				si.incrementCodecCount(mtype)
			}
		} // end of isRtpmap

		// "a=fmtp:96 apt=100"
		// "a=fmtp:111 minptime=10; useinbandfec=1"
		// "a=fmtp:107 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f"
		if isFmtp {
			parts := strings.Split(strings.Split(line, ":")[1], " ")
			if len(parts) >= 2 {
				payloadType, err := strconv.ParseUint(parts[0], 10, 32)
				if err != nil {
					log.Printf("Unexpected ftmp.payloadType(%s), oringin info=%v", parts[0], parts)
					continue
				}

				optionMap := map[string]string{}
				// "a=fmtp:111 minptime=10; useinbandfec=1"
				if len(parts) > 2 {
					for i := 1; i < len(parts); i++ {
						optionParts := strings.Split(strings.Split(parts[i], ";")[0], "=")
						optionMap[optionParts[0]] = optionParts[1]
					}
				} else { //len(parts)==2
					//a=fmtp:96 apt=100
					//a=fmtp:107 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f
					optionList := strings.Split(parts[1], ";")
					for i := 0; i < len(optionList); i++ {
						optionParts := strings.Split(optionList[i], "=")
						optionMap[optionParts[0]] = optionParts[1]
					}
				}

				mapElement, find := si.payloadParsedMap[payloadType]
				if find {
					for k, v := range optionMap {
						mapElement.formatParameters[k] = v
						mapElement.mediaType = mtype
					}
				} else {
					var theRTPMap RtpMap
					theRTPMap.payloadType = payloadType
					theRTPMap.formatParameters = optionMap
					theRTPMap.mediaType = mtype
					si.payloadParsedMap[payloadType] = &theRTPMap
					si.incrementCodecCount(mtype)
				}
			}
		} // end of isFmtp

		// a=rtcp-fb:100 ccm fir
		// a=rtcp-fb:100 nack
		// a=rtcp-fb:100 nack pli
		// 最后一种尤其要注意，不能单纯用空格分割
		if isFeedback {
			parts := strings.Split(strings.Split(line, ":")[1], " ")
			payloadType, err := strconv.ParseUint(parts[0], 10, 32)
			if err != nil {
				log.Printf("Unexpected feedback(%s), oringin info=%v", parts[0], parts)
				continue
			}

			fbContent := strings.Split(line, parts[0]+" ")

			if len(fbContent) >= 2 {
				mapElement, find := si.payloadParsedMap[payloadType]
				if find {
					mapElement.feedbackTypes = append(mapElement.feedbackTypes, fbContent[1])
				} else {
					var theRTPMap RtpMap
					theRTPMap.payloadType = payloadType
					theRTPMap.feedbackTypes = append(theRTPMap.feedbackTypes, fbContent[1])
					theRTPMap.formatParameters = make(map[string]string)
					theRTPMap.mediaType = mtype
					si.payloadParsedMap[payloadType] = &theRTPMap
					si.incrementCodecCount(mtype)
				}
			}
		} //end of isFeedback

		//@TODO 对于多WMS和多SSRC的要能正确处理
		//a=ssrc:3570614608 cname:4TOk42mSjXCkVIa6
		//a=ssrc:3570614608 msid:lgsCFqt9kN2fVKw5wg3NKqGdATQoltEwOdMS 35429d94-5637-4686-9ecd-7d0622261ce8
		//a=ssrc:3570614608 mslabel:lgsCFqt9kN2fVKw5wg3NKqGdATQoltEwOdMS
		//a=ssrc:3570614608 label:35429d94-5637-4686-9ecd-7d0622261ce8
		if isSsrc && !isSsrcGroup {
			parts := strings.Split(strings.Split(line, ":")[1], " ")
			if mtype == VIDEO_TYPE && si.videoSsrc == 0 {
				ssrc, err := strconv.ParseUint(parts[0], 10, 32)
				if err != nil {
					log.Printf("Unexpected ssrc(%s), oringin info=%v", parts[0], parts)
					continue
				}
				si.videoSsrc = ssrc
			} else if mtype == AUDIO_TYPE && si.audioSsrc == 0 {
				ssrc, err := strconv.ParseUint(parts[0], 10, 32)
				if err != nil {
					log.Printf("Unexpected ssrc(%s), oringin info=%v", parts[0], parts)
					continue
				}
				si.audioSsrc = ssrc
			}
		} // end of isSsrc

		// "a=ssrc-group:FID 2231627014(video的ssrcid) 632943048(video的rtxssrc)"
		if isSsrcGroup {
			if mtype == VIDEO_TYPE { //只有video属性下才会有
				parts := strings.Split(strings.Split(line, ":")[1], " ")
				if len(parts) >= 3 {
					//为啥叫rtxssrc，什么含义呢？
					rtxssrc, err := strconv.ParseUint(parts[2], 10, 32)
					if err != nil {
						log.Printf("Unexpected rtxssrc(%s), oringin info=%v", parts[2], parts)
						continue
					}
					si.videoRtxSsrc = rtxssrc
				}
			}
		} // end of isSsrcGroup

		// 以下这些是不知道顺序的，往下放 @FIXME
		// "a=crypto:sha xxxx" 真实场景下暂时没见到这个属性
		if isCrypt {
			var crypto CryptoInfo
			cryptopiece := strings.Split(strings.Split(line, ":")[1], " ")

			// FIXME: add error checking here
			crypto.cipherSuite = cryptopiece[1]
			crypto.keyParams = cryptopiece[3]
			crypto.mediaType = mtype
			si.cryptoVector = append(si.cryptoVector, &crypto)
		} //end of isCrypt

		// "b=AS:" 没见过实例
		if isBandwidth {
			if mtype == VIDEO_TYPE {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					videoBandwidth, err := strconv.ParseUint(parts[1], 10, 32)
					if err != nil {
						log.Printf("Unexpected videoBandwidth(%s), oringin info=%v", parts[1], parts)
						continue
					}
					si.videoBandwidth = videoBandwidth
				}
			}
		} //end of isBandwidth

	} //end of for

	if len(si.iceVideoUsername) == 0 && len(si.iceAudioUsername) == 0 {
		return false
	} else if len(si.iceVideoUsername) == 0 {
		log.Println("Video credentials empty, setting the audio ones")
		si.iceVideoUsername = si.iceAudioUsername
		si.iceVideoPassword = si.iceAudioPassword
	} else if len(si.iceAudioUsername) == 0 {
		log.Println("Audio credentials empty, setting the video ones")
		si.iceAudioUsername = si.iceVideoUsername
		si.iceAudioPassword = si.iceVideoPassword
	}

	for i := 0; i < len(si.candidateVector); i++ {
		cand := si.candidateVector[i]
		cand.isBundle = si.isBundle
		if cand.mediaType == VIDEO_TYPE {
			cand.username = si.iceVideoUsername
			cand.password = si.iceVideoPassword
		} else {
			cand.username = si.iceAudioUsername
			cand.password = si.iceAudioPassword
		}
	}

	return true
}

func (si *SdpInfo) incrementCodecCount(mtype MediaType) {
	if mtype == AUDIO_TYPE {
		si.audioCodecs++
	} else {
		si.videoCodecs++
	}
}

//eg:  "a=candidate:1467250027 1 udp 2122260223 192.168.0.196 46243 typ host generation 0"
//eg: "a=candidate:1367696781 1 udp 33562367 138. 49462 typ relay raddr 138.4 rport 53531 generation 0"
func (si *SdpInfo) processCandidate(parts []string, mtype MediaType) bool {
	var cand CandidateInfo
	var typesStr = []string{"host", "srflx", "prflx", "relay"}
	cand.mediaType = mtype
	cand.foundation = parts[0] //这个数字干啥用的？
	compID, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil { // componentID=0代表rtp，=1代表rtcp
		log.Printf("Unexpected ComponentID(%s), oringin info=%v", parts[1], parts)
	}
	cand.componentID = compID

	cand.netProtocol = parts[2] // tcp or udp

	//暂时不支持TCP candidate
	if strings.ToLower(cand.netProtocol) != "udp" {
		return false
	}

	priority, err := strconv.ParseUint(parts[3], 10, 32)
	if err != nil {
		log.Printf("Unexpected priority(%s), oringin info=%v", parts[3], parts)
	}
	cand.priority = priority

	cand.hostAddress = parts[4]

	hostport, err := strconv.ParseUint(parts[5], 10, 32)
	if err != nil {
		log.Printf("Unexpected HostPort(%s), oringin info=%v", parts[5], parts)
	}
	cand.hostPort = uint32(hostport)

	if parts[6] != "typ" {
		return false
	}

	candType := 2 //默认走SRFLX吧...
	for i := 0; i < 4; i++ {
		if parts[7] == typesStr[i] {
			candType = i
			break
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
		cand.hostType = SRFLX
	}

	if cand.hostType == SRFLX || cand.hostType == RELAY {
		cand.rAddress = parts[9]
		rport, err := strconv.ParseUint(parts[11], 10, 32)
		if err != nil {
			log.Printf("Unexpected rport(%s), oringin info=%v", parts[10], parts)
		}
		cand.rPort = uint32(rport)
	}

	si.candidateVector = append(si.candidateVector, &cand)
	return true
}
