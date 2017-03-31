'use strict';

var localStreamReady = false;
var doOfferAlready = false

var localStream;
var pc;
var remoteStream;

//只需要跨网直连模式
var wantHostMode      = false;
var wantReflexiveMode = true;
var wantRelayMode     = false;

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

var socket;
if (window["WebSocket"]) {
      socket = new WebSocket("wss://"+document.location.host+"/websocket");
      socket.onclose = function (evt) {
          var item = document.createElement("div");
          item.innerHTML = "<b>Connection closed.</b>";
      };
      
      socket.onopen = function (evt) 
      {
        if (localStreamReady && !doOfferAlready)
        {
          doOfferAlready = true;
          doOffer();
        }
      }

      socket.onmessage = function (evt) 
      {
          var message = JSON.parse(evt.data);
          console.log('[OnMessage]['+message.type+'] Receive from the other!', message);
          if (message.type === 'answer') 
          {
            pc.setRemoteDescription(new RTCSessionDescription(message));
          }
          else if (message.type === 'candidate' ) 
          {
            var candidate = new RTCIceCandidate({
                sdpMLineIndex: message.label,
                candidate: message.candidate
              });
            pc.addIceCandidate(candidate);
          } 
          else if (message.type === 'hangup' ) 
          {
            handleRemoteHangup();
          }
          else
          {
            console.log('[OnMessage]['+message.type+'] unknown msg:', message);
          }
      };
} else {
    var item = document.createElement("div");
    item.innerHTML = "<b>Your browser does not support WebSockets.</b>";
}

