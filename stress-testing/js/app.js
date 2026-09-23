/**
 * Respect Browser Stress Testing Suite - Master App Controller
 * Mengintegrasikan seluruh modul: Telemetry HUD, API Dump explorer,
 * Capability Matrix, Media Lab, Stress Benchmarks, dan Ekspor Laporan.
 */

(function () {
  'use strict';

  var auditReport = {
    timestamp: new Date().toISOString(),
    environment: {},
    hardware: {},
    webgl: {},
    fingerprints: {},
    capabilities: {},
    apiDump: {},
    benchmarks: {}
  };

  // =========================================================================
  // LOGGING SYSTEM
  // =========================================================================
  function log(msg, type) {
    type = type || 'info';
    var terminal = document.getElementById('terminalBody');
    if (!terminal) return;

    var line = document.createElement('div');
    line.className = 'log-line ' + type;

    var time = document.createElement('span');
    time.className = 'time';
    var now = new Date();
    time.textContent = '[' + now.toTimeString().split(' ')[0] + '.' + String(now.getMilliseconds()).padStart(3, '0') + ']';

    var text = document.createElement('span');
    text.className = 'msg';
    text.textContent = (type === 'ok' ? '✓ ' : type === 'fail' ? '✗ ' : 'ℹ ') + msg;

    line.appendChild(time);
    line.appendChild(text);
    terminal.appendChild(line);
    terminal.scrollTop = terminal.scrollHeight;
  }

  window.clearTerminal = function () {
    var terminal = document.getElementById('terminalBody');
    if (terminal) terminal.innerHTML = '';
  };

  // =========================================================================
  // TAB NAVIGATION
  // =========================================================================
  function initTabs() {
    var tabs = document.querySelectorAll('.tab-btn');
    var panes = document.querySelectorAll('.tab-pane');

    tabs.forEach(function (tab) {
      tab.addEventListener('click', function () {
        var targetId = tab.getAttribute('data-tab');

        tabs.forEach(function (t) { t.classList.remove('active'); });
        panes.forEach(function (p) { p.classList.remove('active'); });

        tab.classList.add('active');
        var targetPane = document.getElementById(targetId);
        if (targetPane) targetPane.classList.add('active');

        // Lazy init when switching
        if (targetId === 'tab-stress') {
          var canvas = document.getElementById('particleCanvas');
          if (canvas) {
            StressBenchmarkEngine.initParticleCanvas(canvas, 5000);
            StressBenchmarkEngine.startParticleSimulation(function (fpsData) {
              var meter = document.getElementById('fpsValue');
              if (meter) meter.textContent = fpsData.fps;
              var minMeter = document.getElementById('minFpsValue');
              if (minMeter) minMeter.textContent = fpsData.minFps;
            });
          }
        }
      });
    });
  }

  // =========================================================================
  // HUD TELEMETRY UPDATE
  // =========================================================================
  function initTelemetry() {
    var env = FingerprintEngine.getEnvironment();
    auditReport.environment = env;

    var engineBadge = document.getElementById('hudEngineBadge');
    if (engineBadge) {
      engineBadge.textContent = env.engineFlavor;
      if (env.respectHooks.mbQuery) {
        engineBadge.classList.add('ok');
      }
    }

    var platformEl = document.getElementById('hudPlatform');
    if (platformEl) platformEl.textContent = env.platform;

    var coresEl = document.getElementById('hudCores');
    var ramEl = document.getElementById('hudRam');

    FingerprintEngine.getHardwareSpecs(function (specs) {
      auditReport.hardware = specs;
      if (coresEl) coresEl.textContent = specs.cpuCores + ' Threads';
      if (ramEl) ramEl.textContent = specs.deviceMemoryGB;

      var resEl = document.getElementById('hudResolution');
      if (resEl) resEl.textContent = specs.screenWidth + 'x' + specs.screenHeight + ' (@' + specs.pixelRatio + 'x)';
    });

    log('Engine terdeteksi: ' + env.engineFlavor, 'ok');
    log('Platform OS: ' + env.platform + ' | User-Agent: ' + env.userAgent.substring(0, 65) + '...', 'info');
  }

  // =========================================================================
  // CAPABILITY MATRIX POPULATION
  // =========================================================================
  function runCapabilities() {
    log('Menjalankan 120+ Web Standards Capability Test...', 'info');
    var cap = CapabilityEngine.runAllTests();
    auditReport.capabilities = cap;

    // Update Dashboard Tiles
    var scoreTile = document.getElementById('totalScoreVal');
    if (scoreTile) scoreTile.textContent = cap.percentage + '%';

    var passedTile = document.getElementById('totalPassedVal');
    if (passedTile) passedTile.textContent = cap.passed + ' / ' + cap.total;

    var meterFill = document.getElementById('scoreMeterFill');
    if (meterFill) {
      meterFill.style.width = cap.percentage + '%';
      if (parseFloat(cap.percentage) >= 75) {
        meterFill.className = 'meter-fill green';
      }
    }

    // Populate Table
    var tbody = document.getElementById('capabilityTableBody');
    if (tbody) {
      tbody.innerHTML = '';
      cap.tests.forEach(function (t) {
        var tr = document.createElement('tr');

        var tdCat = document.createElement('td');
        tdCat.innerHTML = '<span class="code-pill">' + t.category + '</span>';

        var tdName = document.createElement('td');
        tdName.innerHTML = '<strong>' + t.name + '</strong>';

        var tdStatus = document.createElement('td');
        tdStatus.innerHTML = t.pass ?
          '<span class="status-badge pass">PASS</span>' :
          '<span class="status-badge fail">FAIL</span>';

        var tdNote = document.createElement('td');
        tdNote.className = 'muted';
        tdNote.textContent = t.note;

        tr.appendChild(tdCat);
        tr.appendChild(tdName);
        tr.appendChild(tdStatus);
        tr.appendChild(tdNote);
        tbody.appendChild(tr);
      });
    }

    log('Capability Test selesai: ' + cap.passed + ' lulus dari ' + cap.total + ' (' + cap.percentage + '%)', 'ok');
  }

  // =========================================================================
  // API DUMP POPULATION
  // =========================================================================
  function populateApiDump(category, query) {
    var filtered = ApiDumpEngine.filterApis(query, category);
    var tbody = document.getElementById('apiDumpTableBody');
    if (!tbody) return;

    tbody.innerHTML = '';
    var maxDisplay = 250; // Cap to prevent table rendering lag
    var displayItems = filtered.slice(0, maxDisplay);

    displayItems.forEach(function (api) {
      var tr = document.createElement('tr');

      var tdName = document.createElement('td');
      tdName.innerHTML = '<code>' + api.name + '</code>' + (api.isRespect ? ' <span class="status-badge info">Respect Hook</span>' : '');

      var tdType = document.createElement('td');
      tdType.innerHTML = '<span class="status-badge ' + (api.isConstructor ? 'pass' : api.isCallable ? 'info' : 'warn') + '">' + api.summary + '</span>';

      var tdOrigin = document.createElement('td');
      tdOrigin.className = 'muted';
      tdOrigin.textContent = api.origin;

      tr.appendChild(tdName);
      tr.appendChild(tdType);
      tr.appendChild(tdOrigin);
      tbody.appendChild(tr);
    });

    var countLabel = document.getElementById('apiFilteredCount');
    if (countLabel) {
      countLabel.textContent = 'Menampilkan ' + displayItems.length + ' dari ' + filtered.length + ' API (Total: ' + ApiDumpEngine.dumpGlobalApis().counts.total + ')';
    }
  }

  function initApiDump() {
    var dump = ApiDumpEngine.dumpGlobalApis();
    auditReport.apiDump = dump;

    var totalApisVal = document.getElementById('totalApisVal');
    if (totalApisVal) totalApisVal.textContent = dump.counts.total;

    var constructorsVal = document.getElementById('totalConstructorsVal');
    if (constructorsVal) constructorsVal.textContent = dump.counts.constructors;

    var functionsVal = document.getElementById('totalFunctionsVal');
    if (functionsVal) functionsVal.textContent = dump.counts.functions;

    var objectsVal = document.getElementById('totalObjectsVal');
    if (objectsVal) objectsVal.textContent = dump.counts.objects;

    populateApiDump('all', '');

    var searchInput = document.getElementById('apiSearchInput');
    var filterSelect = document.getElementById('apiFilterSelect');

    if (searchInput) {
      searchInput.addEventListener('input', function () {
        populateApiDump(filterSelect ? filterSelect.value : 'all', searchInput.value);
      });
    }

    if (filterSelect) {
      filterSelect.addEventListener('change', function () {
        populateApiDump(filterSelect.value, searchInput ? searchInput.value : '');
      });
    }
  }

  // =========================================================================
  // FINGERPRINT LAB POPULATION
  // =========================================================================
  function initFingerprints() {
    // 1. WebGL
    var webgl = FingerprintEngine.getWebGLFingerprint();
    auditReport.webgl = webgl;

    var gpuVendorEl = document.getElementById('gpuVendor');
    var gpuRendererEl = document.getElementById('gpuRenderer');
    var glVersionEl = document.getElementById('glVersion');
    var glExtsEl = document.getElementById('glExtensionsCount');

    if (gpuVendorEl) gpuVendorEl.textContent = webgl.unmaskedVendor || webgl.vendor || 'N/A';
    if (gpuRendererEl) gpuRendererEl.textContent = webgl.unmaskedRenderer || webgl.renderer || 'N/A';
    if (glVersionEl) glVersionEl.textContent = webgl.version || 'Unsupported';
    if (glExtsEl) glExtsEl.textContent = webgl.extensionsCount + ' Extensions';

    // 2. Canvas
    var canvasFp = FingerprintEngine.getCanvasFingerprint();
    auditReport.fingerprints.canvas = canvasFp;

    var canvasHashEl = document.getElementById('canvasHashVal');
    if (canvasHashEl) canvasHashEl.textContent = canvasFp.hash;

    var canvasImg = document.getElementById('canvasFpPreview');
    if (canvasImg && canvasFp.previewData) {
      canvasImg.src = canvasFp.previewData;
    }

    // 3. Audio
    FingerprintEngine.getAudioFingerprint(function (audioFp) {
      auditReport.fingerprints.audio = audioFp;
      var audioHashEl = document.getElementById('audioHashVal');
      if (audioHashEl) audioHashEl.textContent = audioFp.hash;
    });

    // 4. Fonts
    var fontProbe = FingerprintEngine.probeSystemFonts();
    auditReport.fingerprints.fonts = fontProbe;

    var fontCountEl = document.getElementById('fontsDetectedCount');
    if (fontCountEl) fontCountEl.textContent = fontProbe.detectedCount + ' / ' + fontProbe.totalTested + ' Fonts';

    var fontBadges = document.getElementById('fontsBadgeContainer');
    if (fontBadges && fontProbe.detectedFonts) {
      fontBadges.innerHTML = '';
      fontProbe.detectedFonts.forEach(function (f) {
        var span = document.createElement('span');
        span.className = 'status-badge info';
        span.style.margin = '2px';
        span.textContent = f;
        fontBadges.appendChild(span);
      });
    }

    // 5. Codecs
    var codecs = FingerprintEngine.probeMediaCodecs();
    auditReport.fingerprints.codecs = codecs;

    var videoCodecsContainer = document.getElementById('videoCodecsList');
    if (videoCodecsContainer) {
      videoCodecsContainer.innerHTML = '';
      codecs.video.forEach(function (c) {
        var row = document.createElement('div');
        row.className = 'test';
        row.innerHTML = '<span>' + c.name + '</span><span class="status-badge ' + (c.pass ? 'pass' : 'fail') + '">' + c.support.toUpperCase() + '</span>';
        videoCodecsContainer.appendChild(row);
      });
    }

    var audioCodecsContainer = document.getElementById('audioCodecsList');
    if (audioCodecsContainer) {
      audioCodecsContainer.innerHTML = '';
      codecs.audio.forEach(function (a) {
        var aRow = document.createElement('div');
        aRow.className = 'test';
        aRow.innerHTML = '<span>' + a.name + '</span><span class="status-badge ' + (a.pass ? 'pass' : 'fail') + '">' + a.support.toUpperCase() + '</span>';
        audioCodecsContainer.appendChild(aRow);
      });
    }

    // 6. Storage Quota
    FingerprintEngine.probeStorage(function (storage) {
      auditReport.fingerprints.storage = storage;
      var quotaEl = document.getElementById('storageQuotaVal');
      if (quotaEl) quotaEl.textContent = storage.quotaEstimate || 'N/A';
    });
  }

  // =========================================================================
  // MEDIA LAB WIRING
  // =========================================================================
  function initMediaLab() {
    var startCamBtn = document.getElementById('btnStartCam');
    var stopCamBtn = document.getElementById('btnStopCam');
    var snapCamBtn = document.getElementById('btnSnapCam');
    var camVideo = document.getElementById('cameraVideo');
    var camCanvas = document.getElementById('cameraCanvas');
    var camStatus = document.getElementById('cameraStatus');
    var filterSelect = document.getElementById('cameraFilterSelect');

    if (startCamBtn) {
      startCamBtn.addEventListener('click', function () {
        MediaLabEngine.startCamera(camVideo, camStatus, function (ok) {
          if (ok) log('Kamera video berhasil dimulai.', 'ok');
          else log('Gagal membuka kamera video.', 'fail');
        });
      });
    }

    if (stopCamBtn) {
      stopCamBtn.addEventListener('click', function () {
        MediaLabEngine.stopCamera(camVideo, camStatus);
        log('Kamera video dimatikan.', 'info');
      });
    }

    if (snapCamBtn) {
      snapCamBtn.addEventListener('click', function () {
        MediaLabEngine.snapshotCamera(camVideo, camCanvas, filterSelect ? filterSelect.value : 'none');
        log('Snapshot kamera diambil dengan filter: ' + (filterSelect ? filterSelect.value : 'none'), 'info');
      });
    }

    // Microphone
    var startMicBtn = document.getElementById('btnStartMic');
    var stopMicBtn = document.getElementById('btnStopMic');
    var micCanvas = document.getElementById('micCanvas');
    var micMeter = document.getElementById('micMeterFill');
    var micStatus = document.getElementById('micStatus');

    if (startMicBtn) {
      startMicBtn.addEventListener('click', function () {
        MediaLabEngine.startMicrophone(micCanvas, micMeter, micStatus, function (ok) {
          if (ok) log('Audio capture aktif. Oscilloscope berjalan pada 60 FPS.', 'ok');
          else log('Gagal mengakses mikrofon.', 'fail');
        });
      });
    }

    if (stopMicBtn) {
      stopMicBtn.addEventListener('click', function () {
        MediaLabEngine.stopMicrophone(micStatus);
        log('Audio capture dimatikan.', 'info');
      });
    }

    // Web Audio Synthesizer
    var startSynthBtn = document.getElementById('btnStartSynth');
    var stopSynthBtn = document.getElementById('btnStopSynth');
    var synthFreqInput = document.getElementById('synthFreqRange');
    var synthFreqVal = document.getElementById('synthFreqVal');
    var synthWaveSelect = document.getElementById('synthWaveSelect');
    var synthStatus = document.getElementById('synthStatus');

    if (synthFreqInput) {
      synthFreqInput.addEventListener('input', function () {
        var hz = parseInt(synthFreqInput.value, 10);
        if (synthFreqVal) synthFreqVal.textContent = hz + ' Hz';
        MediaLabEngine.updateSynthFreq(hz);
      });
    }

    if (startSynthBtn) {
      startSynthBtn.addEventListener('click', function () {
        var hz = parseInt(synthFreqInput ? synthFreqInput.value : 440, 10);
        var wave = synthWaveSelect ? synthWaveSelect.value : 'sine';
        MediaLabEngine.startSynth(hz, wave, 0.2, synthStatus);
        log('Web Audio Synthesizer membunyikan frekuensi ' + hz + ' Hz (' + wave + ').', 'ok');
      });
    }

    if (stopSynthBtn) {
      stopSynthBtn.addEventListener('click', function () {
        MediaLabEngine.stopSynth(synthStatus);
        log('Web Audio Synthesizer dihentikan.', 'info');
      });
    }

    // Dynamic Video Player
    var testVideo = document.getElementById('testVideo');
    var videoStatus = document.getElementById('videoStatus');
    MediaLabEngine.initVideoGenerator(testVideo, videoStatus);

    // TTS
    var speakBtn = document.getElementById('btnSpeak');
    var stopSpeakBtn = document.getElementById('btnStopSpeak');
    var ttsText = document.getElementById('ttsTextInput');
    var ttsStatus = document.getElementById('ttsStatus');

    if (speakBtn) {
      speakBtn.addEventListener('click', function () {
        MediaLabEngine.speakText(ttsText ? ttsText.value : '', 1.0, 1.0, ttsStatus);
        log('Speech Synthesis mengucapkan teks: "' + (ttsText ? ttsText.value : '') + '"', 'info');
      });
    }

    if (stopSpeakBtn) {
      stopSpeakBtn.addEventListener('click', function () {
        MediaLabEngine.stopSpeech();
        if (ttsStatus) ttsStatus.textContent = 'Dibatalkan.';
      });
    }
  }

  // =========================================================================
  // STRESS BENCHMARKS WIRING
  // =========================================================================
  function initStressLab() {
    // 1. DOM Stress
    var btnDomStress = document.getElementById('btnRunDomStress');
    var domContainer = document.getElementById('domStressContainer');
    var domResult = document.getElementById('domStressResult');

    if (btnDomStress) {
      btnDomStress.addEventListener('click', function () {
        log('Memulai DOM Thrashing Stress Test (5,000 Nodes)...', 'info');
        btnDomStress.disabled = true;

        setTimeout(function () {
          var res = StressBenchmarkEngine.runDomStress(5000, domContainer);
          auditReport.benchmarks.dom = res;
          if (domResult) {
            domResult.textContent = JSON.stringify(res, null, 2);
          }
          btnDomStress.disabled = false;
          log('DOM Stress selesai: ' + res.insertTimeMs + 'ms insert, ' + res.opsPerSec + ' ops/sec', 'ok');
        }, 50);
      });
    }

    // 2. Canvas Particles
    var particleSlider = document.getElementById('particleCountRange');
    var particleCountLabel = document.getElementById('particleCountLabel');

    if (particleSlider) {
      particleSlider.addEventListener('input', function () {
        var count = parseInt(particleSlider.value, 10);
        if (particleCountLabel) particleCountLabel.textContent = count + ' Particles';
        StressBenchmarkEngine.setParticleCount(count);
        log('Jumlah partikel disetel ke: ' + count, 'info');
      });
    }

    // 3. Compute Benchmark
    var btnMainCompute = document.getElementById('btnRunMainCompute');
    var btnWorkerCompute = document.getElementById('btnRunWorkerCompute');
    var computeResult = document.getElementById('computeResult');

    if (btnMainCompute) {
      btnMainCompute.addEventListener('click', function () {
        log('Menjalankan Prime Sieve di Main Thread (Sync)...', 'warn');
        btnMainCompute.disabled = true;
        setTimeout(function () {
          StressBenchmarkEngine.runComputeBenchmark(1500000, false, function (res) {
            auditReport.benchmarks.computeMain = res;
            if (computeResult) computeResult.textContent = JSON.stringify(res, null, 2);
            btnMainCompute.disabled = false;
            log('Main Thread Compute selesai dalam ' + res.elapsedMs + ' ms.', 'ok');
          });
        }, 30);
      });
    }

    if (btnWorkerCompute) {
      btnWorkerCompute.addEventListener('click', function () {
        log('Menjalankan Prime Sieve di Background Web Worker...', 'info');
        btnWorkerCompute.disabled = true;
        StressBenchmarkEngine.runComputeBenchmark(1500000, true, function (res) {
          auditReport.benchmarks.computeWorker = res;
          if (computeResult) computeResult.textContent = JSON.stringify(res, null, 2);
          btnWorkerCompute.disabled = false;
          log('Worker Thread Compute selesai dalam ' + res.elapsedMs + ' ms (Latency: ' + res.totalLatencyMs + ' ms)', 'ok');
        });
      });
    }

    // 4. Memory Stress
    var btnMemStress = document.getElementById('btnRunMemStress');
    var memResult = document.getElementById('memStressResult');

    if (btnMemStress) {
      btnMemStress.addEventListener('click', function () {
        log('Mengalokasikan 200 MB TypedArray memory blocks...', 'info');
        btnMemStress.disabled = true;
        setTimeout(function () {
          var res = StressBenchmarkEngine.runMemoryStress(200);
          auditReport.benchmarks.memory = res;
          if (memResult) memResult.textContent = JSON.stringify(res, null, 2);
          btnMemStress.disabled = false;
          log('Memory Stress selesai: ' + res.allocatedMb + ' MB dialokasikan (' + res.allocTimeMs + ' ms). Kecepatan: ' + res.throughputMbSec + ' MB/s', 'ok');
        }, 50);
      });
    }

    // 5. Storage I/O
    var btnStorageStress = document.getElementById('btnRunStorageStress');
    var storageResult = document.getElementById('storageIoResult');

    if (btnStorageStress) {
      btnStorageStress.addEventListener('click', function () {
        log('Menjalankan Storage I/O Stress (1,000 bulk records)...', 'info');
        btnStorageStress.disabled = true;
        setTimeout(function () {
          var res = StressBenchmarkEngine.runStorageIoStress(1000);
          auditReport.benchmarks.storage = res;
          if (storageResult) storageResult.textContent = JSON.stringify(res, null, 2);
          btnStorageStress.disabled = false;
          log('Storage I/O selesai: ' + res.writeOpsPerSec + ' write ops/sec, ' + res.readOpsPerSec + ' read ops/sec.', 'ok');
        }, 50);
      });
    }
  }

  // =========================================================================
  // REPORT EXPORT & CLIPBOARD
  // =========================================================================
  window.downloadAuditJson = function () {
    var dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(auditReport, null, 2));
    var dlAnchor = document.createElement('a');
    dlAnchor.setAttribute('href', dataStr);
    dlAnchor.setAttribute('download', 'respect-browser-audit-' + Date.now() + '.json');
    document.body.appendChild(dlAnchor);
    dlAnchor.click();
    document.body.removeChild(dlAnchor);
    log('Audit report berhasil diunduh dalam format JSON.', 'ok');
  };

  window.copyAuditMarkdown = function () {
    var md = '# ⚡ Respect Browser Capability & Stress Audit Report\n\n' +
      '- **Waktu Audit**: `' + auditReport.timestamp + '`\n' +
      '- **Engine Flavor**: `' + (auditReport.environment.engineFlavor || 'Unknown') + '`\n' +
      '- **Platform**: `' + (auditReport.environment.platform || 'Unknown') + '`\n' +
      '- **User-Agent**: `' + (auditReport.environment.userAgent || 'Unknown') + '`\n' +
      '- **Hardware**: `' + (auditReport.hardware.cpuCores || 'N/A') + ' Cores`, `' + (auditReport.hardware.deviceMemoryGB || 'N/A') + ' RAM`\n' +
      '- **GPU Unmasked**: `' + (auditReport.webgl.unmaskedRenderer || 'N/A') + '` (' + (auditReport.webgl.unmaskedVendor || 'N/A') + ')\n' +
      '- **Canvas Hash**: `' + (auditReport.fingerprints.canvas ? auditReport.fingerprints.canvas.hash : 'N/A') + '`\n' +
      '- **Audio Hash**: `' + (auditReport.fingerprints.audio ? auditReport.fingerprints.audio.hash : 'N/A') + '`\n' +
      '- **Capability Score**: **' + (auditReport.capabilities.percentage || '0') + '%** (' + (auditReport.capabilities.passed || 0) + ' / ' + (auditReport.capabilities.total || 0) + ')\n' +
      '- **Total Global APIs**: **' + (auditReport.apiDump.counts ? auditReport.apiDump.counts.total : 'N/A') + '**\n\n' +
      '---\n*Generated by Respect Browser Stress Testing Suite*';

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(md).then(function () {
        alert('Ringkasan Markdown berhasil disalin ke clipboard!');
        log('Markdown report disalin ke clipboard.', 'ok');
      }).catch(function (e) {
        prompt('Salin teks markdown berikut:', md);
      });
    } else {
      prompt('Salin teks markdown berikut:', md);
    }
  };

  // =========================================================================
  // BOOTSTRAP INITIALIZATION
  // =========================================================================
  window.addEventListener('DOMContentLoaded', function () {
    log('Menginisialisasi Respect Browser Stress Testing Suite...', 'info');
    initTabs();
    initTelemetry();
    runCapabilities();
    initApiDump();
    initFingerprints();
    initMediaLab();
    initStressLab();
    log('Seluruh subsistem berhasil dimuat dan siap untuk pengujian beban.', 'ok');
  });

})();
