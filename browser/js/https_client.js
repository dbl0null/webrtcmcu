'use strict';

var localStreamReady = false;
var doOfferAlready = false;

var localStream;
var pc;
var remoteStream;

//只需要跨网直连模式
var wantHostMode = false;
var wantReflexiveMode = true;
var wantRelayMode = false;

var pcConfig = {};

// Set up audio and video regardless of what devices are present.
/*
var sdpConstraints = {
  'mandatory': {
    'OfferToReceiveAudio': true,
    'OfferToReceiveVideo': true
  }
};
*/

/////////////////////////////////////////////

var enterRoomButton = document.getElementById('enteroom');
var userIDTxt = document.getElementById('userid');
var roomIDTxt = document.getElementById('roomid');

var userIDVal = ''
var roomIDVal = ''
var dataChanPort = 0

/////////////////////////////////////////////

enterRoomButton.disabled = true;
var socket;
if (window["WebSocket"]) {
  socket = new WebSocket("wss://" + document.location.host + "/websocket");
  socket.onclose = function (evt) {
    var item = document.createElement("div");
    item.innerHTML = "<b>Connection closed.</b>";
  };

  socket.onopen = function (evt) {
    if (localStreamReady && !doOfferAlready) {
      doOfferAlready = true;
      doOffer();
    }
  }

  socket.onmessage = function (evt) 
  {
    var message = JSON.parse(evt.data);
    console.log('[OnMessage][' + message.type + '] Receive from the other!', message);
    if (message.type === 'answer') {
      pc.setRemoteDescription(new RTCSessionDescription(message));
    }
    else if (message.type === 'candidate') {
      var candidate = new RTCIceCandidate({
        sdpMLineIndex: message.label,
        candidate: message.candidate
      });
      pc.addIceCandidate(candidate);
    }
    ///收到别人的进房间通知，此时可以启动通话
    else if (message.type === 'participate') {
      alert('Other people('+message.userid+') come in room('+roomid+')');
    }
    ///自己进房间成功了，开始能力协商吧~
    else if (message.type === 'enteroomres') {
      if (message.error != '') {
        enterRoomButton.disabled = false
        alert('{' + message.userid + '} enter room{' + message.roomid + '} error: ' + message.error);
      } else {
        enterRoomButton.disabled = true
        dataChanPort = message.datachanport
        if (socket.readyState === 1 && !doOfferAlready) // 1=OPEN, 连接已开启并准备好进行通信。
        {
          doOfferAlready = true;
          doOffer();
          alert('{' + message.userid + '} enter room{' + message.roomid + '} succ! participants:' + message.participants);
        }
        else {
          enterRoomButton.disabled = false
          alert('socket.readyState=' + socket.readyState + ', doOfferAlready=' + doOfferAlready);
        }
      }
    }
    else {
      console.log('[OnMessage][' + message.type + '] unknown msg:', message);
    }
  };
} else {
  var item = document.createElement("div");
  item.innerHTML = "<b>Your browser does not support WebSockets.</b>";
}

/////////////////////////////////////////////
enterRoomButton.onclick = enterRoom;
enterRoomButton.disabled = false;

function enterRoom() {
  enterRoomButton.disabled = true;
  userIDVal = userIDTxt.value;
  roomIDVal = roomIDTxt.value
  sendMsgByWS({
    type: "enteroom",
    userid: userIDVal,
    roomid: roomIDVal
  });
}

/////////////////////////////////////////////
function sendMsgByWS(message) {
  //http://stackoverflow.com/questions/13028604/sending-a-javascript-object-through-websockets-with-faye
  //统一发送json格式的数据，用type字段标识当命令号
  //webrtc定义了：offer、answer、candidate，我们也可以自定义类型
  socket.send(JSON.stringify(message));
  console.log('[SEND_MSG_BY_WS]: ', message);
}

////////////////////////////////////////////////////

var localVideo = document.querySelector('#localVideo');
var remoteVideo = document.querySelector('#remoteVideo');

