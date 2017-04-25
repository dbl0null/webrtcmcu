//一些舍不得删又没啥用的代码放这里先
/////////////////////////////////////////////////////////////////////////////
function preferH264(sdp) {
  var sdpLines = sdp.split('\r\n');
  var mLineIndex;
  // Search for m line.
  for (var i = 0; i < sdpLines.length; i++) {
    if (sdpLines[i].search('m=video') !== -1) {
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


function removeVideoCodecByName(sdpLines, mLineIndex, targetName) {
  var targetPayload = removeRtpmapByName(sdpLines, mLineIndex, targetName);
  if (targetPayload) {
    removeRtcpfbByPayload(sdpLines, mLineIndex, targetPayload);
    removeFmtpByPayload(sdpLines, mLineIndex, targetPayload);
    var associatedPayload = removeFmtpAptByPayload(sdpLines, mLineIndex, targetPayload)
    if (associatedPayload) {
      removeRtmpByPayload(sdpLines, mLineIndex, associatedPayload)
    }
  }
}

function removeRtmpByPayload(sdpLines, mLineIndex, targetPayload) {
  var mLineElements = sdpLines[mLineIndex].split(' ');
  var regularEq = new RegExp('a=rtpmap:' + targetPayload + ' ', 'i');

  for (var i = sdpLines.length - 1; i >= 0; i--) {
    if (sdpLines[i].search(regularEq) != -1) {
      var cnPos = mLineElements.indexOf(targetPayload);
      if (cnPos !== -1) {
        mLineElements.splice(cnPos, 1);
      }
      sdpLines.splice(i, 1);
    }
  }

  sdpLines[mLineIndex] = mLineElements.join(' ');
}

function removeFmtpAptByPayload(sdpLines, mLineIndex, targetPayload) {
  var associatedPayload = null;
  var regularEq = new RegExp('a=fmtp:[0-9]+ apt=' + targetPayload, 'i');

  for (var i = sdpLines.length - 1; i >= 0; i--) {
    if (sdpLines[i].search(regularEq) != -1) {
      associatedPayload = sdpLines[i].split(/[:| ]/)[1];
      sdpLines.splice(i, 1);
    }
  }

  return associatedPayload;
}

function removeFmtpByPayload(sdpLines, mLineIndex, targetPayload) {
  for (var i = sdpLines.length - 1; i >= 0; i--) {
    if (sdpLines[i].search('a=fmtp:' + targetPayload) != -1)
      sdpLines.splice(i, 1);
  }
}

function removeRtcpfbByPayload(sdpLines, mLineIndex, targetPayload) {
  for (var i = sdpLines.length - 1; i >= 0; i--) {
    if (sdpLines[i].search('a=rtcp-fb:' + targetPayload) != -1)
      sdpLines.splice(i, 1);
  }
}

function removeRtpmapByName(sdpLines, mLineIndex, targetName) {
  var mLineElements = sdpLines[mLineIndex].split(' ');
  var regularEq = new RegExp('a=rtpmap:(\\d+) ' + targetName + '\/\\d+', 'i');

  var targetPayload = null;
  for (var i = sdpLines.length - 1; i >= 0; i--) {
    var payload = extractSdp(sdpLines[i], regularEq);

    if (payload) {
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


function extractSdp(sdpLine, pattern) {
  var result = sdpLine.match(pattern);
  return result && result.length === 2 ? result[1] : null;
}


// Set the selected codec to the first in m line.
function setDefaultCodec(mLine, payload) {
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