/////////////////////////////////////////////
function sendMsgByWS(message)
{
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
.then(gotStream)//启动设备会被卡住，gotStream可能后面才会执行
.catch(function(e) {
  alert('getUserMedia() error: ' + e.name);
});

function gotStream(stream) 
{
  localVideo.src = window.URL.createObjectURL(stream);
  localStream = stream;
  createPeerConnection();
  pc.addStream(localStream);

  localStreamReady = true
  //通道准备好了再doOffer
  if( socket.readyState === 1 && !doOfferAlready) // 1=OPEN, 连接已开启并准备好进行通信。
  {
    doOfferAlready = true;
    doOffer();
  }
}

//浏览器关闭动作
window.onbeforeunload = function() 
{
  socket.close();
  console.log('[OnBeforeUnload] close()');
};

/////////////////////////////////////////////////////////

function createPeerConnection() 
{
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

function handleIceCandidate(event) 
{
  var ice = event.candidate;

  if (ice) 
  {
    var isHost = (ice.candidate.indexOf('typ host') !== -1);
    var isSrflx = (ice.candidate.indexOf('srflx') !== -1);
    var isRelay = (ice.candidate.indexOf('relay') !== -1);
    var candidateType = isHost?'host':(isSrflx?'srflx':'relay');

    if(wantHostMode && ice.candidate.indexOf('typ host') == -1) 
    {
      console.log('>>>>>> handleIceCandidate(event) pass candidate ['+candidateType + ']');
      return;
    }

    if(wantReflexiveMode && ice.candidate.indexOf('srflx') == -1) 
    {
      console.log('>>>>>> handleIceCandidate(event) pass candidate ['+candidateType + ']');
      return;
    }

    if(wantRelayMode && ice.candidate.indexOf('relay') == -1) 
    {
      console.log('>>>>>> handleIceCandidate(event) pass candidate ['+candidateType + ']');
      return;
    }

    console.log('>>>>>> handleIceCandidate(event) selected a '+ candidateType+' candidate and send to the other');
    sendMsgByWS({
      type: 'candidate',
      label: event.candidate.sdpMLineIndex,
      id: event.candidate.sdpMid,
      candidate: event.candidate.candidate
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

function doOffer() 
{
  console.log('>>>>>> doOffer(): Sending offer to peer');
  pc.createOffer(setLocalAndSendMessage, handleCreateOfferError);
}

function doAnswer() 
{
  console.log('>>>>>> doAnswer(): Sending answer to peer.');
  pc.createAnswer().then(
    setLocalAndSendMessage,
    onCreateSessionDescriptionError
  );
}

function setLocalAndSendMessage(sessionDescription)
{
  // Set Opus as the preferred codec in SDP if Opus is present.
  sessionDescription.sdp = preferOpus(sessionDescription.sdp);
  //console.log('????? before=', sessionDescription.sdp)
  sessionDescription.sdp = preferH264(sessionDescription.sdp);
  //console.log('????? after=', sessionDescription.sdp)
  pc.setLocalDescription(sessionDescription);
  sendMsgByWS(sessionDescription);
}

function handleCreateOfferError(event) {
  console.log('createOffer() error: ', event);
}

function onCreateSessionDescriptionError(error) {
  console.trace('Failed to create session description: ' + error.toString());
}

function stop() {
  // isAudioMuted = false;
  // isVideoMuted = false;
  pc.close();
  pc = null;
  console.log('>>>>>> stop().');
}

///////////////////////////////////////////
function preferH264(sdp)
{
  var sdpLines = sdp.split('\r\n');
  var mLineIndex;
  // Search for m line.
  for (var i = 0; i < sdpLines.length; i++) 
  {
    if (sdpLines[i].search('m=video') !== -1) 
    {
      mLineIndex = i;
      break;
    }
  }

  // If H264 is available, set it as the default in m line.
  /*
  for (i = 0; i < sdpLines.length; i++) 
  {
    if (sdpLines[i].search('H264/') !== -1) 
    {
      var h264Payload = extractSdp(sdpLines[i], /:(\d+) H264\//i);
      if (h264Payload) 
      {
        sdpLines[mLineIndex] = setDefaultCodec(sdpLines[mLineIndex], h264Payload);
      }
      break;
    }
  }
  */

  removeVideoCodecByName(sdpLines, mLineIndex, 'red');
  removeVideoCodecByName(sdpLines, mLineIndex, 'ulpfec');
  removeVideoCodecByName(sdpLines, mLineIndex, 'VP8');
  removeVideoCodecByName(sdpLines, mLineIndex, 'VP9');
  
  sdp = sdpLines.join('\r\n');
  return sdp;
}


function removeVideoCodecByName(sdpLines, mLineIndex, targetName)
{
  var targetPayload = removeRtpmapByName(sdpLines, mLineIndex, targetName);
  if( targetPayload )
  {
    removeRtcpfbByPayload(sdpLines, mLineIndex, targetPayload);
    removeFmtpByPayload(sdpLines, mLineIndex, targetPayload);
    var associatedPayload = removeFmtpAptByPayload(sdpLines, mLineIndex, targetPayload)
    if( associatedPayload )
    {
     removeRtmpByPayload(sdpLines, mLineIndex, associatedPayload)
    }
  }
}

function removeRtmpByPayload(sdpLines, mLineIndex, targetPayload)
{
  var mLineElements = sdpLines[mLineIndex].split(' ');
  var regularEq = new RegExp('a=rtpmap:'+targetPayload+' ', 'i' );

  for (var i = sdpLines.length - 1; i >= 0; i--) 
  {
    if (sdpLines[i].search(regularEq) !=-1)
    {
      var cnPos = mLineElements.indexOf(targetPayload);
      if (cnPos !== -1) {
        mLineElements.splice(cnPos, 1);
      }
      sdpLines.splice(i, 1);
    }
  }

  sdpLines[mLineIndex] = mLineElements.join(' ');
}

function removeFmtpAptByPayload(sdpLines, mLineIndex, targetPayload)
{
  var associatedPayload = null;
  var regularEq = new RegExp('a=fmtp:[0-9]+ apt='+targetPayload, 'i' );

  for (var i = sdpLines.length - 1; i >= 0; i--) 
  {
    if (sdpLines[i].search(regularEq) !=-1)
    {
      associatedPayload = sdpLines[i].split(/[:| ]/)[1];
      sdpLines.splice(i, 1);
    }
  }

  return associatedPayload;
}

function removeFmtpByPayload(sdpLines, mLineIndex, targetPayload)
{
  for (var i = sdpLines.length - 1; i >= 0; i--) 
  {
    if (sdpLines[i].search('a=fmtp:'+targetPayload) !=-1) 
      sdpLines.splice(i, 1);
  }
}

function removeRtcpfbByPayload(sdpLines, mLineIndex, targetPayload)
{
    for (var i = sdpLines.length - 1; i >= 0; i--) 
    {
      if (sdpLines[i].search('a=rtcp-fb:'+targetPayload) !=-1) 
        sdpLines.splice(i, 1);
    }
}

function removeRtpmapByName(sdpLines, mLineIndex, targetName)
{
  var mLineElements = sdpLines[mLineIndex].split(' ');
  var regularEq = new RegExp('a=rtpmap:(\\d+) '+targetName+'\/\\d+', 'i' );

  var targetPayload = null;
  for (var i = sdpLines.length - 1; i >= 0; i--) 
  {
    var payload = extractSdp(sdpLines[i], regularEq);

    if (payload) 
    {
      var cnPos = mLineElements.indexOf(payload);
      if (cnPos !== -1) {
        // Remove CN payload from m line.
        mLineElements.splice(cnPos, 1);
      }
      // Remove CN line in sdp
      sdpLines.splice(i, 1);

      targetPayload = payload;
    }
  }

  sdpLines[mLineIndex] = mLineElements.join(' ');
  return targetPayload;
}


function extractSdp(sdpLine, pattern) 
{
  var result = sdpLine.match(pattern);
  return result && result.length === 2 ? result[1] : null;
}


// Set the selected codec to the first in m line.
function setDefaultCodec(mLine, payload) 
{
  var elements = mLine.split(' ');
  var newLine = [];
  var index = 0;
  for (var i = 0; i < elements.length; i++) {
    if (index === 3) { // Format of media starts from the fourth.
      newLine[index++] = payload; // Put target payload to the first.
    }
    if (elements[i] !== payload) {
      newLine[index++] = elements[i];
    }
  }
  return newLine.join(' ');
}


///////////////////////////////////////////

// Set Opus as the default audio codec if it's present.
function preferOpus(sdp) {
  var sdpLines = sdp.split('\r\n');
  var mLineIndex;
  // Search for m line.
  for (var i = 0; i < sdpLines.length; i++) {
    if (sdpLines[i].search('m=audio') !== -1) {
      mLineIndex = i;
      break;
    }
  }
  if (mLineIndex === null) {
    return sdp;
  }

  // If Opus is available, set it as the default in m line.
  for (i = 0; i < sdpLines.length; i++) {
    if (sdpLines[i].search('opus/48000') !== -1) {
      var opusPayload = extractSdp(sdpLines[i], /:(\d+) opus\/48000/i);
      if (opusPayload) {
        sdpLines[mLineIndex] = setDefaultAudioCodec(sdpLines[mLineIndex],
          opusPayload);
      }
      break;
    }
  }

  // Remove CN in m line and sdp.
  sdpLines = removeCN(sdpLines, mLineIndex);

  sdp = sdpLines.join('\r\n');
  return sdp;
}

// Set the selected codec to the first in m line.
function setDefaultAudioCodec(mLine, payload) {
  var elements = mLine.split(' ');
  var newLine = [];
  var index = 0;
  for (var i = 0; i < elements.length; i++) {
    if (index === 3) { // Format of media starts from the fourth.
      newLine[index++] = payload; // Put target payload to the first.
    }
    if (elements[i] !== payload) {
      newLine[index++] = elements[i];
    }
  }
  return newLine.join(' ');
}

// Strip CN from sdp before CN constraints is ready.
function removeCN(sdpLines, mLineIndex) {
  var mLineElements = sdpLines[mLineIndex].split(' ');
  // Scan from end for the convenience of removing an item.
  for (var i = sdpLines.length - 1; i >= 0; i--) {
    var payload = extractSdp(sdpLines[i], /a=rtpmap:(\d+) CN\/\d+/i);
    if (payload) {
      var cnPos = mLineElements.indexOf(payload);
      if (cnPos !== -1) {
        // Remove CN payload from m line.
        mLineElements.splice(cnPos, 1);
      }
      // Remove CN line in sdp
      sdpLines.splice(i, 1);
    }
  }

  sdpLines[mLineIndex] = mLineElements.join(' ');
  return sdpLines;
}