/**
 *  go test -run="Sdp|Bufio" -v
 *  go test -v -test.run TestProcessSdp
 *	go test sdp.go sdp_test.go -v
 */
package signaling

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestProcessSdp(t *testing.T) {
	var sdp SdpInfo
	sdp.processSdp(exampleFromWeb, "other")
}

func TestBufio(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(ghostAnswerSdp))
	for scanner.Scan() {
		t.Log("len=", len(scanner.Text()), scanner.Text()) // Println will add back the final '\n'
	}
	t.Log("--------------------------------------------------------")
	sdpReader := bufio.NewReader(strings.NewReader(ghostAnswerSdp))
	for {
		//line里面保留了最后的回车换行这个不好，弃
		line, err := sdpReader.ReadString('\n')
		t.Log("len=", len(line), line)
		if err == io.EOF {
			break
		}
	}
}

func TestStringUtil(t *testing.T) {
	fmt.Printf("%q\n", strings.Split("a,b,c", ","))
	fmt.Printf("%q\n", strings.Split("a man a plan a canal panama", "a "))
	fmt.Printf("%q\n", strings.Split(" xyz ", ""))
	fmt.Printf("%q\n", strings.Split("", "Bernardo O'Higgins"))
}

var ghostAnswerSdp = `v=0
o=- 9221255610544057670 2 IN IP4 127.0.0.1
s=-
t=0 0
a=group:BUNDLE audio video
a=msid-semantic: WMS KoXevu1EQy9vnPxfGgjuwjQRzCIHzasMfVG3
m=audio 9 UDP/TLS/RTP/SAVPF 111 103 104 9 0 8 126
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:B5l2
a=ice-pwd:nBp7g7QihTj4y3FKvBIOx7fE
a=fingerprint:sha-256 44:AC:0C:61:47:FF:83:52:A2:97:59:B8:4B:B9:29:78:40:4F:4A:5C:8F:72:6F:94:10:60:C2:FA:F1:EE:50:4B
a=setup:active
a=mid:audio
a=extmap:1 urn:ietf:params:rtp-hdrext:ssrc-audio-level
a=extmap:3 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=sendrecv
a=rtcp-mux
a=rtpmap:111 opus/48000/2
a=rtcp-fb:111 transport-cc
a=fmtp:111 minptime=10;useinbandfec=1
a=rtpmap:103 ISAC/16000
a=rtpmap:104 ISAC/32000
a=rtpmap:9 G722/8000
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:126 telephone-event/8000
a=ssrc:2326073733 cname:1P21K7s5u19ytY8A
a=ssrc:2326073733 msid:KoXevu1EQy9vnPxfGgjuwjQRzCIHzasMfVG3 3f857bb4-ada7-4fea-9e63-02f2d45b03b0
a=ssrc:2326073733 mslabel:KoXevu1EQy9vnPxfGgjuwjQRzCIHzasMfVG3
a=ssrc:2326073733 label:3f857bb4-ada7-4fea-9e63-02f2d45b03b0
m=video 9 UDP/TLS/RTP/SAVPF 107 99
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:B5l2
a=ice-pwd:nBp7g7QihTj4y3FKvBIOx7fE
a=fingerprint:sha-256 44:AC:0C:61:47:FF:83:52:A2:97:59:B8:4B:B9:29:78:40:4F:4A:5C:8F:72:6F:94:10:60:C2:FA:F1:EE:50:4B
a=setup:active
a=mid:video
a=extmap:2 urn:ietf:params:rtp-hdrext:toffset
a=extmap:3 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=extmap:4 urn:3gpp:video-orientation
a=extmap:5 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01
a=extmap:6 http://www.webrtc.org/experiments/rtp-hdrext/playout-delay
a=sendrecv
a=rtcp-mux
a=rtcp-rsize
a=rtpmap:107 H264/90000
a=rtcp-fb:107 ccm fir
a=rtcp-fb:107 nack
a=rtcp-fb:107 nack pli
a=rtcp-fb:107 goog-remb
a=rtcp-fb:107 transport-cc
a=fmtp:107 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f
a=rtpmap:99 rtx/90000
a=fmtp:99 apt=107
a=ssrc-group:FID 923057356 763358564
a=ssrc:923057356 cname:1P21K7s5u19ytY8A
a=ssrc:923057356 msid:KoXevu1EQy9vnPxfGgjuwjQRzCIHzasMfVG3 b71ad83f-8873-412d-8d8a-258289bb4ee5
a=ssrc:923057356 mslabel:KoXevu1EQy9vnPxfGgjuwjQRzCIHzasMfVG3
a=ssrc:923057356 label:b71ad83f-8873-412d-8d8a-258289bb4ee5
a=ssrc:763358564 cname:1P21K7s5u19ytY8A
a=ssrc:763358564 msid:KoXevu1EQy9vnPxfGgjuwjQRzCIHzasMfVG3 b71ad83f-8873-412d-8d8a-258289bb4ee5
a=ssrc:763358564 mslabel:KoXevu1EQy9vnPxfGgjuwjQRzCIHzasMfVG3
a=ssrc:763358564 label:b71ad83f-8873-412d-8d8a-258289bb4ee5`

