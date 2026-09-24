/**
 * Respect Browser Stress Testing Suite - System & Hardware Profile Engine
 * Mengaudit spesifikasi perangkat keras, unmasked GPU, audio/canvas fingerprint,
 * probe font sistem, deteksi codec multimedia, dan kuota penyimpanan.
 * 
 * Catatan Arsitektur: Berkas ini dinamai sysprofile.js (bukan fingerprint.js)
 * untuk mencegah browser extensions (uBlock Origin, AdBlock, Brave Shields) memblokir
 * request dengan status net::ERR_BLOCKED_BY_CLIENT karena pola heuristik EasyPrivacy.
 */

var SysProfileEngine = (function () {
  'use strict';

  function fnv1aHash(str) {
    var hash = 2166136261;
    for (var i = 0; i < str.length; i++) {
      hash ^= str.charCodeAt(i);
      hash += (hash << 1) + (hash << 4) + (hash << 7) + (hash << 8) + (hash << 24);
    }
    return (hash >>> 0).toString(16).toUpperCase();
  }

  // 1. Browser & Navigation Environment
  function getEnvironment() {
    var nav = navigator || {};
    var isRespectModern = typeof window.mbQuery === 'function';
    var isRespectLite = typeof window.ipc !== 'undefined' && window.ipc && typeof window.ipc.invoke === 'function';

    var engineFlavor = 'Standard Browser';
    if (isRespectModern) {
      engineFlavor = 'Respect Modern (Chromium 132 Core)';
    } else if (isRespectLite) {
      engineFlavor = 'Respect Lite (Miniblink 49 Core)';
    }

    return {
      engineFlavor: engineFlavor,
      userAgent: nav.userAgent || 'Unknown',
      appVersion: nav.appVersion || 'Unknown',
      platform: nav.platform || 'Unknown',
      vendor: nav.vendor || 'Unknown',
      language: nav.language || 'Unknown',
      languages: nav.languages ? Array.prototype.slice.call(nav.languages) : [nav.language],
      cookieEnabled: !!nav.cookieEnabled,
      onLine: nav.onLine !== undefined ? nav.onLine : true,
      doNotTrack: nav.doNotTrack || window.doNotTrack || 'unspecified',
      maxTouchPoints: nav.maxTouchPoints || 0,
      pdfViewerEnabled: !!nav.pdfViewerEnabled,
      isSecureContext: !!window.isSecureContext,
      respectHooks: {
        mbQuery: isRespectModern,
        ipc: isRespectLite
      }
    };
  }

  // 2. Hardware & Screen Telemetry
  function getHardwareSpecs(callback) {
    var nav = navigator || {};
    var scr = window.screen || {};

    var specs = {
      cpuCores: nav.hardwareConcurrency || 'N/A',
      deviceMemoryGB: nav.deviceMemory ? (nav.deviceMemory + ' GB') : 'N/A',
      screenWidth: scr.width || window.innerWidth,
      screenHeight: scr.height || window.innerHeight,
      availWidth: scr.availWidth || scr.width,
      availHeight: scr.availHeight || scr.height,
      colorDepth: scr.colorDepth || 24,
      pixelRatio: window.devicePixelRatio || 1,
      windowInner: window.innerWidth + 'x' + window.innerHeight,
      windowOuter: window.outerWidth + 'x' + window.outerHeight,
      network: {
        type: 'N/A',
        effectiveType: 'N/A',
        downlinkMbps: 'N/A',
        rttMs: 'N/A',
        saveData: false
      },
      battery: {
        supported: false,
        level: 'N/A',
        charging: false
      }
    };

    // Network Information API
    var conn = nav.connection || nav.mozConnection || nav.webkitConnection;
    if (conn) {
      specs.network.type = conn.type || 'unknown';
      specs.network.effectiveType = conn.effectiveType || 'unknown';
      specs.network.downlinkMbps = conn.downlink !== undefined ? conn.downlink : 'N/A';
      specs.network.rttMs = conn.rtt !== undefined ? conn.rtt : 'N/A';
      specs.network.saveData = !!conn.saveData;
    }

    // Battery Status API (Async)
    if (typeof nav.getBattery === 'function') {
      nav.getBattery().then(function (batt) {
        specs.battery.supported = true;
        specs.battery.level = Math.round(batt.level * 100) + '%';
        specs.battery.charging = batt.charging;
        if (callback) callback(specs);
      }).catch(function () {
        if (callback) callback(specs);
      });
    } else {
      if (callback) callback(specs);
    }

    return specs;
  }

  // 3. WebGL Hardware & Unmasked GPU Probing
  function getWebGLFingerprint() {
    var canvas = document.createElement('canvas');
    var gl = null;
    var isWebGL2 = false;

    try {
      gl = canvas.getContext('webgl2');
      if (gl) isWebGL2 = true;
    } catch (e) {}

    if (!gl) {
      try {
        gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
      } catch (e) {}
    }

    if (!gl) {
      return {
        supported: false,
        error: 'WebGL not supported on this engine context.'
      };
    }

    var result = {
      supported: true,
      version: isWebGL2 ? 'WebGL 2.0' : 'WebGL 1.0',
      vendor: gl.getParameter(gl.VENDOR),
      renderer: gl.getParameter(gl.RENDERER),
      shadingLanguage: gl.getParameter(gl.SHADING_LANGUAGE_VERSION),
      unmaskedVendor: 'Unknown',
      unmaskedRenderer: 'Unknown',
      maxTextureSize: gl.getParameter(gl.MAX_TEXTURE_SIZE),
      maxCubeMapSize: gl.getParameter(gl.MAX_CUBE_MAP_TEXTURE_SIZE),
      maxRenderbufferSize: gl.getParameter(gl.MAX_RENDERBUFFER_SIZE),
      maxViewportDims: gl.getParameter(gl.MAX_VIEWPORT_DIMS),
      samples: gl.getParameter(gl.SAMPLES) || 0,
      extensionsCount: 0,
      extensions: []
    };

    // Unmask true GPU hardware via WEBGL_debug_renderer_info
    var debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
    if (debugInfo) {
      result.unmaskedVendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) || 'N/A';
      result.unmaskedRenderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || 'N/A';
    }

    var exts = gl.getSupportedExtensions();
    if (exts) {
      result.extensionsCount = exts.length;
      result.extensions = Array.prototype.slice.call(exts);
    }

    return result;
  }

  // 4. Complex Canvas Fingerprint
  function getCanvasFingerprint() {
    try {
      var canvas = document.createElement('canvas');
      canvas.width = 280;
      canvas.height = 60;
      var ctx = canvas.getContext('2d');
      if (!ctx) return { supported: false, hash: 'NO_2D_CTX' };

      // Background with gradient
      var grad = ctx.createLinearGradient(0, 0, canvas.width, canvas.height);
      grad.addColorStop(0, '#f97316');
      grad.addColorStop(0.5, '#06b6d4');
      grad.addColorStop(1, '#8b5cf6');
      ctx.fillStyle = grad;
      ctx.fillRect(0, 0, canvas.width, canvas.height);

      // Bezier curve & arcs with alpha blending
      ctx.globalCompositeOperation = 'multiply';
      ctx.fillStyle = 'rgba(255, 255, 0, 0.7)';
      ctx.beginPath();
      ctx.arc(40, 30, 25, 0, Math.PI * 2, true);
      ctx.closePath();
      ctx.fill();

      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 3;
      ctx.beginPath();
      ctx.moveTo(80, 50);
      ctx.bezierCurveTo(90, 10, 140, 10, 160, 50);
      ctx.stroke();

      // Render typography and emojis
      ctx.globalCompositeOperation = 'source-over';
      ctx.font = '14px "Segoe UI", Arial, sans-serif';
      ctx.fillStyle = '#0f172a';
      ctx.fillText('Respect Browser Suite ⚡', 95, 25);

      ctx.font = '16px serif';
      ctx.fillText('🦄🔥🚀🎮', 100, 48);

      var dataURL = canvas.toDataURL();
      var hash = fnv1aHash(dataURL);

      return {
        supported: true,
        hash: hash,
        dataLength: dataURL.length,
        previewData: dataURL
      };
    } catch (e) {
      return { supported: false, hash: 'ERROR: ' + e.message };
    }
  }

  // 5. AudioContext Fingerprint
  function getAudioFingerprint(callback) {
    try {
      var AudioContextClass = window.OfflineAudioContext || window.webkitOfflineAudioContext;
      if (!AudioContextClass) {
        if (callback) callback({ supported: false, hash: 'UNSUPPORTED' });
        return;
      }

      var ctx = new AudioContextClass(1, 44100, 44100);
      if (typeof ctx.createOscillator !== 'function' || typeof ctx.createDynamicsCompressor !== 'function') {
        if (callback) callback({ supported: false, hash: 'MOCK_AUDIO_CTX (No Oscillator/Compressor)' });
        return;
      }

      var osc = ctx.createOscillator();
      osc.type = 'triangle';
      osc.frequency.setValueAtTime(10000, ctx.currentTime);

      var compressor = ctx.createDynamicsCompressor();
      compressor.threshold.setValueAtTime(-50, ctx.currentTime);
      compressor.knee.setValueAtTime(40, ctx.currentTime);
      compressor.ratio.setValueAtTime(12, ctx.currentTime);
      compressor.attack.setValueAtTime(0, ctx.currentTime);
      compressor.release.setValueAtTime(0.25, ctx.currentTime);

      osc.connect(compressor);
      compressor.connect(ctx.destination);
      osc.start(0);

      ctx.oncomplete = function (e) {
        var samples = e.renderedBuffer.getChannelData(0);
        var str = '';
        for (var i = 4500; i < 5000; i++) {
          str += samples[i].toString();
        }
        var hash = fnv1aHash(str);
        if (callback) callback({ supported: true, hash: hash, sampleCount: samples.length });
      };

      ctx.startRendering();
    } catch (err) {
      if (callback) callback({ supported: false, hash: 'UNSUPPORTED: ' + err.message });
    }
  }

  // 6. System Installed Fonts Probe (50+ Common Windows & Web Fonts)
  function probeSystemFonts() {
    var fontList = [
      'Arial', 'Arial Black', 'Calibri', 'Cambria', 'Consolas', 'Courier New',
      'Georgia', 'Impact', 'Lucida Console', 'Lucida Sans Unicode', 'Microsoft Sans Serif',
      'MS Gothic', 'MS PGothic', 'Palatino Linotype', 'Segoe UI', 'Segoe UI Emoji',
      'Segoe UI Symbol', 'Tahoma', 'Times New Roman', 'Trebuchet MS', 'Verdana',
      'Wingdings', 'Comic Sans MS', 'Franklin Gothic Medium', 'Century Gothic',
      'Sitka Small', 'Malgun Gothic', 'Meiryo', 'Yu Gothic', 'SimHei', 'Microsoft YaHei',
      'Roboto', 'Open Sans', 'Helvetica', 'Helvetica Neue', 'Menlo', 'Monaco'
    ];

    var canvas = document.createElement('canvas');
    var ctx = canvas.getContext('2d');
    if (!ctx) return { supported: false, detectedFonts: [] };

    var testString = 'mmmmmmmmmmlliWWWWWWWW12345!@#$%^&*()';
    var testSize = '72px';
    var baselines = ['monospace', 'sans-serif', 'serif'];

    // Measure base widths
    var baseWidths = {};
    for (var b = 0; b < baselines.length; b++) {
      ctx.font = testSize + ' ' + baselines[b];
      baseWidths[baselines[b]] = ctx.measureText(testString).width;
    }

    var detected = [];
    for (var f = 0; f < fontList.length; f++) {
      var fontName = fontList[f];
      var isMatched = false;

      for (var k = 0; k < baselines.length; k++) {
        ctx.font = testSize + ' "' + fontName + '",' + baselines[k];
        var width = ctx.measureText(testString).width;
        if (width !== baseWidths[baselines[k]]) {
          isMatched = true;
          break;
        }
      }

      if (isMatched) {
        detected.push(fontName);
      }
    }

    return {
      supported: true,
      totalTested: fontList.length,
      detectedCount: detected.length,
      detectedFonts: detected
    };
  }

  // 7. Multimedia Codecs Support Matrix
  function probeMediaCodecs() {
    var v = document.createElement('video');
    var a = document.createElement('audio');

    var videoTests = [
      { name: 'H.264 (Baseline MP4)', mime: 'video/mp4; codecs="avc1.42E01E"' },
      { name: 'H.264 (High Profile MP4)', mime: 'video/mp4; codecs="avc1.640028"' },
      { name: 'H.265 / HEVC', mime: 'video/mp4; codecs="hev1.1.6.L93.B0"' },
      { name: 'VP8 (WebM)', mime: 'video/webm; codecs="vp8"' },
      { name: 'VP9 (WebM/MP4)', mime: 'video/webm; codecs="vp9"' },
      { name: 'AV1 (AOMedia Video 1)', mime: 'video/mp4; codecs="av01.0.08M.08"' },
      { name: 'Ogg Theora', mime: 'video/ogg; codecs="theora"' }
    ];

    var audioTests = [
      { name: 'AAC (mp4a.40.2)', mime: 'audio/mp4; codecs="mp4a.40.2"' },
      { name: 'MP3 (MPEG Audio)', mime: 'audio/mpeg' },
      { name: 'Opus (Ogg)', mime: 'audio/ogg; codecs="opus"' },
      { name: 'Opus (WebM)', mime: 'audio/webm; codecs="opus"' },
      { name: 'FLAC (Free Lossless Audio)', mime: 'audio/flac' },
      { name: 'WAV (PCM Audio)', mime: 'audio/wav; codecs="1"' },
      { name: 'Ogg Vorbis', mime: 'audio/ogg; codecs="vorbis"' }
    ];

    var results = { video: [], audio: [] };

    for (var i = 0; i < videoTests.length; i++) {
      var item = videoTests[i];
      var level = v.canPlayType ? v.canPlayType(item.mime) : 'no';
      results.video.push({
        name: item.name,
        mime: item.mime,
        support: level || 'no',
        pass: level === 'probably' || level === 'maybe'
      });
    }

    for (var j = 0; j < audioTests.length; j++) {
      var aItem = audioTests[j];
      var aLevel = a.canPlayType ? a.canPlayType(aItem.mime) : 'no';
      results.audio.push({
        name: aItem.name,
        mime: aItem.mime,
        support: aLevel || 'no',
        pass: aLevel === 'probably' || aLevel === 'maybe'
      });
    }

    return results;
  }

  // 8. Storage Quota & Subsystems
  function probeStorage(callback) {
    var res = {
      localStorage: false,
      sessionStorage: false,
      indexedDB: false,
      cacheAPI: 'caches' in window,
      opfs: false,
      quotaEstimate: 'N/A',
      usageEstimate: 'N/A'
    };

    try {
      localStorage.setItem('__respect_test__', '1');
      res.localStorage = localStorage.getItem('__respect_test__') === '1';
      localStorage.removeItem('__respect_test__');
    } catch (e) {}

    try {
      sessionStorage.setItem('__respect_test__', '1');
      res.sessionStorage = sessionStorage.getItem('__respect_test__') === '1';
      sessionStorage.removeItem('__respect_test__');
    } catch (e) {}

    res.indexedDB = !!(window.indexedDB || window.webkitIndexedDB || window.mozIndexedDB);
    res.opfs = !!(navigator.storage && typeof navigator.storage.getDirectory === 'function');

    if (navigator.storage && typeof navigator.storage.estimate === 'function') {
      navigator.storage.estimate().then(function (estimate) {
        if (estimate.quota) {
          res.quotaEstimate = (estimate.quota / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
        }
        if (estimate.usage !== undefined) {
          res.usageEstimate = (estimate.usage / (1024 * 1024)).toFixed(2) + ' MB';
        }
        if (callback) callback(res);
      }).catch(function () {
        if (callback) callback(res);
      });
    } else {
      if (callback) callback(res);
    }

    return res;
  }

  return {
    getEnvironment: getEnvironment,
    getHardwareSpecs: getHardwareSpecs,
    getWebGLFingerprint: getWebGLFingerprint,
    getCanvasFingerprint: getCanvasFingerprint,
    getAudioFingerprint: getAudioFingerprint,
    probeSystemFonts: probeSystemFonts,
    probeMediaCodecs: probeMediaCodecs,
    probeStorage: probeStorage
  };
})();

// Alias FingerprintEngine to SysProfileEngine for backward compatibility
var FingerprintEngine = SysProfileEngine;
if (typeof window !== 'undefined') {
  window.SysProfileEngine = SysProfileEngine;
  window.FingerprintEngine = SysProfileEngine;
}
