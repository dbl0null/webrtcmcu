# webrtcmcu
提供信令（Signaling）服务（HTTPS）、建立浏览器与AccessService之间的媒体通道以及相关逻辑功能的Web页面（后面再考虑分离出去）。AccessService会为每个接入的浏览器终端创建一个对应的虚拟对端（Ghost），并与之建立中转模式的媒体流通道，可能会包括：ICE（STUN/TURN)建立中转连接、DTLS加密握手以及SRTP/SRTCP数据的加解密、处理（转给混音、混流服务）和传输（转发给相同会话里的其他人）