var ghostOfferSdp = `v=0
o=- 4070829141141562085 2 IN IP4 127.0.0.1
s=-
t=0 0
a=group:BUNDLE audio video
a=msid-semantic: WMS PmjlFaJdelMzWQvkizKtQ2EMAQtBRHOsQbPK
m=audio 9 UDP/TLS/RTP/SAVPF 111 103 104 9 0 8 126
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:0Go3
a=ice-pwd:p8cjZGj7C34C3WsrbQpTtkXr
a=fingerprint:sha-256 30:26:FC:A6:6B:0B:96:55:58:5E:10:5F:32:87:4D:26:AA:35:B0:D3:56:72:FE:64:F7:8D:3F:0B:8C:79:96:0D
a=setup:actpass
a=mid:audio
a=extmap:1 urn:ietf:params:rtp-hdrext:ssrc-audio-level
a=extmap:3 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=sendrecv
a=rtcp-mux
a=rtpmap:111 opus/48000/2
a=rtcp-fb:111 transport-cc
a=fmtp:111 minptime=10;useinbandfec=1
a=rtpmap:103 ISAC/16000
a=rtpmap:104 ISAC/32000
a=rtpmap:9 G722/8000
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:126 telephone-event/8000
a=ssrc:3515318943 cname:idXCTHc9lsOEy+1H
a=ssrc:3515318943 msid:PmjlFaJdelMzWQvkizKtQ2EMAQtBRHOsQbPK 44f66f61-75ee-46d7-a12b-e3687d8ceb4e
a=ssrc:3515318943 mslabel:PmjlFaJdelMzWQvkizKtQ2EMAQtBRHOsQbPK
a=ssrc:3515318943 label:44f66f61-75ee-46d7-a12b-e3687d8ceb4e
m=video 9 UDP/TLS/RTP/SAVPF 107 99
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:0Go3
a=ice-pwd:p8cjZGj7C34C3WsrbQpTtkXr
a=fingerprint:sha-256 30:26:FC:A6:6B:0B:96:55:58:5E:10:5F:32:87:4D:26:AA:35:B0:D3:56:72:FE:64:F7:8D:3F:0B:8C:79:96:0D
a=setup:actpass
a=mid:video
a=extmap:2 urn:ietf:params:rtp-hdrext:toffset
a=extmap:3 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=extmap:4 urn:3gpp:video-orientation
a=extmap:5 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01
a=extmap:6 http://www.webrtc.org/experiments/rtp-hdrext/playout-delay
a=sendrecv
a=rtcp-mux
a=rtcp-rsize
a=rtpmap:107 H264/90000
a=rtcp-fb:107 ccm fir
a=rtcp-fb:107 nack
a=rtcp-fb:107 nack pli
a=rtcp-fb:107 goog-remb
a=rtcp-fb:107 transport-cc
a=fmtp:107 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f
a=rtpmap:99 rtx/90000
a=fmtp:99 apt=107
a=ssrc-group:FID 2362060203 4056169427
a=ssrc:2362060203 cname:idXCTHc9lsOEy+1H
a=ssrc:2362060203 msid:PmjlFaJdelMzWQvkizKtQ2EMAQtBRHOsQbPK 469d252e-1a49-4c0e-9c3e-05e4411e0343
a=ssrc:2362060203 mslabel:PmjlFaJdelMzWQvkizKtQ2EMAQtBRHOsQbPK
a=ssrc:2362060203 label:469d252e-1a49-4c0e-9c3e-05e4411e0343
a=ssrc:4056169427 cname:idXCTHc9lsOEy+1H
a=ssrc:4056169427 msid:PmjlFaJdelMzWQvkizKtQ2EMAQtBRHOsQbPK 469d252e-1a49-4c0e-9c3e-05e4411e0343
a=ssrc:4056169427 mslabel:PmjlFaJdelMzWQvkizKtQ2EMAQtBRHOsQbPK
a=ssrc:4056169427 label:469d252e-1a49-4c0e-9c3e-05e4411e0343`

