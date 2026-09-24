/**
 * Respect Browser Stress Testing Suite - Master App Controller
 * Mengintegrasikan Telemetry HUD, API Dump explorer dengan deep inspection,
 * Capability Matrix, Media Lab, Stress Benchmarks, Live JS REPL, dan Bulletproof Report Modal.
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
    apiDump: {
      counts: {},
      summaryApis: []
    },
    benchmarks: {},
    interactions: [],
    checklist: {},
    manualNotes: '',
    logs: { count: 0, events: [] }
  };

  // Live event log (console + errors + runner) yang ikut diekspor ke laporan.
  var eventLog = [];
  var interactionResults = [];
  var MAX_LOG_EVENTS = 2000;

  // History for interactive REPL
  var replHistory = [];
  var replHistoryIndex = -1;

  // =========================================================================
  // DEFENSIVE ENGINE RESOLVERS (Anti-Block & Crash-Proof Fallbacks)
  // =========================================================================
  var ProfileEngine = (typeof window !== 'undefined' && (window.SysProfileEngine || window.FingerprintEngine)) || {
    getEnvironment: function () {
      var nav = typeof navigator !== 'undefined' ? navigator : {};
      var isRespectModern = typeof window.mbQuery === 'function';
      var isRespectLite = typeof window.ipc !== 'undefined' && window.ipc && typeof window.ipc.invoke === 'function';
      return {
        engineFlavor: isRespectModern ? 'Respect Modern (Chromium 132 Core)' : isRespectLite ? 'Respect Lite (Miniblink 49 Core)' : 'Standard Web Context',
        userAgent: nav.userAgent || 'Unknown',
        appVersion: nav.appVersion || 'Unknown',
        platform: nav.platform || 'Unknown',
        vendor: nav.vendor || 'Unknown',
        language: nav.language || 'Unknown',
        languages: nav.languages ? Array.prototype.slice.call(nav.languages) : [],
        cookieEnabled: !!nav.cookieEnabled,
        onLine: nav.onLine !== undefined ? nav.onLine : true,
        doNotTrack: nav.doNotTrack || 'unspecified',
        maxTouchPoints: nav.maxTouchPoints || 0,
        pdfViewerEnabled: !!nav.pdfViewerEnabled,
        isSecureContext: !!window.isSecureContext,
        respectHooks: { mbQuery: isRespectModern, ipc: isRespectLite }
      };
    },
    getHardwareSpecs: function (cb) {
      var nav = typeof navigator !== 'undefined' ? navigator : {};
      var scr = typeof window.screen !== 'undefined' ? window.screen : {};
      var res = {
        cpuCores: nav.hardwareConcurrency || 'N/A',
        deviceMemoryGB: nav.deviceMemory ? (nav.deviceMemory + ' GB') : 'N/A',
        screenWidth: scr.width || 0,
        screenHeight: scr.height || 0,
        colorDepth: scr.colorDepth || 24,
        pixelRatio: window.devicePixelRatio || 1
      };
      if (cb) cb(res);
      return res;
    },
    getWebGLFingerprint: function () { return { supported: false, unmaskedVendor: 'N/A', unmaskedRenderer: 'N/A' }; },
    getCanvasFingerprint: function () { return { supported: false, hash: 'BLOCKED_BY_CLIENT' }; },
    getAudioFingerprint: function (cb) { if (cb) cb({ supported: false, hash: 'BLOCKED_BY_CLIENT' }); },
    probeSystemFonts: function () { return { supported: false, detectedCount: 0, totalTested: 0, detectedFonts: [] }; },
    probeMediaCodecs: function () { return { video: [], audio: [] }; },
    probeStorage: function (cb) { var r = { localStorage: true, sessionStorage: true, indexedDB: true }; if (cb) cb(r); return r; }
  };
  var FingerprintEngine = ProfileEngine;
  var SysProfileEngine = ProfileEngine;

  var CapabilityEngine = (typeof window !== 'undefined' && window.CapabilityEngine) || {
    runAllTests: function () {
      return { percentage: '0.0', passed: 0, failed: 0, total: 0, tests: [], byCategory: {} };
    }
  };

  var ApiDumpEngine = (typeof window !== 'undefined' && window.ApiDumpEngine) || {
    dumpGlobalApis: function () {
      return { counts: { total: 0, constructors: 0, functions: 0, objects: 0, values: 0, respectHooks: 0 }, apis: [] };
    },
    filterApis: function () { return []; },
    getNamespaceDetails: function () { return null; }
  };

  var MediaLabEngine = (typeof window !== 'undefined' && window.MediaLabEngine) || {
    initCameraPreview: function () {},
    startCamera: function (v, s, cb) { if (cb) cb(false, new Error('MediaLabEngine unavailable')); },
    stopCamera: function () {},
    snapshotCamera: function () {},
    startMicrophone: function (c, m, s, cb) { if (cb) cb(false, new Error('MediaLabEngine unavailable')); },
    stopMicrophone: function () {},
    startSynth: function () {},
    updateSynthFreq: function () {},
    stopSynth: function () {},
    initVideoGenerator: function () {},
    speakText: function () {},
    stopSpeech: function () {}
  };

  var StressBenchmarkEngine = (typeof window !== 'undefined' && window.StressBenchmarkEngine) || {
    runDomStress: function () { return { opsPerSec: 0, insertTimeMs: '0' }; },
    initParticleCanvas: function () {},
    startParticleSimulation: function () {},
    setParticleCount: function () {},
    stopParticleSimulation: function () {},
    runComputeBenchmark: function (m, w, cb) { if (cb) cb({ elapsedMs: '0', totalLatencyMs: '0' }); },
    runMemoryStress: function () { return { allocatedMb: 0, allocTimeMs: '0', throughputMbSec: 0 }; },
    runStorageIoStress: function () { return { writeOpsPerSec: 0, readOpsPerSec: 0 }; }
  };

  var FeatureLabEngine = (typeof window !== 'undefined' && window.FeatureLabEngine) || {
    list: function () { return []; },
    get: function () { return null; },
    run: function (id) {
      return Promise.resolve({ id: id, category: 'unknown', name: id, pass: false, detail: 'FeatureLabEngine tidak tersedia', durationMs: '0' });
    },
    runAll: function () { return Promise.resolve([]); },
    abort: function () {},
    resetAbort: function () {}
  };

  // =========================================================================
  // SMART LOGGING, CONSOLE CAPTURE & GLOBAL ERROR HANDLER
  // =========================================================================
  function timestampIso() {
    try { return new Date().toISOString(); } catch (e) { return String(Date.now()); }
  }

  function recordEvent(level, source, message) {
    try {
      eventLog.push({
        time: timestampIso(),
        level: String(level || 'info'),
        source: String(source || 'app'),
        message: String(message === undefined ? '' : message).substring(0, 4000)
      });
      if (eventLog.length > MAX_LOG_EVENTS) {
        eventLog.splice(0, eventLog.length - MAX_LOG_EVENTS);
      }
    } catch (e) {}
  }

  function stringifyArg(arg) {
    if (arg === null) return 'null';
    if (arg === undefined) return 'undefined';
    var t = typeof arg;
    if (t === 'string') return arg;
    if (t === 'number' || t === 'boolean' || t === 'bigint') return String(arg);
    if (t === 'function') return 'function ' + (arg.name || 'anonymous') + '()';
    if (t === 'symbol') return arg.toString();
    if (t === 'object') {
      try { return JSON.stringify(arg); } catch (e) { return Object.prototype.toString.call(arg); }
    }
    return String(arg);
  }

  // Rekam console.log/info/warn/error/debug agar semua pesan engine ikut tersimpan.
  function installConsoleCapture() {
    if (typeof console === 'undefined') return;
    var levels = ['log', 'info', 'warn', 'error', 'debug'];
    for (var i = 0; i < levels.length; i++) {
      (function (level) {
        var original = console[level];
        console[level] = function () {
          var args = Array.prototype.slice.call(arguments);
          var parts = [];
          for (var a = 0; a < args.length; a++) parts.push(stringifyArg(args[a]));
          recordEvent(level === 'log' ? 'info' : level, 'console', parts.join(' '));
          if (typeof original === 'function') {
            try { original.apply(console, args); } catch (e) {}
          }
        };
      })(levels[i]);
    }
  }

  function installGlobalHandlers() {
    window.onerror = function (msg, url, lineNo, columnNo, error) {
      var filename = url ? url.substring(url.lastIndexOf('/') + 1) : 'script';
      var detail = msg + ' (' + filename + ':' + lineNo + (columnNo ? ':' + columnNo : '') + ')';
      log('UNCAUGHT EXCEPTION: ' + detail, 'fail');
      recordEvent('error', 'window.onerror', detail);
      return false;
    };

    window.onunhandledrejection = function (event) {
      var reason = event && event.reason ? (event.reason.message || String(event.reason)) : 'Unknown rejection';
      log('UNHANDLED PROMISE REJECTION: ' + reason, 'fail');
      recordEvent('error', 'unhandledrejection', reason);
    };

    // Jejak navigasi & lifecycle agar log lengkap.
    try {
      window.addEventListener('load', function () { recordEvent('info', 'lifecycle', 'window load event fired; href=' + window.location.href); });
      document.addEventListener('visibilitychange', function () { recordEvent('info', 'lifecycle', 'visibilitychange -> ' + document.visibilityState); });
      window.addEventListener('beforeunload', function () { recordEvent('info', 'lifecycle', 'beforeunload fired'); });
      window.addEventListener('hashchange', function () { recordEvent('info', 'lifecycle', 'hashchange -> ' + window.location.hash); });
      window.addEventListener('popstate', function () { recordEvent('info', 'lifecycle', 'popstate -> ' + window.location.href); });
    } catch (e) {}
  }

  function log(msg, type) {
    type = type || 'info';
    recordEvent(type, 'suite', msg);
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
    text.textContent = (type === 'ok' ? '✓ ' : type === 'fail' ? '✗ ' : type === 'warn' ? '⚠ ' : 'ℹ ') + msg;

    line.appendChild(time);
    line.appendChild(text);
    terminal.appendChild(line);
    terminal.scrollTop = terminal.scrollHeight;
  }

  window.clearTerminal = function () {
    var terminal = document.getElementById('terminalBody');
    if (terminal) terminal.innerHTML = '';
  };

  window.getRespectEventLog = function () {
    return eventLog.slice();
  };


  // =========================================================================
  // INTERACTIVE JAVASCRIPT REPL CONSOLE
  // =========================================================================
  function initRepl() {
    var input = document.getElementById('replInput');
    var btn = document.getElementById('btnReplRun');
    if (!input) return;

    function executeRepl() {
      var code = input.value.trim();
      if (!code) return;

      replHistory.push(code);
      replHistoryIndex = replHistory.length;
      input.value = '';

      log('> ' + code, 'info');

      try {
        /* eslint-disable no-eval */
        var result = window.eval(code);
        var resultStr = '';
        if (result === undefined) resultStr = 'undefined';
        else if (result === null) resultStr = 'null';
        else if (typeof result === 'object') {
          try {
            resultStr = JSON.stringify(result, null, 2);
          } catch (e) {
            resultStr = String(result);
          }
        } else {
          resultStr = String(result);
        }
        log('=> ' + resultStr, 'ok');
      } catch (err) {
        log('ERROR: ' + (err.stack || err.message || String(err)), 'fail');
      }
    }

    input.addEventListener('keydown', function (e) {
      if (e.key === 'Enter') {
        executeRepl();
      } else if (e.key === 'ArrowUp') {
        if (replHistory.length > 0 && replHistoryIndex > 0) {
          replHistoryIndex--;
          input.value = replHistory[replHistoryIndex];
        }
      } else if (e.key === 'ArrowDown') {
        if (replHistoryIndex < replHistory.length - 1) {
          replHistoryIndex++;
          input.value = replHistory[replHistoryIndex];
        } else {
          replHistoryIndex = replHistory.length;
          input.value = '';
        }
      }
    });

    if (btn) {
      btn.addEventListener('click', executeRepl);
    }
  }

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

        // Lazy particle init
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
    // Lengkapi info lokasi/origin nyata untuk routing diagnostics.
    try {
      env.origin = (window.location.origin !== undefined) ? window.location.origin : (window.location.protocol + '//' + window.location.host);
      env.href = window.location.href;
      env.protocol = window.location.protocol;
      env.host = window.location.host;
      env.hostname = window.location.hostname;
      env.port = window.location.port;
      env.pathname = window.location.pathname;
      env.search = window.location.search;
      env.hash = window.location.hash;
      env.referrer = document.referrer || '';
      env.title = document.title;
      env.crossOriginIsolated = !!window.crossOriginIsolated;
    } catch (e) {}
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

    // Environment card values
    var envEngineName = document.getElementById('envEngineName');
    if (envEngineName) envEngineName.textContent = env.engineFlavor;

    var envUserAgent = document.getElementById('envUserAgent');
    if (envUserAgent) envUserAgent.textContent = env.userAgent;

    var envIpcStatus = document.getElementById('envIpcStatus');
    if (envIpcStatus) {
      envIpcStatus.textContent = env.respectHooks.mbQuery ? 'Chromium 132 (mbQuery Native)' :
                                env.respectHooks.ipc ? 'Miniblink 49 (window.ipc Native)' : 'Standard Web Context';
    }

    var envSecureContext = document.getElementById('envSecureContext');
    if (envSecureContext) envSecureContext.textContent = env.isSecureContext ? 'YES (Secure)' : 'NO (Local/HTTP)';

    var envLanguage = document.getElementById('envLanguage');
    if (envLanguage) envLanguage.textContent = env.language + ' (' + env.languages.join(', ') + ')';

    var envOnline = document.getElementById('envOnline');
    if (envOnline) envOnline.textContent = env.onLine ? 'Connected (Online)' : 'Offline';

    var coresEl = document.getElementById('hudCores');
    var ramEl = document.getElementById('hudRam');

    FingerprintEngine.getHardwareSpecs(function (specs) {
      auditReport.hardware = specs;
      if (coresEl) coresEl.textContent = specs.cpuCores + ' Threads';
      if (ramEl) ramEl.textContent = specs.deviceMemoryGB;

      var resEl = document.getElementById('hudResolution');
      if (resEl) resEl.textContent = specs.screenWidth + 'x' + specs.screenHeight + ' (@' + specs.pixelRatio + 'x)';
    });

    log('Engine: ' + env.engineFlavor + ' | Origin: ' + (window.location.origin || 'N/A'), 'ok');
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

    log('Capability Test selesai: ' + cap.passed + '/' + cap.total + ' (' + cap.percentage + '%)', 'ok');
  }

  // =========================================================================
  // API DUMP POPULATION WITH DEEP INSPECTION & SAMPLES
  // =========================================================================
  function populateApiDump(category, query) {
    var filtered = ApiDumpEngine.filterApis(query, category);
    var tbody = document.getElementById('apiDumpTableBody');
    if (!tbody) return;

    tbody.innerHTML = '';
    var maxDisplay = 300; // Cap to prevent table rendering lag
    var displayItems = filtered.slice(0, maxDisplay);

    displayItems.forEach(function (api) {
      var tr = document.createElement('tr');
      tr.style.cursor = 'pointer';

      var tdName = document.createElement('td');
      tdName.innerHTML = '<code>' + api.name + '</code>' +
        (api.isRespect ? ' <span class="status-badge info">Respect Hook</span>' : '') +
        (api.descriptor ? ' <small style="color:var(--text-dim); margin-left:6px;">[' + api.descriptor + ']</small>' : '');

      var tdType = document.createElement('td');
      tdType.innerHTML = '<span class="status-badge ' + (api.isConstructor ? 'pass' : api.isCallable ? 'info' : 'warn') + '">' + api.summary + '</span>';

      var tdVal = document.createElement('td');
      tdVal.style.fontFamily = 'var(--font-mono)';
      tdVal.style.fontSize = '11px';
      tdVal.style.color = '#38bdf8';
      tdVal.textContent = api.sampleValue;

      var tdOrigin = document.createElement('td');
      tdOrigin.className = 'muted';
      tdOrigin.textContent = api.origin;

      tr.appendChild(tdName);
      tr.appendChild(tdType);
      tr.appendChild(tdVal);
      tr.appendChild(tdOrigin);

      // Expand on click to show deep properties
      tr.addEventListener('click', function () {
        var existingBox = tr.nextElementSibling;
        if (existingBox && existingBox.classList.contains('expanded-api-row')) {
          existingBox.remove();
          return;
        }

        var expandTr = document.createElement('tr');
        expandTr.className = 'expanded-api-row';
        var expandTd = document.createElement('td');
        expandTd.colSpan = 4;

        var box = document.createElement('div');
        box.className = 'api-detail-box';

        var content = '<strong>PROPERTI:</strong> ' + api.name + '\n' +
                      '<strong>TIPE:</strong> ' + api.type + ' | ' + api.summary + '\n' +
                      '<strong>DESCRIPTOR:</strong> ' + (api.descriptor || 'N/A') + '\n' +
                      '<strong>NILAI SAAT INI:</strong> ' + api.sampleValue;

        if (api.subProperties && api.subProperties.length > 0) {
          content += '\n<strong>SUB-PROPERTIES / KEYS:</strong> ' + api.subProperties.join(', ');
        }

        box.innerHTML = content.replace(/\n/g, '<br>');
        expandTd.appendChild(box);
        expandTr.appendChild(expandTd);
        tr.parentNode.insertBefore(expandTr, tr.nextSibling);
      });

      tbody.appendChild(tr);
    });

    var countLabel = document.getElementById('apiFilteredCount');
    if (countLabel) {
      countLabel.textContent = 'Menampilkan ' + displayItems.length + ' dari ' + filtered.length + ' API (Total: ' + ApiDumpEngine.dumpGlobalApis().counts.total + ')';
    }
  }

  function initApiDump() {
    var dump = ApiDumpEngine.dumpGlobalApis();
    auditReport.apiDump.counts = dump.counts;

    // Only store safe summary in auditReport to prevent circular JSON crash
    auditReport.apiDump.summaryApis = dump.apis.map(function (a) {
      return {
        name: a.name,
        type: a.type,
        summary: a.summary,
        value: a.sampleValue,
        origin: a.origin
      };
    });

    var totalApisVal = document.getElementById('totalApisVal');
    if (totalApisVal) totalApisVal.textContent = dump.counts.total;

    var totalApisCountTile = document.getElementById('totalApisCountTile');
    if (totalApisCountTile) totalApisCountTile.textContent = dump.counts.total;

    var totalApisBadge = document.getElementById('totalApisBadge');
    if (totalApisBadge) totalApisBadge.textContent = dump.counts.total;

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

    // Populate Key Essential Namespaces in Dashboard
    populateEssentialNamespaces();
  }

  function populateEssentialNamespaces() {
    var container = document.getElementById('essentialNamespacesContainer');
    if (!container) return;

    var targets = ['location', 'navigator', 'screen', 'document'];
    container.innerHTML = '';

    targets.forEach(function (ns) {
      var details = ApiDumpEngine.getNamespaceDetails(ns);
      if (!details) return;

      var card = document.createElement('div');
      card.className = 'card';
      card.style.padding = '12px';

      var h = document.createElement('h4');
      h.style.color = 'var(--accent-cyan)';
      h.style.marginBottom = '6px';
      h.style.fontSize = '13px';
      h.textContent = 'window.' + ns;

      var pre = document.createElement('pre');
      pre.className = 'code-block';
      pre.style.maxHeight = '140px';
      pre.textContent = JSON.stringify(details, null, 2);

      card.appendChild(h);
      card.appendChild(pre);
      container.appendChild(card);
    });
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
    var gpuRendererDetailed = document.getElementById('gpuRendererDetailed');
    var glVersionEl = document.getElementById('glVersion');
    var glExtsEl = document.getElementById('glExtensionsCount');
    var glMaxTexture = document.getElementById('glMaxTexture');
    var glShadingLang = document.getElementById('glShadingLang');

    var rendererStr = webgl.unmaskedRenderer || webgl.renderer || 'N/A';
    var vendorStr = webgl.unmaskedVendor || webgl.vendor || 'N/A';

    if (gpuVendorEl) gpuVendorEl.textContent = vendorStr;
    if (gpuRendererEl) gpuRendererEl.textContent = rendererStr;
    if (gpuRendererDetailed) gpuRendererDetailed.textContent = rendererStr + ' (' + vendorStr + ')';
    if (glVersionEl) glVersionEl.textContent = webgl.version || 'Unsupported';
    if (glExtsEl) glExtsEl.textContent = webgl.extensionsCount + ' Extensions';
    if (glMaxTexture) glMaxTexture.textContent = webgl.maxTextureSize ? (webgl.maxTextureSize + ' px') : 'N/A';
    if (glShadingLang) glShadingLang.textContent = webgl.shadingLanguage || 'N/A';

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
    if (MediaLabEngine.initCameraPreview) {
      try { MediaLabEngine.initCameraPreview(); } catch (e) {}
    }

    var startCamBtn = document.getElementById('btnStartCam');
    var stopCamBtn = document.getElementById('btnStopCam');
    var snapCamBtn = document.getElementById('btnSnapCam');
    var camVideo = document.getElementById('cameraVideo');
    var camCanvas = document.getElementById('cameraCanvas');
    var camStatus = document.getElementById('cameraStatus');
    var filterSelect = document.getElementById('cameraFilterSelect');

    if (startCamBtn) {
      startCamBtn.addEventListener('click', function () {
        try {
          log('Meminta izin akses kamera video...', 'info');
          MediaLabEngine.startCamera(camVideo, camStatus, function (ok, res) {
            if (ok) {
              if (res && res.isVirtual) {
                log('Virtual Test Camera aktif: 60 FPS live simulation stream (Crash-proof).', 'ok');
              } else {
                log('Kamera hardware aktif (1280x720).', 'ok');
              }
            } else {
              log('Akses kamera tidak aktif: ' + (res && res.message ? res.message : 'Dibatasi/Ditolak'), 'fail');
            }
          });
        } catch (e) {
          log('Kamera start error: ' + (e.message || String(e)), 'fail');
        }
      });
    }

    if (stopCamBtn) {
      stopCamBtn.addEventListener('click', function () {
        try {
          MediaLabEngine.stopCamera(camVideo, camStatus);
          log('Kamera dimatikan.', 'info');
        } catch (e) {
          log('Kamera stop error: ' + (e.message || String(e)), 'warn');
        }
      });
    }

    if (snapCamBtn) {
      snapCamBtn.addEventListener('click', function () {
        try {
          MediaLabEngine.snapshotCamera(camVideo, camCanvas, filterSelect ? filterSelect.value : 'none');
          log('Snapshot kamera diambil dengan filter: ' + (filterSelect ? filterSelect.value : 'none'), 'info');
        } catch (e) {
          log('Snapshot error: ' + (e.message || String(e)), 'warn');
        }
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
        try {
          log('Meminta izin akses mikrofon audio...', 'info');
          MediaLabEngine.startMicrophone(micCanvas, micMeter, micStatus, function (ok, err) {
            if (ok) log('Mikrofon aktif. Oscilloscope 60 FPS berjalan.', 'ok');
            else log('Akses mikrofon tidak aktif: ' + (err && err.message ? err.message : 'Dibatasi/Ditolak'), 'fail');
          });
        } catch (e) {
          log('Mikrofon start error: ' + (e.message || String(e)), 'fail');
        }
      });
    }

    if (stopMicBtn) {
      stopMicBtn.addEventListener('click', function () {
        try {
          MediaLabEngine.stopMicrophone(micStatus);
          log('Mikrofon dimatikan.', 'info');
        } catch (e) {
          log('Mikrofon stop error: ' + (e.message || String(e)), 'warn');
        }
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
        try {
          var hz = parseInt(synthFreqInput.value, 10);
          if (synthFreqVal) synthFreqVal.textContent = hz + ' Hz';
          MediaLabEngine.updateSynthFreq(hz);
        } catch (e) {}
      });
    }

    if (startSynthBtn) {
      startSynthBtn.addEventListener('click', function () {
        try {
          var hz = parseInt(synthFreqInput ? synthFreqInput.value : 440, 10);
          var wave = synthWaveSelect ? synthWaveSelect.value : 'sine';
          MediaLabEngine.startSynth(hz, wave, 0.2, synthStatus);
          log('Web Audio Synthesizer: ' + hz + ' Hz (' + wave + ').', 'info');
        } catch (e) {
          log('Synthesizer error: ' + (e.message || String(e)), 'warn');
        }
      });
    }

    if (stopSynthBtn) {
      stopSynthBtn.addEventListener('click', function () {
        try {
          MediaLabEngine.stopSynth(synthStatus);
          log('Synthesizer dimatikan.', 'info');
        } catch (e) {
          log('Synthesizer stop error: ' + (e.message || String(e)), 'warn');
        }
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
        log('Speech Synthesis mengucapkan teks.', 'info');
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
        log('Memulai DOM Thrashing (5,000 Nodes)...', 'info');
        btnDomStress.disabled = true;

        setTimeout(function () {
          var res = StressBenchmarkEngine.runDomStress(5000, domContainer);
          auditReport.benchmarks.dom = res;
          if (domResult) {
            domResult.textContent = JSON.stringify(res, null, 2);
          }
          btnDomStress.disabled = false;
          log('DOM Stress selesai: ' + res.insertTimeMs + 'ms insert (' + res.opsPerSec + ' ops/sec)', 'ok');
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
        log('Jumlah partikel disetel: ' + count, 'info');
      });
    }

    // 3. Compute Benchmark
    var btnMainCompute = document.getElementById('btnRunMainCompute');
    var btnWorkerCompute = document.getElementById('btnRunWorkerCompute');
    var computeResult = document.getElementById('computeResult');

    if (btnMainCompute) {
      btnMainCompute.addEventListener('click', function () {
        log('Menjalankan Prime Sieve di Main Thread...', 'warn');
        btnMainCompute.disabled = true;
        setTimeout(function () {
          StressBenchmarkEngine.runComputeBenchmark(1500000, false, function (res) {
            auditReport.benchmarks.computeMain = res;
            if (computeResult) computeResult.textContent = JSON.stringify(res, null, 2);
            btnMainCompute.disabled = false;
            log('Main Thread selesai: ' + res.elapsedMs + ' ms.', 'ok');
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
          log('Worker Thread selesai: ' + res.elapsedMs + ' ms (Latency: ' + res.totalLatencyMs + ' ms)', 'ok');
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
          log('Memory Stress: ' + res.allocatedMb + ' MB dialokasikan (' + res.allocTimeMs + ' ms, ' + res.throughputMbSec + ' MB/s)', 'ok');
        }, 50);
      });
    }

    // 5. Storage I/O
    var btnStorageStress = document.getElementById('btnRunStorageStress');
    var storageResult = document.getElementById('storageIoResult');

    if (btnStorageStress) {
      btnStorageStress.addEventListener('click', function () {
        log('Menjalankan Storage I/O (1,000 bulk records)...', 'info');
        btnStorageStress.disabled = true;
        setTimeout(function () {
          var res = StressBenchmarkEngine.runStorageIoStress(1000);
          auditReport.benchmarks.storage = res;
          if (storageResult) storageResult.textContent = JSON.stringify(res, null, 2);
          btnStorageStress.disabled = false;
          log('Storage I/O: ' + res.writeOpsPerSec + ' write ops/sec, ' + res.readOpsPerSec + ' read ops/sec.', 'ok');
        }, 50);
      });
    }
  }

  // =========================================================================
  // BULLETPROOF REPORT MODAL & CLIPBOARD ENGINE
  // =========================================================================
  var currentReportFormat = 'json';

  function snapshotReport() {
    auditReport.timestamp = auditReport.timestamp || new Date().toISOString();
    auditReport.interactions = interactionResults.slice();
    auditReport.logs = { count: eventLog.length, events: eventLog.slice() };
    if (typeof readManualChecklist === 'function') auditReport.checklist = readManualChecklist();
    var notesEl = document.getElementById('manualNotesInput');
    auditReport.manualNotes = notesEl ? notesEl.value : (auditReport.manualNotes || '');
  }

  function buildChecklistLines() {
    var items = (typeof manualChecklistItems === 'function') ? manualChecklistItems() : [];
    var state = (typeof readManualChecklist === 'function') ? readManualChecklist() : {};
    var lines = [];
    for (var i = 0; i < items.length; i++) {
      var it = items[i];
      var s = state[it.id];
      var mark = s && s.done ? '[x]' : '[ ]';
      lines.push('- ' + mark + ' ' + it.label + (s && s.done && s.at ? ' _(diverifikasi ' + s.at + ')_' : ''));
    }
    return lines;
  }

  function generateMarkdownText() {
    snapshotReport();
    var cap = auditReport.capabilities || {};
    var failed = (cap.tests || []).filter(function (t) { return !t.pass; });
    var md = '# ⚡ Respect Browser Capability & Stress Audit Report\n\n' +
      '- **Waktu Audit**: `' + auditReport.timestamp + '`\n' +
      '- **Engine Flavor**: `' + (auditReport.environment.engineFlavor || 'Unknown') + '`\n' +
      '- **Platform**: `' + (auditReport.environment.platform || 'Unknown') + '`\n' +
      '- **User-Agent**: `' + (auditReport.environment.userAgent || 'Unknown') + '`\n' +
      '- **Origin**: `' + (auditReport.environment.origin || window.location.origin || 'N/A') + '`\n' +
      '- **Location**: `' + (auditReport.environment.href || window.location.href) + '`\n' +
      '- **Secure Context**: `' + (auditReport.environment.isSecureContext ? 'YES' : 'NO') + '` | **Cross-Origin Isolated**: `' + (!!auditReport.environment.crossOriginIsolated) + '`\n' +
      '- **Hardware**: `' + (auditReport.hardware.cpuCores || 'N/A') + ' Cores`, `' + (auditReport.hardware.deviceMemoryGB || 'N/A') + ' RAM`\n' +
      '- **Display**: `' + (auditReport.hardware.screenWidth || 0) + 'x' + (auditReport.hardware.screenHeight || 0) + ' (@' + (auditReport.hardware.pixelRatio || 1) + 'x)`\n' +
      '- **GPU Unmasked**: `' + (auditReport.webgl.unmaskedRenderer || 'N/A') + '` (' + (auditReport.webgl.unmaskedVendor || 'N/A') + ')\n' +
      '- **Canvas Hash**: `' + (auditReport.fingerprints.canvas ? auditReport.fingerprints.canvas.hash : 'N/A') + '`\n' +
      '- **Audio Hash**: `' + (auditReport.fingerprints.audio ? auditReport.fingerprints.audio.hash : 'N/A') + '`\n' +
      '- **Capability Score**: **' + (cap.percentage || '0') + '%** (' + (cap.passed || 0) + ' / ' + (cap.total || 0) + ')\n' +
      '- **Total Global APIs**: **' + (auditReport.apiDump.counts ? auditReport.apiDump.counts.total : 'N/A') + '**\n' +
      '- **Interaksi Teruji**: ' + (auditReport.interactions.length) + ' aksi\n' +
      '- **Log Terekam**: ' + (auditReport.logs.count) + ' event\n\n';

    if (failed.length > 0) {
      md += '## ❌ Fitur Standar Web yang Gagal (' + failed.length + ')\n\n';
      for (var i = 0; i < failed.length; i++) {
        md += '- **[' + failed[i].category + ']** ' + failed[i].name + ' (`' + failed[i].id + '`) — ' + failed[i].note + '\n';
      }
      md += '\n';
    }

    if (auditReport.interactions.length > 0) {
      md += '## 🧪 Hasil Uji Interaktif & Routing\n\n';
      md += '| Kategori | Aksi | Status | Detail | Waktu |\n|---|---|---|---|---|\n';
      for (var j = 0; j < auditReport.interactions.length; j++) {
        var r = auditReport.interactions[j];
        md += '| ' + r.category + ' | ' + r.name + ' | ' + (r.pass ? '✅ PASS' : '❌ FAIL') + ' | ' + String(r.detail).replace(/\|/g, '\\|') + ' | ' + r.durationMs + ' ms |\n';
      }
      md += '\n';
    }

    var checklistLines = buildChecklistLines();
    if (checklistLines.length > 0) {
      md += '## 📋 Checklist Verifikasi Manual\n\n' + checklistLines.join('\n') + '\n\n';
    }

    if (auditReport.manualNotes) {
      md += '## 📝 Catatan Manual Penguji\n\n' + auditReport.manualNotes + '\n\n';
    }

    md += '---\n*Generated by Respect Browser Stress Testing Suite*\n';
    return md;
  }

  function generateJsonText() {
    snapshotReport();
    return JSON.stringify(auditReport, null, 2);
  }

  function generateLogText() {
    snapshotReport();
    var lines = [];
    lines.push('========================================================');
    lines.push(' RESPECT STRESS TESTING - EVENT LOG');
    lines.push('========================================================');
    lines.push('Waktu       : ' + auditReport.timestamp);
    lines.push('Engine      : ' + (auditReport.environment.engineFlavor || 'Unknown'));
    lines.push('Origin      : ' + (auditReport.environment.origin || 'N/A'));
    lines.push('Location    : ' + (auditReport.environment.href || 'N/A'));
    lines.push('User-Agent  : ' + (auditReport.environment.userAgent || 'Unknown'));
    lines.push('SecureCtx   : ' + (auditReport.environment.isSecureContext ? 'YES' : 'NO'));
    lines.push('Total Log   : ' + auditReport.logs.count + ' event');
    lines.push('--------------------------------------------------------');
    var evts = auditReport.logs.events;
    for (var i = 0; i < evts.length; i++) {
      var e = evts[i];
      lines.push('[' + e.time + '] [' + String(e.level).toUpperCase() + '] (' + e.source + ') ' + e.message);
    }
    lines.push('--------------------------------------------------------');
    lines.push('CATATAN MANUAL:');
    lines.push(auditReport.manualNotes || '(kosong)');
    lines.push('========================================================');
    return lines.join('\r\n');
  }

  window.openReportModal = function (format) {
    currentReportFormat = format || 'json';
    var modal = document.getElementById('reportModal');
    var area = document.getElementById('reportContentArea');
    var title = document.getElementById('modalReportTitle');
    var tabJson = document.getElementById('modalTabJson');
    var tabMd = document.getElementById('modalTabMd');
    var tabLog = document.getElementById('modalTabLog');

    if (!modal || !area) return;

    if (tabJson) tabJson.className = 'btn btn-sm';
    if (tabMd) tabMd.className = 'btn btn-sm';
    if (tabLog) tabLog.className = 'btn btn-sm';

    if (currentReportFormat === 'markdown') {
      area.value = generateMarkdownText();
      if (title) title.textContent = '📋 Ringkasan Laporan Markdown';
      if (tabMd) tabMd.className = 'btn btn-sm btn-primary';
    } else if (currentReportFormat === 'text') {
      area.value = generateLogText();
      if (title) title.textContent = '🖥️ Log Mentah (Console + Event + Catatan)';
      if (tabLog) tabLog.className = 'btn btn-sm btn-primary';
    } else {
      area.value = generateJsonText();
      if (title) title.textContent = '💾 Laporan Audit Lengkap (JSON)';
      if (tabJson) tabJson.className = 'btn btn-sm btn-primary';
    }

    modal.classList.add('open');
    log('Membuka jendela inspektur laporan (' + currentReportFormat.toUpperCase() + ').', 'info');
  };

  window.closeReportModal = function () {
    var modal = document.getElementById('reportModal');
    if (modal) modal.classList.remove('open');
  };

  window.switchReportFormat = function (format) {
    window.openReportModal(format);
  };

  window.copyReportFromModal = function () {
    var area = document.getElementById('reportContentArea');
    var copyBtn = document.getElementById('btnModalCopy');
    if (!area) return;

    var text = area.value;
    var success = false;

    // Strategy 1: Async Clipboard API
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(function () {
        indicateCopySuccess(copyBtn);
      }).catch(function () {
        // Fallback to execCommand
        fallbackExecCopy(area, copyBtn);
      });
      return;
    }

    // Strategy 2: execCommand fallback
    fallbackExecCopy(area, copyBtn);
  };

  function fallbackExecCopy(textareaEl, btnEl) {
    try {
      textareaEl.focus();
      textareaEl.select();
      var ok = document.execCommand('copy');
      if (ok) {
        indicateCopySuccess(btnEl);
      } else {
        alert('Teks telah dipilih. Tekan Ctrl+C untuk menyalin.');
      }
    } catch (e) {
      alert('Teks telah dipilih. Tekan Ctrl+C untuk menyalin.');
    }
  }

  function indicateCopySuccess(btnEl) {
    log('Laporan berhasil disalin ke clipboard!', 'ok');
    if (btnEl) {
      var original = btnEl.innerHTML;
      btnEl.innerHTML = '✓ Berhasil Disalin!';
      btnEl.classList.add('btn-success');
      setTimeout(function () {
        btnEl.innerHTML = original;
        btnEl.classList.remove('btn-success');
      }, 2000);
    }
  }

  window.downloadReportFile = function () {
    var area = document.getElementById('reportContentArea');
    if (!area) return;

    var content = area.value;
    var isMd = currentReportFormat === 'markdown';
    var isLog = currentReportFormat === 'text';
    var mime = isMd ? 'text/markdown;charset=utf-8' : (isLog ? 'text/plain;charset=utf-8' : 'application/json;charset=utf-8');
    var ext = isMd ? '.md' : (isLog ? '.log' : '.json');
    var filename = 'respect-audit-' + Date.now() + ext;

    try {
      var blob = new Blob([content], { type: mime });
      var url = URL.createObjectURL(blob);
      var a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      setTimeout(function () { URL.revokeObjectURL(url); }, 5000);
      log('File laporan berhasil diunduh: ' + filename, 'ok');
    } catch (e) {
      // Data URI fallback
      try {
        var dataUri = 'data:' + mime + ',' + encodeURIComponent(content);
        var a2 = document.createElement('a');
        a2.href = dataUri;
        a2.download = filename;
        document.body.appendChild(a2);
        a2.click();
        document.body.removeChild(a2);
        log('File laporan diunduh via data URI fallback.', 'ok');
      } catch (err2) {
        log('Browser memblokir unduhan file langsung. Silakan gunakan tombol "Salin ke Clipboard" pada modal.', 'warn');
        alert('Browser Anda memblokir unduhan file langsung. Silakan gunakan tombol "Salin ke Clipboard" di modal untuk menyalin data!');
      }
    }
  };

  // =========================================================================
  // FEATURE LAB / QUICK RUNNER
  // Satu tempat terpusat untuk seluruh aksi uji (otomatis & yang butuh klik).
  // =========================================================================
  var CHECKLIST_KEY = '__respect_manual_checklist__';
  var runnerRunning = false;

  function manualChecklistItems() {
    return [
      { id: 'chk-alert', label: 'alert() muncul sebagai dialog & tidak crash' },
      { id: 'chk-confirm', label: 'confirm() mengembalikan OK/Cancel dengan benar' },
      { id: 'chk-prompt', label: 'prompt() menerima input & mengembalikan teks' },
      { id: 'chk-camera-stop', label: 'Tombol Stop Kamera TIDAK menutup paksa (force close)' },
      { id: 'chk-mic', label: 'Mikrofon & oscilloscope berjalan' },
      { id: 'chk-tts', label: 'TTS mengucapkan teks' },
      { id: 'chk-file-input', label: 'Dialog file picker terbuka & file terbaca' },
      { id: 'chk-clipboard', label: 'Copy/paste ke OS clipboard berhasil' },
      { id: 'chk-popup', label: 'window.open / popup berjalan' },
      { id: 'chk-fetch-local', label: 'fetch() relatif (virtual host/local server) berhasil' },
      { id: 'chk-reload-router', label: 'Reload/navigasi ulang tidak kehilangan sesi' },
      { id: 'chk-no-disk', label: 'Tidak ada file ekstra di disk saat runtime' }
    ];
  }

  function readManualChecklist() {
    try {
      var raw = localStorage.getItem(CHECKLIST_KEY);
      return raw ? JSON.parse(raw) : {};
    } catch (e) { return {}; }
  }

  function writeManualChecklist(state) {
    try { localStorage.setItem(CHECKLIST_KEY, JSON.stringify(state)); } catch (e) {}
  }

  function renderChecklist() {
    var box = document.getElementById('manualChecklistContainer');
    if (!box) return;
    var items = manualChecklistItems();
    var state = readManualChecklist();
    box.innerHTML = '';
    for (var i = 0; i < items.length; i++) {
      (function (it) {
        var row = document.createElement('label');
        row.className = 'check-row';
        var cb = document.createElement('input');
        cb.type = 'checkbox';
        cb.id = it.id;
        cb.checked = !!(state[it.id] && state[it.id].done);
        cb.addEventListener('change', function () {
          var s = readManualChecklist();
          if (cb.checked) s[it.id] = { done: true, at: new Date().toISOString() };
          else delete s[it.id];
          writeManualChecklist(s);
          log('Checklist "' + it.label + '" => ' + (cb.checked ? 'TERVERIFIKASI' : 'belum diverifikasi'), cb.checked ? 'ok' : 'info');
        });
        var span = document.createElement('span');
        span.textContent = it.label;
        row.appendChild(cb);
        row.appendChild(span);
        box.appendChild(row);
      })(items[i]);
    }
  }

  function escapeHtml(s) {
    return String(s === undefined ? '' : s)
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  function renderFeatureTables() {
    var actions = FeatureLabEngine.list();
    var autoBody = document.getElementById('runnerAutoBody');
    var manualBody = document.getElementById('runnerManualBody');
    if (autoBody) autoBody.innerHTML = '';
    if (manualBody) manualBody.innerHTML = '';

    for (var i = 0; i < actions.length; i++) {
      (function (a) {
        var tr = document.createElement('tr');

        var tdCat = document.createElement('td');
        tdCat.innerHTML = '<span class="code-pill">' + escapeHtml(a.category) + '</span>';

        var tdName = document.createElement('td');
        tdName.innerHTML = '<strong>' + escapeHtml(a.name) + '</strong>';

        var tdStatus = document.createElement('td');
        tdStatus.id = 'runner-status-' + a.id;
        tdStatus.innerHTML = '<span class="muted">Belum diuji</span>';

        var tdAct = document.createElement('td');
        var btn = document.createElement('button');
        btn.className = 'btn btn-sm ' + (a.kind === 'manual' ? 'btn-success' : 'btn-primary');
        btn.textContent = a.kind === 'manual' ? '👆 Jalankan' : '▶ Jalankan';
        btn.addEventListener('click', function () { runSingleAction(a.id, btn); });
        tdAct.appendChild(btn);
        if (a.danger) {
          var warn = document.createElement('span');
          warn.className = 'status-badge warn';
          warn.style.marginLeft = '6px';
          warn.textContent = 'DIALOG';
          tdAct.appendChild(warn);
        }
        if (a.hazard) {
          var hz = document.createElement('span');
          hz.className = 'status-badge fail';
          hz.style.marginLeft = '6px';
          hz.textContent = 'HAZARD';
          hz.title = 'Dapat menimpa window utama / membekukan aplikasi. Hanya dijalankan manual, tidak pernah oleh Run All.';
          tdAct.appendChild(hz);
        }

        tr.appendChild(tdCat);
        tr.appendChild(tdName);
        tr.appendChild(tdStatus);
        tr.appendChild(tdAct);
        var target = (a.kind === 'manual') ? manualBody : autoBody;
        if (target) target.appendChild(tr);
      })(actions[i]);
    }
  }

  function markRunnerStatus(res) {
    var td = document.getElementById('runner-status-' + res.id);
    if (!td) return;
    td.innerHTML = '<span class="status-badge ' + (res.pass ? 'pass' : 'fail') + '">' +
      (res.pass ? 'PASS' : 'FAIL') + '</span> <small class="muted">' +
      escapeHtml(String(res.detail).substring(0, 140)) + ' (' + res.durationMs + 'ms)</small>';
  }

  function recordInteraction(res) {
    var found = -1;
    for (var i = 0; i < interactionResults.length; i++) {
      if (interactionResults[i].id === res.id) { found = i; break; }
    }
    if (found >= 0) interactionResults[found] = res;
    else interactionResults.push(res);
    markRunnerStatus(res);
    updateRunnerCounters();
  }

  function updateRunnerCounters() {
    var total = interactionResults.length;
    var pass = 0;
    for (var i = 0; i < total; i++) { if (interactionResults[i].pass) pass++; }
    var elTotal = document.getElementById('runnerTotal');
    var elPass = document.getElementById('runnerPassed');
    var elFail = document.getElementById('runnerFailed');
    if (elTotal) elTotal.textContent = total;
    if (elPass) elPass.textContent = pass;
    if (elFail) elFail.textContent = total - pass;
  }

  function runSingleAction(id, btn) {
    var a = FeatureLabEngine.get(id) || { name: id };
    if (btn) btn.disabled = true;
    log('Menjalankan aksi: ' + a.name + ' [' + id + ']', 'info');
    return FeatureLabEngine.run(id, function (res) {
      recordInteraction(res);
      log((res.pass ? 'PASS' : 'FAIL') + ' - ' + a.name + ': ' + res.detail + ' (' + res.durationMs + ' ms)', res.pass ? 'ok' : 'fail');
    }).then(function (res) {
      if (btn) btn.disabled = false;
      return res;
    });
  }

  window.runAllFeatureTests = function (includeManual) {
    if (runnerRunning) { log('Runner sedang berjalan, tunggu selesai.', 'warn'); return; }
    runnerRunning = true;
    var queue = FeatureLabEngine.list().filter(function (a) { return includeManual ? true : a.kind !== 'manual'; });
    var progress = document.getElementById('runnerProgress');
    var progressWrap = document.getElementById('runnerProgressWrap');
    if (progressWrap) progressWrap.style.display = 'block';
    if (progress) progress.style.width = '0%';
    log('▶ Menjalankan ' + queue.length + ' aksi ' + (includeManual ? '(termasuk manual/dialog)' : 'otomatis') + ' secara berurutan...', 'info');

    var doneCount = 0;
    FeatureLabEngine.runAll(function (res) {
      doneCount++;
      recordInteraction(res);
      log('[' + doneCount + '/' + queue.length + '] ' + (res.pass ? 'PASS' : 'FAIL') + ' ' + res.name + ' - ' + res.detail, res.pass ? 'ok' : 'fail');
      if (progress) progress.style.width = Math.round((doneCount / queue.length) * 100) + '%';
    }, { includeManual: includeManual }).then(function (results) {
      runnerRunning = false;
      var pass = 0;
      for (var i = 0; i < results.length; i++) { if (results[i].pass) pass++; }
      log('✔ Runner selesai: ' + pass + '/' + results.length + ' PASS. Buka tab Laporan untuk mengekspor.', pass === results.length ? 'ok' : 'warn');
    });
  };

  window.stopFeatureTests = function () {
    FeatureLabEngine.abort();
    log('⏹ Permintaan stop dikirim (aksi yang sedang berjalan diselesaikan lebih dulu).', 'warn');
  };

  window.resetFeatureTests = function () {
    interactionResults = [];
    auditReport.interactions = [];
    renderFeatureTables();
    updateRunnerCounters();
    log('Runner direset.', 'info');
  };

  window.clearManualState = function () {
    try {
      localStorage.removeItem(CHECKLIST_KEY);
      localStorage.removeItem('__respect_manual_notes__');
    } catch (e) {}
    var notesEl = document.getElementById('manualNotesInput');
    if (notesEl) notesEl.value = '';
    auditReport.checklist = {};
    auditReport.manualNotes = '';
    renderChecklist();
    log('Checklist verifikasi & catatan manual dibersihkan.', 'info');
  };

  function initFeatureLab() {
    renderFeatureTables();
    updateRunnerCounters();
    renderChecklist();

    var notesEl = document.getElementById('manualNotesInput');
    if (notesEl) {
      try { notesEl.value = localStorage.getItem('__respect_manual_notes__') || ''; } catch (e) {}
      notesEl.addEventListener('input', function () {
        try { localStorage.setItem('__respect_manual_notes__', notesEl.value); } catch (e) {}
      });
    }

    var interceptorEl = document.getElementById('runnerInterceptor');
    if (interceptorEl) {
      var bits = [];
      if (typeof window.mbQuery === 'function') bits.push('mbQuery');
      if (window.ipc && typeof window.ipc.invoke === 'function') bits.push('ipc');
      if (typeof window.fetch === 'function') bits.push('fetch');
      if (typeof window.Worker === 'function') bits.push('Worker');
      interceptorEl.textContent = bits.length ? bits.join(' + ') : 'none';
    }
  }

  // =========================================================================
  // BOOTSTRAP INITIALIZATION
  // =========================================================================
  window.addEventListener('DOMContentLoaded', function () {
    installConsoleCapture();
    installGlobalHandlers();
    log('Menginisialisasi Respect Browser Stress Testing Suite...', 'info');
    try { initTabs(); } catch (e) { log('Init Tabs warning: ' + (e.message || e), 'warn'); }
    try { initTelemetry(); } catch (e) { log('Init Telemetry warning: ' + (e.message || e), 'warn'); }
    try { runCapabilities(); } catch (e) { log('Init Capabilities warning: ' + (e.message || e), 'warn'); }
    try { initApiDump(); } catch (e) { log('Init ApiDump warning: ' + (e.message || e), 'warn'); }
    try { initFingerprints(); } catch (e) { log('Init Fingerprints warning: ' + (e.message || e), 'warn'); }
    try { initMediaLab(); } catch (e) { log('Init MediaLab warning: ' + (e.message || e), 'warn'); }
    try { initStressLab(); } catch (e) { log('Init StressLab warning: ' + (e.message || e), 'warn'); }
    try { initFeatureLab(); } catch (e) { log('Init FeatureLab warning: ' + (e.message || e), 'warn'); }
    try { initRepl(); } catch (e) { log('Init REPL warning: ' + (e.message || e), 'warn'); }
    log('Seluruh subsistem siap. Gunakan tab "Uji Interaktif" untuk menjalankan semua aksi uji.', 'ok');
  });

})();