navigator.mediaDevices.getUserMedia({
  audio: true,
  video: true
})
  .then(gotLocalStream)//启动设备会被卡住，gotStream可能后面才会执行
  .catch(function (e) {
    alert('getUserMedia() error: ' + e.name);
  });

function gotLocalStream(stream) {
  localVideo.src = window.URL.createObjectURL(stream);
  localStream = stream;
  createPeerConnection();
  pc.addStream(localStream);
  localStreamReady = true;
}

//浏览器关闭动作
window.onbeforeunload = function () {
  socket.close();
  console.log('[OnBeforeUnload] close()');
};

/////////////////////////////////////////////////////////

function createPeerConnection() {
  try {
    pc = new RTCPeerConnection();
    pc.onicecandidate = handleIceCandidate;
    pc.onaddstream = handleRemoteStreamAdded;
    pc.onremovestream = handleRemoteStreamRemoved;
    console.log('>>>>>> createPeerConnection()', pc);
  } catch (e) {
    console.log('>>>>>> createPeerConnection() Failed to create PeerConnection, exception: ' + e.message);
    alert('Cannot create RTCPeerConnection object.');
    return;
  }
}

function handleIceCandidate(event) {
  var ice = event.candidate;

  if (ice) {
    var isHost = (ice.candidate.indexOf('typ host') !== -1);
    var isSrflx = (ice.candidate.indexOf('srflx') !== -1);
    var isRelay = (ice.candidate.indexOf('relay') !== -1);
    var candidateType = isHost ? 'host' : (isSrflx ? 'srflx' : 'relay');

    if (wantHostMode && ice.candidate.indexOf('typ host') == -1) {
      console.log('>>>>>> handleIceCandidate(event) pass candidate [' + candidateType + ']');
      return;
    }

    if (wantReflexiveMode && ice.candidate.indexOf('srflx') == -1) {
      console.log('>>>>>> handleIceCandidate(event) pass candidate [' + candidateType + ']');
      return;
    }

    if (wantRelayMode && ice.candidate.indexOf('relay') == -1) {
      console.log('>>>>>> handleIceCandidate(event) pass candidate [' + candidateType + ']');
      return;
    }

    console.log('>>>>>> handleIceCandidate(event) selected a ' + candidateType + ' candidate and send to the other');
    sendMsgByWS({
      type: 'candidate',
      label: event.candidate.sdpMLineIndex,
      id: event.candidate.sdpMid,
      candidate: event.candidate.candidate,
      userid: userIDVal,
      roomid: roomIDVal
    });
  }
  else {
    console.log('>>>>>> handleIceCandidate(event) End of candidates.');
  }
}

function handleRemoteStreamAdded(event) {
  console.log('>>>>>> handleRemoteStreamAdded(event) Remote stream added.', event);
  remoteVideo.src = window.URL.createObjectURL(event.stream);
  remoteStream = event.stream;
}

function handleRemoteStreamRemoved(event) {
  console.log('>>>>>> handleRemoteStreamRemoved(event) Remote stream removed. Event: ', event);
}

function doOffer() {
  console.log('>>>>>> doOffer(): Sending offer to peer');
  pc.createOffer(setLocalAndSendMessage, handleCreateOfferError);
}

function doAnswer() {
  console.log('>>>>>> doAnswer(): Sending answer to peer.');
  pc.createAnswer().then(
    setLocalAndSendMessage,
    onCreateSessionDescriptionError
  );
}

function setLocalAndSendMessage(sessionDescription) {
  pc.setLocalDescription(sessionDescription);
  sendMsgByWS({
    type: 'offer',
    sdp: sessionDescription.sdp,
    userid: userIDVal,
    roomid: roomIDVal
  });
}

function handleCreateOfferError(event) {
  console.log('createOffer() error: ', event);
}

function onCreateSessionDescriptionError(error) {
  console.trace('Failed to create session description: ' + error.toString());
}