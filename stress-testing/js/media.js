/**
 * Respect Browser Stress Testing Suite - Media, Audio, Video & Sensor Lab
 * Menguji fungsionalitas Camera, Live Microphone Oscilloscope, Web Audio Synth,
 * Video Stream generator mandiri, dan Speech Synthesis / Recognition.
 */

var MediaLabEngine = (function () {
  'use strict';

  var cameraStream = null;
  var micStream = null;
  var micAudioCtx = null;
  var micAnalyser = null;
  var micAnimId = null;

  var synthCtx = null;
  var synthOsc = null;
  var synthGain = null;

  var videoGeneratorAnimId = null;
  var videoGeneratorCanvas = null;

  var virtualCamAnimId = null;
  var virtualCamCanvas = null;
  var virtualCamStream = null;

  // =========================================================================
  // 1. CAMERA TEST & IMAGE PROCESSING FILTERS (100% Crash-Proof Pure Canvas Feed)
  // =========================================================================
  var cameraRunning = false;
  var virtualCamAnimId = null;

  function getCamCanvas() {
    return document.getElementById('cameraVirtualFeed');
  }

  function drawStandbyFrame() {
    var canvas = getCamCanvas();
    if (!canvas) return;
    var ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Dark sleek background
    ctx.fillStyle = '#070b14';
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    // Subtle Grid
    ctx.strokeStyle = '#111e33';
    ctx.lineWidth = 1;
    for (var x = 0; x < canvas.width; x += 32) {
      ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, canvas.height); ctx.stroke();
    }
    for (var y = 0; y < canvas.height; y += 32) {
      ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(canvas.width, y); ctx.stroke();
    }

    // Standby Text
    ctx.font = 'bold 15px -apple-system, sans-serif';
    ctx.fillStyle = '#64748b';
    ctx.textAlign = 'center';
    ctx.fillText('📷 CAMERA FEED STANDBY', canvas.width / 2, canvas.height / 2 - 12);
    ctx.font = '12px monospace';
    ctx.fillStyle = '#475569';
    ctx.fillText('Klik "Buka Kamera" untuk menyalakan video stream simulasi 60 FPS', canvas.width / 2, canvas.height / 2 + 14);
    ctx.textAlign = 'left';
  }

  function initCameraPreview() {
    drawStandbyFrame();
  }

  function startCamera(videoEl, statusEl, callback) {
    if (cameraRunning) {
      // Sudah berjalan: tetap panggil callback agar pemanggil tidak menggantung.
      if (callback) callback(true, { isVirtual: true, alreadyRunning: true, canvas: getCamCanvas() });
      return;
    }
    cameraRunning = true;

    var canvas = getCamCanvas();
    if (!canvas) {
      cameraRunning = false;
      if (statusEl) statusEl.textContent = 'Elemen canvas kamera tidak ditemukan.';
      if (callback) callback(false, new Error('Canvas not found'));
      return;
    }
    var ctx = canvas.getContext('2d');
    if (!ctx) {
      cameraRunning = false;
      if (statusEl) statusEl.textContent = '2D context kamera tidak tersedia.';
      if (callback) callback(false, new Error('Camera 2D context unavailable'));
      return;
    }

    var frame = 0;
    var lastTime = performance.now();
    var fps = 60;
    var scanY = 0;
    var targetX = 320, targetY = 180, tvx = 2.5, tvy = 1.8;

    function renderVirtualCam() {
      if (!cameraRunning) return;
      virtualCamAnimId = requestAnimationFrame(renderVirtualCam);
      frame++;
      var now = performance.now();
      if (frame % 15 === 0) {
        fps = Math.round(1000 / (now - lastTime || 16.6));
      }
      lastTime = now;

      // 1. Dark futuristic background
      ctx.fillStyle = '#070b14';
      ctx.fillRect(0, 0, 640, 360);

      // 2. High-tech Grid
      ctx.strokeStyle = '#111e33';
      ctx.lineWidth = 1;
      for (var x = 0; x < 640; x += 32) {
        ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, 360); ctx.stroke();
      }
      for (var y = 0; y < 360; y += 32) {
        ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(640, y); ctx.stroke();
      }

      // 3. Scanline Laser
      scanY = (scanY + 2) % 360;
      ctx.fillStyle = 'rgba(6, 182, 212, 0.15)';
      ctx.fillRect(0, scanY, 640, 4);

      // 4. Moving Target Box (Simulated Face Tracker)
      targetX += tvx; targetY += tvy;
      if (targetX < 120 || targetX > 520) tvx = -tvx;
      if (targetY < 80 || targetY > 280) tvy = -tvy;

      var boxW = 100, boxH = 120;
      var bx = targetX - boxW / 2;
      var by = targetY - boxH / 2;

      ctx.strokeStyle = '#10b981';
      ctx.lineWidth = 2;
      // Reticle corners
      ctx.beginPath();
      var cl = 15;
      ctx.moveTo(bx, by + cl); ctx.lineTo(bx, by); ctx.lineTo(bx + cl, by);
      ctx.moveTo(bx + boxW - cl, by); ctx.lineTo(bx + boxW, by); ctx.lineTo(bx + boxW, by + cl);
      ctx.moveTo(bx, by + boxH - cl); ctx.lineTo(bx, by + boxH); ctx.lineTo(bx + cl, by + boxH);
      ctx.moveTo(bx + boxW - cl, by + boxH); ctx.lineTo(bx + boxW, by + boxH); ctx.lineTo(bx + boxW, by + boxH - cl);
      ctx.stroke();

      // Center crosshair
      ctx.strokeStyle = 'rgba(16, 185, 129, 0.5)';
      ctx.beginPath();
      ctx.moveTo(targetX - 8, targetY); ctx.lineTo(targetX + 8, targetY);
      ctx.moveTo(targetX, targetY - 8); ctx.lineTo(targetX, targetY + 8);
      ctx.stroke();

      // Tracker telemetry label
      ctx.font = 'bold 11px monospace';
      ctx.fillStyle = '#10b981';
      ctx.fillText('TARGET: FACE_01 [CONF: 99.2%]', bx, by - 6);
      ctx.font = '10px monospace';
      ctx.fillStyle = '#64748b';
      ctx.fillText('POS: ' + Math.round(targetX) + ',' + Math.round(targetY), bx, by + boxH + 14);

      // 5. Color Test Bars (SMPTE miniature)
      var colors = ['#f8fafc', '#eab308', '#06b6d4', '#22c55e', '#ec4899', '#ef4444', '#3b82f6'];
      var barW = 640 / colors.length;
      for (var ci = 0; ci < colors.length; ci++) {
        ctx.fillStyle = colors[ci];
        ctx.fillRect(ci * barW, 336, barW, 24);
      }

      // 6. Top Telemetry Overlay
      ctx.font = 'bold 14px monospace';
      ctx.fillStyle = '#38bdf8';
      ctx.fillText('RESPECT VIRTUAL CAMERA LAB (60 FPS)', 20, 28);

      ctx.font = '12px monospace';
      ctx.fillStyle = '#94a3b8';
      ctx.fillText('TIMESTAMP : ' + (new Date()).toISOString(), 20, 48);
      ctx.fillText('STREAM FPS: ' + fps + ' FPS | RESOLUTION: 1280x720 IDEAL', 20, 66);

      // Watermark indicator
      ctx.font = 'bold 11px monospace';
      ctx.fillStyle = '#f59e0b';
      ctx.fillText('● LIVE TEST FEED', 510, 28);
    }

    renderVirtualCam();

    if (statusEl) {
      statusEl.textContent = 'Kamera Aktif: Virtual Test Stream (Live 60 FPS Simulasi)';
    }
    if (callback) callback(true, { isVirtual: true, canvas: canvas });
  }

  function stopCamera(videoEl, statusEl) {
    cameraRunning = false;
    if (virtualCamAnimId) {
      try {
        cancelAnimationFrame(virtualCamAnimId);
      } catch (e) {}
      virtualCamAnimId = null;
    }

    drawStandbyFrame();

    if (statusEl) {
      statusEl.textContent = 'Kamera Dimatikan (Standby)';
    }
  }

  function snapshotCamera(videoEl, canvasEl, filterType) {
    if (!canvasEl) return;
    var ctx = canvasEl.getContext('2d');
    if (!ctx) return;

    var src = getCamCanvas();
    if (!src) return;

    canvasEl.width = src.width || 640;
    canvasEl.height = src.height || 360;
    ctx.drawImage(src, 0, 0, canvasEl.width, canvasEl.height);

    if (filterType && filterType !== 'none') {
      var imgData = ctx.getImageData(0, 0, canvasEl.width, canvasEl.height);
      var d = imgData.data;

      for (var i = 0; i < d.length; i += 4) {
        var r = d[i], g = d[i + 1], b = d[i + 2];
        if (filterType === 'grayscale') {
          var gray = 0.299 * r + 0.587 * g + 0.114 * b;
          d[i] = d[i + 1] = d[i + 2] = gray;
        } else if (filterType === 'invert') {
          d[i] = 255 - r;
          d[i + 1] = 255 - g;
          d[i + 2] = 255 - b;
        } else if (filterType === 'sepia') {
          d[i] = (r * 0.393) + (g * 0.769) + (b * 0.189);
          d[i + 1] = (r * 0.349) + (g * 0.686) + (b * 0.168);
          d[i + 2] = (r * 0.272) + (g * 0.534) + (b * 0.131);
        }
      }
      ctx.putImageData(imgData, 0, 0);
    }
  }

  // =========================================================================
  // 2. MICROPHONE & LIVE REAL-TIME OSCILLOSCOPE
  // =========================================================================
  function startMicrophone(canvasEl, meterEl, statusEl, callback) {
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      var note = 'Audio capture (getUserMedia) tidak aktif (Origin tidak berada di Secure Context atau dibatasi)';
      if (typeof window.isSecureContext !== 'undefined' && !window.isSecureContext) {
        note += ' [isSecureContext: false]';
      }
      if (statusEl) statusEl.textContent = note;
      if (callback) callback(false, new Error(note));
      return;
    }

    stopMicrophone(null);

    navigator.mediaDevices.getUserMedia({ audio: true, video: false })
      .then(function (stream) {
        micStream = stream;
        var AudioCtx = window.AudioContext || window.webkitAudioContext;
        if (!AudioCtx) {
          if (statusEl) statusEl.textContent = 'AudioContext tidak didukung.';
          return;
        }
        micAudioCtx = new AudioCtx();
        if (typeof micAudioCtx.createAnalyser !== 'function' || typeof micAudioCtx.createMediaStreamSource !== 'function') {
          if (statusEl) statusEl.textContent = 'createAnalyser/createMediaStreamSource tidak tersedia di AudioContext build ini.';
          return;
        }
        micAnalyser = micAudioCtx.createAnalyser();
        micAnalyser.fftSize = 1024;

        var source = micAudioCtx.createMediaStreamSource(stream);
        source.connect(micAnalyser);

        drawMicOscilloscope(canvasEl, meterEl);
        if (statusEl) statusEl.textContent = 'Mikrofon Aktif: Sampling 44.1kHz / 48kHz';
        if (callback) callback(true);
      })
      .catch(function (err) {
        if (statusEl) statusEl.textContent = 'Gagal akses mikrofon: ' + (err.name || 'Error') + ' (' + (err.message || String(err)) + ')';
        if (callback) callback(false, err);
      });
  }

  function drawMicOscilloscope(canvasEl, meterEl) {
    if (!micAnalyser || !canvasEl) return;
    var ctx = canvasEl.getContext('2d');
    var bufferLength = micAnalyser.fftSize;
    var dataArray = new Uint8Array(bufferLength);

    function render() {
      if (!micAnalyser) return;
      micAnimId = requestAnimationFrame(render);
      micAnalyser.getByteTimeDomainData(dataArray);

      ctx.fillStyle = '#070b12';
      ctx.fillRect(0, 0, canvasEl.width, canvasEl.height);
      ctx.lineWidth = 2;
      ctx.strokeStyle = '#06b6d4';
      ctx.beginPath();

      var sliceWidth = canvasEl.width / bufferLength;
      var x = 0;
      var sumSquares = 0.0;

      for (var i = 0; i < bufferLength; i++) {
        var v = dataArray[i] / 128.0;
        var y = (v * canvasEl.height) / 2;
        var deviation = (dataArray[i] - 128) / 128;
        sumSquares += deviation * deviation;

        if (i === 0) {
          ctx.moveTo(x, y);
        } else {
          ctx.lineTo(x, y);
        }
        x += sliceWidth;
      }

      ctx.lineTo(canvasEl.width, canvasEl.height / 2);
      ctx.stroke();

      // RMS Decibel estimation for VU meter
      var rms = Math.sqrt(sumSquares / bufferLength);
      var percent = Math.min(100, Math.round(rms * 250));
      if (meterEl) {
        meterEl.style.width = percent + '%';
      }
    }

    render();
  }

  function stopMicrophone(statusEl) {
    if (micAnimId) {
      try {
        cancelAnimationFrame(micAnimId);
      } catch (e) {}
      micAnimId = null;
    }
    if (micStream) {
      try {
        var tracks = micStream.getTracks ? micStream.getTracks() : [];
        for (var i = 0; i < tracks.length; i++) {
          try {
            if (tracks[i] && typeof tracks[i].stop === 'function') {
              tracks[i].stop();
            }
          } catch (te) {}
        }
      } catch (se) {}
      micStream = null;
    }
    if (micAudioCtx) {
      try {
        micAudioCtx.close();
      } catch (e) {}
      micAudioCtx = null;
    }
    micAnalyser = null;
    if (statusEl) {
      try {
        statusEl.textContent = 'Mikrofon Dimatikan (Standby)';
      } catch (e) {}
    }
  }

  // =========================================================================
  // 3. WEB AUDIO HARMONIC SYNTHESIZER (No file dependency)
  // =========================================================================
  function startSynth(freqHz, waveType, gainVal, statusEl) {
    stopSynth(null);

    try {
      var AudioCtx = window.AudioContext || window.webkitAudioContext;
      if (!AudioCtx) {
        if (statusEl) statusEl.textContent = 'Web Audio API tidak didukung oleh engine ini.';
        return;
      }
      synthCtx = new AudioCtx();
      if (typeof synthCtx.createOscillator !== 'function') {
        if (statusEl) statusEl.textContent = 'createOscillator() tidak tersedia di AudioContext build ini.';
        return;
      }
      synthOsc = synthCtx.createOscillator();
      synthGain = synthCtx.createGain ? synthCtx.createGain() : null;

      synthOsc.type = waveType || 'sine';
      if (synthOsc.frequency && typeof synthOsc.frequency.setValueAtTime === 'function') {
        synthOsc.frequency.setValueAtTime(freqHz || 440, synthCtx.currentTime);
      }
      if (synthGain && synthGain.gain && typeof synthGain.gain.setValueAtTime === 'function') {
        synthGain.gain.setValueAtTime(gainVal !== undefined ? gainVal : 0.2, synthCtx.currentTime);
        synthOsc.connect(synthGain);
        synthGain.connect(synthCtx.destination);
      } else {
        synthOsc.connect(synthCtx.destination);
      }
      synthOsc.start();

      if (statusEl) statusEl.textContent = 'Synthesizer Aktif: ' + freqHz + ' Hz (' + waveType + ')';
    } catch (e) {
      if (statusEl) statusEl.textContent = 'Synth: ' + e.message;
    }
  }

  function updateSynthFreq(freqHz) {
    if (synthOsc && synthCtx) {
      synthOsc.frequency.setValueAtTime(freqHz, synthCtx.currentTime);
    }
  }

  function stopSynth(statusEl) {
    if (synthOsc) {
      try { synthOsc.stop(); } catch (e) {}
      synthOsc = null;
    }
    if (synthCtx) {
      try { synthCtx.close(); } catch (e) {}
      synthCtx = null;
    }
    if (statusEl) statusEl.textContent = 'Synthesizer Standby';
  }

  // =========================================================================
  // 4. AUTONOMOUS VIDEO PLAYBACK GENERATOR (Zero-Disk / Zero-Network)
  // =========================================================================
  function initVideoGenerator(videoEl, telemetryEl) {
    if (!videoEl) return;

    if (!videoGeneratorCanvas) {
      videoGeneratorCanvas = document.createElement('canvas');
      videoGeneratorCanvas.width = 640;
      videoGeneratorCanvas.height = 360;
    }

    var ctx = videoGeneratorCanvas.getContext('2d');
    var ballX = 100, ballY = 100, vx = 5, vy = 3;
    var angle = 0;

    function renderFrame() {
      videoGeneratorAnimId = requestAnimationFrame(renderFrame);

      // Background
      ctx.fillStyle = '#0f172a';
      ctx.fillRect(0, 0, 640, 360);

      // Grid
      ctx.strokeStyle = '#1e293b';
      ctx.lineWidth = 1;
      for (var x = 0; x < 640; x += 40) {
        ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, 360); ctx.stroke();
      }
      for (var y = 0; y < 360; y += 40) {
        ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(640, y); ctx.stroke();
      }

      // Rotating Star / Polygon
      ctx.save();
      ctx.translate(320, 180);
      angle += 0.03;
      ctx.rotate(angle);
      ctx.fillStyle = '#3b82f6';
      ctx.fillRect(-50, -50, 100, 100);
      ctx.restore();

      // Bouncing Ball
      ballX += vx; ballY += vy;
      if (ballX < 20 || ballX > 620) vx = -vx;
      if (ballY < 20 || ballY > 340) vy = -vy;

      ctx.fillStyle = '#f59e0b';
      ctx.beginPath();
      ctx.arc(ballX, ballY, 18, 0, Math.PI * 2);
      ctx.fill();

      // Telemetry Text & Clock
      ctx.font = 'bold 20px monospace';
      ctx.fillStyle = '#10b981';
      ctx.fillText('RESPECT VIDEO ENGINE 60 FPS', 30, 40);
      ctx.font = '14px monospace';
      ctx.fillStyle = '#94a3b8';
      ctx.fillText('TIMESTAMP: ' + (new Date()).toISOString(), 30, 70);
      ctx.fillText('CANVAS STREAM -> MEDIASTREAM PIPELINE', 30, 95);
    }

    renderFrame();

    // Stream generation
    if (typeof videoGeneratorCanvas.captureStream === 'function') {
      var stream = videoGeneratorCanvas.captureStream(30);
      videoEl.srcObject = stream;
      videoEl.play().catch(function () {});

      if (telemetryEl) {
        telemetryEl.textContent = 'Stream Status: Active 30fps (Zero-File Canvas Stream)';
      }
    } else {
      if (telemetryEl) {
        telemetryEl.textContent = 'captureStream() tidak didukung oleh browser ini.';
      }
    }
  }

  // =========================================================================
  // 5. SPEECH SYNTHESIS (TTS)
  // =========================================================================
  function speakText(text, pitch, rate, statusEl) {
    if (!('speechSynthesis' in window)) {
      if (statusEl) statusEl.textContent = 'Speech Synthesis API tidak didukung.';
      return;
    }
    window.speechSynthesis.cancel();
    var utterance = new SpeechSynthesisUtterance(text || 'Respect Desktop test.');
    utterance.pitch = pitch || 1.0;
    utterance.rate = rate || 1.0;
    utterance.onend = function () {
      if (statusEl) statusEl.textContent = 'Pengucapan Suara Selesai.';
    };
    utterance.onerror = function (e) {
      if (statusEl) statusEl.textContent = 'TTS Error: ' + e.error;
    };
    window.speechSynthesis.speak(utterance);
    if (statusEl) statusEl.textContent = 'Sedang berbicara...';
  }

  function stopSpeech() {
    if ('speechSynthesis' in window) {
      window.speechSynthesis.cancel();
    }
  }

  return {
    initCameraPreview: initCameraPreview,
    startCamera: startCamera,
    stopCamera: stopCamera,
    snapshotCamera: snapshotCamera,
    startMicrophone: startMicrophone,
    stopMicrophone: stopMicrophone,
    startSynth: startSynth,
    updateSynthFreq: updateSynthFreq,
    stopSynth: stopSynth,
    initVideoGenerator: initVideoGenerator,
    speakText: speakText,
    stopSpeech: stopSpeech
  };
})();

if (typeof window !== 'undefined') {
  window.MediaLabEngine = MediaLabEngine;
}
