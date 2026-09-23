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

  // =========================================================================
  // 1. CAMERA TEST & IMAGE PROCESSING FILTERS
  // =========================================================================
  function startCamera(videoEl, statusEl, callback) {
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      if (statusEl) statusEl.textContent = 'getUserMedia tidak didukung oleh browser ini.';
      return;
    }

    stopCamera();

    navigator.mediaDevices.getUserMedia({
      video: { width: { ideal: 1280 }, height: { ideal: 720 } },
      audio: false
    }).then(function (stream) {
      cameraStream = stream;
      if (videoEl) {
        videoEl.srcObject = stream;
        videoEl.play();
      }
      if (statusEl) statusEl.textContent = 'Kamera Aktif: 1280x720 (Live Stream)';
      if (callback) callback(true, stream);
    }).catch(function (err) {
      if (statusEl) statusEl.textContent = 'Gagal mengakses kamera: ' + err.name + ' (' + err.message + ')';
      if (callback) callback(false, err);
    });
  }

  function stopCamera(videoEl, statusEl) {
    if (cameraStream) {
      cameraStream.getTracks().forEach(function (track) { track.stop(); });
      cameraStream = null;
    }
    if (videoEl) videoEl.srcObject = null;
    if (statusEl) statusEl.textContent = 'Kamera Dimatikan';
  }

  function snapshotCamera(videoEl, canvasEl, filterType) {
    if (!videoEl || !canvasEl) return;
    var ctx = canvasEl.getContext('2d');
    if (!ctx) return;

    canvasEl.width = videoEl.videoWidth || 640;
    canvasEl.height = videoEl.videoHeight || 480;
    ctx.drawImage(videoEl, 0, 0, canvasEl.width, canvasEl.height);

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
      if (statusEl) statusEl.textContent = 'Audio capture tidak didukung.';
      return;
    }

    stopMicrophone();

    navigator.mediaDevices.getUserMedia({ audio: true, video: false })
      .then(function (stream) {
        micStream = stream;
        var AudioCtx = window.AudioContext || window.webkitAudioContext;
        micAudioCtx = new AudioCtx();
        micAnalyser = micAudioCtx.createAnalyser();
        micAnalyser.fftSize = 1024;

        var source = micAudioCtx.createMediaStreamSource(stream);
        source.connect(micAnalyser);

        drawMicOscilloscope(canvasEl, meterEl);
        if (statusEl) statusEl.textContent = 'Mikrofon Aktif: Sampling 44.1kHz / 48kHz';
        if (callback) callback(true);
      })
      .catch(function (err) {
        if (statusEl) statusEl.textContent = 'Gagal akses mikrofon: ' + err.message;
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
    if (micAnimId) cancelAnimationFrame(micAnimId);
    if (micStream) {
      micStream.getTracks().forEach(function (track) { track.stop(); });
      micStream = null;
    }
    if (micAudioCtx) {
      try { micAudioCtx.close(); } catch (e) {}
      micAudioCtx = null;
    }
    micAnalyser = null;
    if (statusEl) statusEl.textContent = 'Mikrofon Dimatikan';
  }

  // =========================================================================
  // 3. WEB AUDIO HARMONIC SYNTHESIZER (No file dependency)
  // =========================================================================
  function startSynth(freqHz, waveType, gainVal, statusEl) {
    stopSynth();

    try {
      var AudioCtx = window.AudioContext || window.webkitAudioContext;
      synthCtx = new AudioCtx();
      synthOsc = synthCtx.createOscillator();
      synthGain = synthCtx.createGain();

      synthOsc.type = waveType || 'sine';
      synthOsc.frequency.setValueAtTime(freqHz || 440, synthCtx.currentTime);
      synthGain.gain.setValueAtTime(gainVal !== undefined ? gainVal : 0.2, synthCtx.currentTime);

      synthOsc.connect(synthGain);
      synthGain.connect(synthCtx.destination);
      synthOsc.start();

      if (statusEl) statusEl.textContent = 'Synthesizer Aktif: ' + freqHz + ' Hz (' + waveType + ')';
    } catch (e) {
      if (statusEl) statusEl.textContent = 'Synth Error: ' + e.message;
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