var exampleFromWeb = `v=0
o=- 4611731400430051336 2 IN IP4 127.0.0.1
s=-
t=0 0
a=group:BUNDLE audio video
a=msid-semantic: WMS lgsCFqt9kN2fVKw5wg3NKqGdATQoltEwOdMS
m=audio 58779 UDP/TLS/RTP/SAVPF 111 103 104 9 0 8 106 105 13 126
c=IN IP4 217.130.243.155
a=rtcp:51472 IN IP4 217.130.243.155
a=candidate:1467250027 1 udp 2122260223 192.168.0.196 46243 typ host generation 0
a=candidate:1467250027 2 udp 2122260222 192.168.0.196 56280 typ host generation 0
a=candidate:435653019 1 tcp 1845501695 192.168.0.196 0 typ host tcptype active generation 0
a=candidate:435653019 2 tcp 1845501695 192.168.0.196 0 typ host tcptype active generation 0
a=candidate:1853887674 1 udp 1518280447 47.61.61.61 36768 typ srflx raddr 192.168.0.196 rport 36768 generation 0
a=candidate:1853887674 2 udp 1518280447 47.61.61.61 36768 typ srflx raddr 192.168.0.196 rport 36768 generation 0
a=candidate:750991856 2 udp 25108222 237.30.30.30 51472 typ relay raddr 47.61.61.61 rport 54763 generation 0
a=candidate:750991856 1 udp 25108223 237.30.30.30 58779 typ relay raddr 47.61.61.61 rport 54761 generation 0
a=ice-ufrag:Oyef7uvBlwafI3hT
a=ice-pwd:T0teqPLNQQOf+5W+ls+P2p16
a=fingerprint:sha-256 49:66:12:17:0D:1C:91:AE:57:4C:C6:36:DD:D5:97:D2:7D:62:C9:9A:7F:B9:A3:F4:70:03:E7:43:91:73:23:5E
a=setup:actpass
a=mid:audio
a=extmap:1 urn:ietf:params:rtp-hdrext:ssrc-audio-level
a=extmap:3 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=sendrecv
a=rtcp-mux
a=rtpmap:111 opus/48000/2
a=fmtp:111 minptime=10; useinbandfec=1
a=rtpmap:103 ISAC/16000
a=rtpmap:104 ISAC/32000
a=rtpmap:9 G722/8000
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:106 CN/32000
a=rtpmap:105 CN/16000
a=rtpmap:13 CN/8000
a=rtpmap:126 telephone-event/8000
a=maxptime:60
a=ssrc:3570614608 cname:4TOk42mSjXCkVIa6
a=ssrc:3570614608 msid:lgsCFqt9kN2fVKw5wg3NKqGdATQoltEwOdMS 35429d94-5637-4686-9ecd-7d0622261ce8
a=ssrc:3570614608 mslabel:lgsCFqt9kN2fVKw5wg3NKqGdATQoltEwOdMS
a=ssrc:3570614608 label:35429d94-5637-4686-9ecd-7d0622261ce8
m=video 60372 UDP/TLS/RTP/SAVPF 100 101 116 117 96
c=IN IP4 217.130.243.155
a=rtcp:64891 IN IP4 217.130.243.155
a=candidate:1467250027 1 udp 2122260223 192.168.0.196 56143 typ host generation 0
a=candidate:1467250027 2 udp 2122260222 192.168.0.196 58874 typ host generation 0
a=candidate:435653019 1 tcp 1518280447 192.168.0.196 0 typ host tcptype active generation 0
a=candidate:435653019 2 tcp 1518280446 192.168.0.196 0 typ host tcptype active generation 0
a=candidate:1853887674 1 udp 1518280447 47.61.61.61 36768 typ srflx raddr 192.168.0.196 rport 36768 generation 0
a=candidate:1853887674 1 udp 1518280447 47.61.61.61 36768 typ srflx raddr 192.168.0.196 rport 36768 generation 0
a=candidate:750991856 1 udp 25108223 237.30.30.30 60372 typ relay raddr 47.61.61.61 rport 54765 generation 0
a=candidate:750991856 2 udp 25108222 237.30.30.30 64891 typ relay raddr 47.61.61.61 rport 54767 generation 0
a=ice-ufrag:Oyef7uvBlwafI3hT
a=ice-pwd:T0teqPLNQQOf+5W+ls+P2p16
a=fingerprint:sha-256 49:66:12:17:0D:1C:91:AE:57:4C:C6:36:DD:D5:97:D2:7D:62:C9:9A:7F:B9:A3:F4:70:03:E7:43:91:73:23:5E
a=setup:actpass
a=mid:video
a=extmap:2 urn:ietf:params:rtp-hdrext:toffset
a=extmap:3 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=extmap:4 urn:3gpp:video-orientation
a=sendrecv
a=rtcp-mux
a=rtpmap:100 VP8/90000
a=rtcp-fb:100 ccm fir
a=rtcp-fb:100 nack
a=rtcp-fb:100 nack pli
a=rtcp-fb:100 goog-remb
a=rtpmap:101 VP9/90000
a=rtcp-fb:101 ccm fir
a=rtcp-fb:101 nack
a=rtcp-fb:101 nack pli
a=rtcp-fb:101 goog-remb
a=rtpmap:116 red/90000
a=rtpmap:117 ulpfec/90000
a=rtpmap:96 rtx/90000
a=fmtp:96 apt=100
a=ssrc-group:FID 2231627014 632943048
a=ssrc:2231627014 cname:4TOk42mSjXCkVIa6
a=ssrc:2231627014 msid:lgsCFqt9kN2fVKw5wg3NKqGdATQoltEwOdMS daed9400-d0dd-4db3-b949-422499e96e2d
a=ssrc:2231627014 mslabel:lgsCFqt9kN2fVKw5wg3NKqGdATQoltEwOdMS
a=ssrc:2231627014 label:daed9400-d0dd-4db3-b949-422499e96e2d
a=ssrc:632943048 cname:4TOk42mSjXCkVIa6
a=ssrc:632943048 msid:lgsCFqt9kN2fVKw5wg3NKqGdATQoltEwOdMS daed9400-d0dd-4db3-b949-422499e96e2d
`
