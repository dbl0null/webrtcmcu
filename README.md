# WebrtcMCU

提供信令（Signaling）服务（HTTPS）、建立浏览器与AccessService之间的媒体通道以及相关逻辑功能的Web页面（后面再考虑分离出去）。AccessService会为每个接入的浏览器终端创建一个对应的虚拟对端（Ghost），并与之建立中转模式的媒体流通道，可能会包括：ICE（STUN/TURN)建立中转连接、DTLS加密握手以及SRTP/SRTCP数据的加解密、处理（转给混音、混流服务）和传输（转发给相同会话里的其他人）

# 信令服务HTTPs证书生成

HTTPs服务所使用的证书生成过程如下所示（由于不是权威认证机构签发的证书，所以浏览器会有警告）：

1. openssl genrsa -out privatekey.pem 1024 //生成私钥
1. openssl req -new -key privatekey.pem -out certrequest.csr //用私钥生成证书
1. openssl x509 -req -in certrequest.csr -signkey privatekey.pem -out certificate.pem //用自己的私钥给自己的证书签名，浏览器会提示不安全，但好歹能自测用

这个简易的证书生成过程是参考[这里](http://blog.csdn.net/u011244942/article/details/49306777)的说明。还可以参考[点我](http://www.cnblogs.com/kyrios/p/tls-and-certificates.html)和[点我](http://cnodejs.org/topic/54745ac22804a0997d38b32d)进一步加深理解。

# 实用工具

## Chrome工具

chrome://webrtc-internals
chrome://webrtc-logs/

## Visual Studio Code Debug

需要安装[Delve](https://github.com/derekparker/delve/blob/master/Documentation/installation/osx/install.md)，Mac上安装有如下提示需要注意：
If you get `"could not launch process: could not fork/exec"`, you need to try in a new terminal.When uninstalling, to remove the dlv-cert certificate, run command below. Alternatively, you may want to delete from the Keychain (with the Imported private key).
 ```   
 $ sudo security delete-certificate -t -c dlv-cert /Library/Keychains/System.keychain
 ```

# 相关依赖

## Go的WebSocket支持库
github.com/gorilla/websocket

