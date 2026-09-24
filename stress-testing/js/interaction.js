/**
 * Respect Browser Stress Testing Suite - Feature & Interaction Lab Engine
 * ------------------------------------------------------------------------
 * Registri terpusat untuk semua aksi pengujian yang memerlukan interaksi
 * (klik pengguna) maupun otomatis: Fetch & Routing, Input & Form, Dialog
 * (alert/confirm/prompt), JS Bridge (mbQuery/ipc), Storage, Web Worker,
 * Media (kamera/mikrofon/synth/TTS), dan API hardware lainnya.
 *
 * Semua aksi didaftarkan lewat FeatureLabEngine.define() sehingga dapat
 * ditampilkan sekaligus di satu panel (Quick Runner) dan dijalankan berurutan
 * oleh FeatureLabEngine.runAll().
 *
 * Catatan kompatibilitas: berkas ini sengaja ditulis gaya ES5 (var + function
 * expression, tanpa arrow/class/optional-chaining/template-literal) agar tetap
 * dapat di-parse oleh engine lama (Miniblink 49) maupun modern (Chromium 132).
 */

var FeatureLabEngine = (function () {
  'use strict';

  var actions = [];
  var byId = {};
  var aborted = false;

  // -------------------------------------------------------------------------
  // HELPERS
  // -------------------------------------------------------------------------
  function safeStr(val) {
    if (val === null) return 'null';
    if (val === undefined) return 'undefined';
    var t = typeof val;
    if (t === 'string') return val.length > 300 ? (val.substring(0, 300) + '...') : val;
    if (t === 'number' || t === 'boolean' || t === 'bigint') return String(val);
    if (t === 'function') return 'function ' + (val.name || 'anonymous');
    if (t === 'symbol') return val.toString();
    try {
      return JSON.stringify(val);
    } catch (e) {
      return Object.prototype.toString.call(val);
    }
  }

  function define(spec) {
    spec.kind = spec.kind || 'auto';
    spec.timeoutMs = spec.timeoutMs || 15000;
    spec.danger = !!spec.danger;
    spec.hazard = !!spec.hazard;
    byId[spec.id] = spec;
    actions.push(spec);
    return spec;
  }

  function list() {
    return actions.slice();
  }

  function get(id) {
    return byId[id] || null;
  }

  function abort() {
    aborted = true;
  }

  function resetAbort() {
    aborted = false;
  }

  function isAborted() {
    return aborted;
  }

  function nowMs() {
    return (typeof performance !== 'undefined' && performance.now) ? performance.now() : Date.now();
  }

  function el(id) {
    return document.getElementById(id);
  }

  // Jalankan SATU aksi, selalu resolve dengan objek hasil terstruktur.
  function run(id, reporter) {
    var a = byId[id];
    if (!a) {
      return Promise.resolve({ id: id, category: 'unknown', name: id, pass: false, detail: 'Aksi tidak dikenal', durationMs: '0' });
    }

    return new Promise(function (resolve) {
      var start = nowMs();
      var settled = false;
      var timer = null;

      function finish(pass, detail) {
        if (settled) return;
        settled = true;
        if (timer) { clearTimeout(timer); timer = null; }
        var res = {
          id: a.id,
          category: a.category,
          name: a.name,
          kind: a.kind,
          pass: !!pass,
          detail: safeStr(detail === undefined ? '' : detail),
          durationMs: (nowMs() - start).toFixed(2)
        };
        if (typeof reporter === 'function') {
          try { reporter(res); } catch (e) {}
        }
        resolve(res);
      }

      timer = setTimeout(function () {
        finish(false, 'TIMEOUT: tidak ada respons dalam ' + a.timeoutMs + ' ms');
      }, a.timeoutMs);

      try {
        a.run(function (pass, detail) { finish(pass, detail); });
      } catch (e) {
        finish(false, 'EXCEPTION: ' + (e && e.message ? e.message : String(e)));
      }
    });
  }

  // Jalankan berurutan (sequential) agar mudah dibaca dan tidak membebani engine.
  function runAll(reporter, opts) {
    opts = opts || {};
    var includeManual = !!opts.includeManual;
    var onlyIds = opts.ids || null;
    resetAbort();

    var queue = actions.filter(function (a) {
      if (onlyIds && onlyIds.indexOf(a.id) === -1) return false;
      if (!includeManual && a.kind === 'manual') return false;
      // Aksi berbahaya (mis. window.open yang bisa menimpa window utama) TIDAK
      // pernah dijalankan oleh runAll; harus diklik manual satu per satu.
      if (a.hazard && !opts.allowHazard) return false;
      return true;
    });

    var results = [];
    var index = 0;

    function step() {
      if (aborted) {
        return Promise.resolve(results);
      }
      if (index >= queue.length) {
        return Promise.resolve(results);
      }
      var a = queue[index++];
      return run(a.id, reporter).then(function (res) {
        results.push(res);
        return step();
      }, function () {
        return step();
      });
    }

    return step();
  }

  // -------------------------------------------------------------------------
  // 1. FETCH & ROUTING (auto)
  // -------------------------------------------------------------------------
  define({
    id: 'route-location', category: 'Routing', name: 'Location / Origin / Protocol details', kind: 'auto',
    run: function (done) {
      var l = window.location;
      var info = {
        href: String(l.href), origin: String(l.origin || 'n/a'), protocol: l.protocol,
        host: l.host, hostname: l.hostname, port: l.port, pathname: l.pathname,
        search: l.search, hash: l.hash, isSecureContext: !!window.isSecureContext
      };
      done(true, safeStr(info));
    }
  });

  define({
    id: 'route-fetch-json', category: 'Routing', name: 'fetch() relative file (data.json)', kind: 'auto',
    run: function (done) {
      if (typeof fetch !== 'function') return done(false, 'fetch() tidak tersedia');
      fetch('data.json', { cache: 'no-store' }).then(function (r) {
        return r.text().then(function (t) {
          var ok = r.ok && t.indexOf('respect-stress-testing') !== -1;
          done(ok, 'status=' + r.status + ' bytes=' + t.length + (ok ? ' JSON OK' : ' content mismatch'));
        });
      }).catch(function (e) { done(false, 'fetch gagal: ' + (e && e.message ? e.message : String(e))); });
    }
  });

  define({
    id: 'route-fetch-query', category: 'Routing', name: 'fetch() relative + query string', kind: 'auto',
    run: function (done) {
      if (typeof fetch !== 'function') return done(false, 'fetch() tidak tersedia');
      fetch('data.json?cachebust=' + Date.now()).then(function (r) {
        return r.text().then(function (t) {
          done(r.ok && t.length > 0, 'status=' + r.status + ' len=' + t.length + ' (query diabaikan oleh virtual host)');
        });
      }).catch(function (e) { done(false, 'fetch gagal: ' + (e && e.message ? e.message : String(e))); });
    }
  });

  define({
    id: 'route-fetch-parent', category: 'Routing', name: 'fetch() sub-folder (js/interaction.js)', kind: 'auto',
    run: function (done) {
      if (typeof fetch !== 'function') return done(false, 'fetch() tidak tersedia');
      fetch('js/interaction.js').then(function (r) {
        return r.text().then(function (t) {
          done(r.ok && t.indexOf('FeatureLabEngine') !== -1, 'status=' + r.status + ' bytes=' + t.length);
        });
      }).catch(function (e) { done(false, 'fetch gagal: ' + (e && e.message ? e.message : String(e))); });
    }
  });

  define({
    id: 'route-fetch-404', category: 'Routing', name: 'fetch() missing file (404 handling)', kind: 'auto',
    run: function (done) {
      if (typeof fetch !== 'function') return done(false, 'fetch() tidak tersedia');
      fetch('__respect_missing_' + Date.now() + '.json').then(function (r) {
        return r.text().then(function (t) {
          var ct = (r.headers && typeof r.headers.get === 'function') ? r.headers.get('content-type') : 'n/a';
          var looksHtml = /<!DOCTYPE|<html/i.test(t);
          var detail;
          if (r.ok && looksHtml) {
            detail = 'status=200 ok=true bytes=' + t.length + ' type=' + ct + ' -> Virtual Host SPA-fallback ke HTML (path tak dikenal tetap 200)';
          } else if (r.ok) {
            detail = 'status=200 ok=true bytes=' + t.length + ' type=' + ct + ' -> path tak dikenal dijawab 200 (bukan 404)';
          } else {
            detail = 'status=' + r.status + ' ok=false type=' + ct + ' -> 404 ditangani dengan benar (tidak crash)';
          }
          done(true, detail);
        });
      }).catch(function (e) {
        done(true, 'Request ditolak dengan error: ' + (e && e.message ? e.message : String(e)) + ' (tidak crash)');
      });
    }
  });

  define({
    id: 'route-fetch-datauri', category: 'Routing', name: 'fetch() data: URI', kind: 'auto',
    run: function (done) {
      if (typeof fetch !== 'function') return done(false, 'fetch() tidak tersedia');
      fetch('data:text/plain;base64,UkVTUEVDVF9PSw==').then(function (r) {
        return r.text().then(function (t) { done(r.ok && t === 'RESPECT_OK', 'status=' + r.status + ' body="' + t + '"'); });
      }).catch(function (e) { done(false, 'fetch data URI gagal: ' + (e && e.message ? e.message : String(e))); });
    }
  });

  define({
    id: 'route-fetch-blob', category: 'Routing', name: 'fetch() blob: URL (createObjectURL)', kind: 'auto',
    run: function (done) {
      if (typeof fetch !== 'function' || typeof Blob !== 'function' || !window.URL || !URL.createObjectURL) {
        return done(false, 'Blob/URL.createObjectURL/fetch tidak lengkap');
      }
      var url = null;
      try {
        var blob = new Blob(['respect-blob-ok'], { type: 'text/plain' });
        url = URL.createObjectURL(blob);
      } catch (e) { return done(false, 'createObjectURL gagal: ' + e.message); }
      fetch(url).then(function (r) {
        return r.text().then(function (t) {
          try { URL.revokeObjectURL(url); } catch (e) {}
          done(r.ok && t === 'respect-blob-ok', 'status=' + r.status + ' body="' + t + '"');
        });
      }).catch(function (e) {
        try { URL.revokeObjectURL(url); } catch (x) {}
        done(false, 'fetch blob gagal: ' + (e && e.message ? e.message : String(e)));
      });
    }
  });

  define({
    id: 'route-fetch-abort', category: 'Routing', name: 'fetch() + AbortController', kind: 'auto', timeoutMs: 8000,
    run: function (done) {
      if (typeof fetch !== 'function' || typeof AbortController !== 'function') {
        return done(false, 'AbortController tidak tersedia');
      }
      var ctrl = new AbortController();
      var p = fetch('data.json?slow=' + Date.now(), { signal: ctrl.signal });
      try { ctrl.abort(); } catch (e) {}
      p.then(function () {
        done(false, 'Request selesai padahal sudah di-abort (abort tidak efektif)');
      }).catch(function (e) {
        var name = (e && e.name) ? e.name : '';
        done(name === 'AbortError', 'Promise ditolak: ' + name + ' - ' + (e && e.message ? e.message : String(e)));
      });
    }
  });

  define({
    id: 'route-xhr', category: 'Routing', name: 'XMLHttpRequest relative', kind: 'auto',
    run: function (done) {
      if (typeof XMLHttpRequest !== 'function') return done(false, 'XMLHttpRequest tidak tersedia');
      try {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', 'data.json', true);
        xhr.onreadystatechange = function () {
          if (xhr.readyState === 4) {
            var ok = xhr.status >= 200 && xhr.status < 300 && xhr.responseText.indexOf('respect') !== -1;
            done(ok, 'status=' + xhr.status + ' bytes=' + (xhr.responseText ? xhr.responseText.length : 0));
          }
        };
        xhr.onerror = function () { done(false, 'XHR onerror (status=' + xhr.status + ')'); };
        xhr.send();
      } catch (e) { done(false, 'XHR exception: ' + e.message); }
    }
  });

  define({
    id: 'route-module-import', category: 'Routing', name: 'Dynamic import() ES Module', kind: 'auto',
    run: function (done) {
      var dynamicImport = null;
      try {
        /* eslint-disable no-new-func */
        dynamicImport = new Function('u', 'return import(u);');
      } catch (e) {
        return done(false, 'Dynamic import() tidak dapat di-parse engine: ' + e.message);
      }
      // Gunakan URL absolut dari document.baseURI: `import()` di dalam new Function
      // bisa me-resolve relatif ke js/interaction.js (bukan root dokumen).
      var url = 'module-probe.mjs';
      try { url = new URL('module-probe.mjs', document.baseURI || window.location.href).href; } catch (e) {}
      dynamicImport(url).then(function (mod) {
        var ok = !!mod && typeof mod.probe === 'function' && mod.probe() === 'module-ok';
        done(ok, 'Module dimuat dari ' + url + '. probe()=' + (mod && mod.probe ? mod.probe() : 'n/a') + ' add(2,3)=' + (mod && mod.add ? mod.add(2, 3) : 'n/a'));
      }).catch(function (e) {
        var base = 'import() ditolak: ' + (e && e.message ? e.message : String(e));
        if (typeof fetch !== 'function') return done(false, base);
        // Diagnostik: cek status & MIME type file module (penyebab umum kegagalan MIME sniffing).
        fetch(url).then(function (r) {
          var ct = (r.headers && typeof r.headers.get === 'function') ? r.headers.get('content-type') : 'n/a';
          return r.text().then(function (t) {
            done(false, base + ' | fetch status=' + r.status + ' content-type=' + ct + ' bytes=' + t.length);
          });
        }).catch(function () { done(false, base); });
      });
    }
  });

  define({
    id: 'route-module-script', category: 'Routing', name: '<script type="module" src> execution', kind: 'auto',
    run: function (done) {
      try {
        var s = document.createElement('script');
        s.type = 'module';
        s.src = 'module-probe.mjs';
        s.onload = function () {
          if (s.parentNode) s.parentNode.removeChild(s);
          done(!!window.__respectModuleLoaded, window.__respectModuleLoaded ? 'Module dieksekusi (global flag terpasang)' : 'Script load event, tapi module tidak dieksekusi');
        };
        s.onerror = function () {
          if (s.parentNode) s.parentNode.removeChild(s);
          done(false, 'Script module gagal dimuat (onerror)');
        };
        document.head.appendChild(s);
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'route-iframe', category: 'Routing', name: 'Same-origin <iframe> load', kind: 'auto',
    run: function (done) {
      try {
        var old = el('__respect_probe_frame');
        if (old && old.parentNode) old.parentNode.removeChild(old);
        var f = document.createElement('iframe');
        f.id = '__respect_probe_frame';
        f.style.cssText = 'width:2px;height:2px;position:absolute;left:-9999px;border:0;';
        var settled = false;
        f.onload = function () {
          if (settled) return; settled = true;
          var val = '';
          try {
            var d = f.contentDocument || (f.contentWindow && f.contentWindow.document);
            var marker = d ? d.getElementById('frameMarker') : null;
            val = marker ? marker.textContent : '';
          } catch (e) { val = 'cross-origin blocked'; }
          setTimeout(function () { if (f.parentNode) f.parentNode.removeChild(f); }, 0);
          done(val === 'RESPECT_IFRAME_OK', 'Konten iframe: "' + val + '"');
        };
        f.onerror = function () { if (settled) return; settled = true; done(false, 'iframe onerror'); };
        document.body.appendChild(f);
        f.src = 'probe-frame.html';
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'route-image', category: 'Routing', name: 'Relative <img> asset load (SVG)', kind: 'auto',
    run: function (done) {
      try {
        var img = new Image();
        var settled = false;
        img.onload = function () {
          if (settled) return; settled = true;
          done(true, 'Gambar termuat ' + img.naturalWidth + 'x' + img.naturalHeight + 'px');
        };
        img.onerror = function () { if (settled) return; settled = true; done(false, 'img onerror (asset tidak terlayani)'); };
        img.src = 'assets/probe.svg';
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'route-stylesheet', category: 'Routing', name: 'Relative <link> stylesheet load', kind: 'auto',
    run: function (done) {
      try {
        var link = document.createElement('link');
        link.rel = 'stylesheet';
        link.href = 'assets/probe.css';
        var settled = false;
        link.onload = function () {
          if (settled) return; settled = true;
          var found = false;
          try {
            for (var i = 0; i < document.styleSheets.length; i++) {
              var ss = document.styleSheets[i];
              if (ss.href && ss.href.indexOf('probe.css') !== -1) {
                try { found = ss.cssRules && ss.cssRules.length > 0; } catch (e) { found = true; }
                break;
              }
            }
          } catch (e) {}
          done(found, 'Stylesheet terpasang & cssRules dapat dibaca: ' + found);
        };
        link.onerror = function () { if (settled) return; settled = true; done(false, 'link onerror'); };
        document.head.appendChild(link);
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'route-fetch-external', category: 'Routing', name: 'fetch() external internet (example.com)', kind: 'auto', timeoutMs: 12000,
    run: function (done) {
      if (typeof fetch !== 'function') return done(false, 'fetch() tidak tersedia');
      var finished = false;
      function fin(pass, detail) { if (finished) return; finished = true; done(pass, detail); }
      var timer = setTimeout(function () { fin(false, 'Timeout jaringan eksternal (>10s)'); }, 10000);
      try {
        fetch('https://example.com/?respect_probe=' + Date.now(), { cache: 'no-store', mode: 'cors' }).then(function (r) {
          clearTimeout(timer);
          fin(r.ok, 'Koneksi internet OK, status=' + r.status);
        }).catch(function (e) {
          clearTimeout(timer);
          fin(false, 'Jaringan eksternal gagal: ' + (e && e.message ? e.message : String(e)) + ' (normal bila offline)');
        });
      } catch (e) { clearTimeout(timer); fin(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'route-custom-scheme', category: 'Routing', name: 'Custom scheme fetch (respect://)', kind: 'auto', timeoutMs: 6000,
    run: function (done) {
      if (typeof fetch !== 'function') return done(false, 'fetch() tidak tersedia');
      fetch('respect://probe').then(function (r) {
        done(true, 'Scheme respect:// direspons status=' + r.status + ' (interceptor aktif)');
      }).catch(function (e) {
        done(true, 'Scheme respect:// tidak dikenal (ditolak bersih): ' + (e && e.message ? e.message : String(e)));
      });
    }
  });

  define({
    id: 'route-hash', category: 'Routing', name: 'Hash navigation + hashchange event', kind: 'auto', timeoutMs: 5000,
    run: function (done) {
      var marker = '#respect-probe-' + Date.now();
      var fired = false;
      function onHash() {
        if (fired) return; fired = true;
        window.removeEventListener('hashchange', onHash);
        var ok = window.location.hash === marker;
        try { history.replaceState(null, '', window.location.pathname + window.location.search); } catch (e) {}
        done(ok, 'hashchange fired, hash=' + window.location.hash);
      }
      try {
        window.addEventListener('hashchange', onHash);
        window.location.hash = marker;
        setTimeout(function () {
          if (!fired) {
            window.removeEventListener('hashchange', onHash);
            done(window.location.hash === marker, 'hash diset ke ' + window.location.hash + ' tanpa event hashchange');
          }
        }, 1200);
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'route-pushstate', category: 'Routing', name: 'history.pushState / replaceState + popstate', kind: 'auto', timeoutMs: 5000,
    run: function (done) {
      if (!window.history || typeof history.pushState !== 'function') return done(false, 'History API tidak tersedia');
      try {
        var base = window.location.pathname + window.location.search;
        history.pushState({ respectProbe: true, n: 1 }, '', base);
        var pushed = !!(history.state && history.state.respectProbe === true);
        history.replaceState({ respectProbe: true, n: 2 }, '', base);
        var replaced = !!(history.state && history.state.n === 2);
        // Uji listener popstate lewat event sintetis (tanpa history.back() agar tidak navigasi keluar).
        var popFired = false;
        function onPop() { popFired = true; }
        window.addEventListener('popstate', onPop);
        try {
          window.dispatchEvent(new PopStateEvent('popstate', { state: history.state }));
        } catch (e) {
          window.dispatchEvent(new Event('popstate'));
        }
        setTimeout(function () {
          window.removeEventListener('popstate', onPop);
          history.replaceState(null, '', base);
          done(pushed && replaced, 'pushState=' + pushed + ' replaceState=' + replaced + ' popstate listener=' + popFired);
        }, 40);
      } catch (e) {
        done(false, 'Exception: ' + e.message);
      }
    }
  });

  define({
    id: 'route-url-resolve', category: 'Routing', name: 'URL() relative resolution', kind: 'auto',
    run: function (done) {
      if (typeof URL !== 'function') return done(false, 'URL constructor tidak tersedia');
      try {
        var u = new URL('../assets/probe.svg', window.location.href);
        var p = new URL('data.json?x=1&y=2', window.location.href);
        done(true, 'resolved=' + u.href + ' | search=' + p.search);
      } catch (e) { done(false, 'URL() exception: ' + e.message); }
    }
  });

  // -------------------------------------------------------------------------
  // 2. INPUT, FORM & EVENT (auto)
  // -------------------------------------------------------------------------
  define({
    id: 'input-text-value', category: 'Input', name: 'Text input: value set & readback', kind: 'auto',
    run: function (done) {
      try {
        var i = document.createElement('input');
        i.type = 'text';
        i.value = 'respect-input-123';
        done(i.value === 'respect-input-123', 'value="' + i.value + '"');
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-events', category: 'Input', name: 'input / change / keydown event dispatch', kind: 'auto',
    run: function (done) {
      try {
        var i = document.createElement('input');
        document.body.appendChild(i);
        var got = { input: 0, change: 0, keydown: 0 };
        i.addEventListener('input', function () { got.input++; });
        i.addEventListener('change', function () { got.change++; });
        i.addEventListener('keydown', function () { got.keydown++; });
        i.value = 'abc';
        i.dispatchEvent(new Event('input', { bubbles: true }));
        i.dispatchEvent(new Event('change', { bubbles: true }));
        i.dispatchEvent(new KeyboardEvent('keydown', { key: 'a', bubbles: true }));
        document.body.removeChild(i);
        done(got.input === 1 && got.change === 1 && got.keydown === 1, safeStr(got));
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-checkbox-radio', category: 'Input', name: 'Checkbox & Radio click state', kind: 'auto',
    run: function (done) {
      try {
        var cb = document.createElement('input'); cb.type = 'checkbox';
        cb.checked = false; cb.click();
        var checkedCb = cb.checked;
        var r1 = document.createElement('input'); r1.type = 'radio'; r1.name = 'respectRadio';
        var r2 = document.createElement('input'); r2.type = 'radio'; r2.name = 'respectRadio';
        document.body.appendChild(r1); document.body.appendChild(r2);
        r1.click(); r2.click();
        var radioOk = r2.checked && !r1.checked;
        document.body.removeChild(r1); document.body.removeChild(r2);
        done(checkedCb && radioOk, 'checkbox.checked=' + checkedCb + ' radio switching=' + radioOk);
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-select-textarea', category: 'Input', name: 'Select option & Textarea', kind: 'auto',
    run: function (done) {
      try {
        var sel = document.createElement('select');
        var o1 = document.createElement('option'); o1.value = 'a'; o1.text = 'A';
        var o2 = document.createElement('option'); o2.value = 'b'; o2.text = 'B';
        sel.appendChild(o1); sel.appendChild(o2);
        sel.value = 'b';
        var ta = document.createElement('textarea');
        ta.value = 'text-area-respect';
        done(sel.value === 'b' && sel.selectedIndex === 1 && ta.value === 'text-area-respect',
          'select=' + sel.value + ' selectedIndex=' + sel.selectedIndex + ' textarea=' + ta.value);
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-range-number', category: 'Input', name: 'Range/Number value & valueAsNumber', kind: 'auto',
    run: function (done) {
      try {
        var r = document.createElement('input'); r.type = 'range'; r.min = '0'; r.max = '100'; r.value = '42';
        var n = document.createElement('input'); n.type = 'number'; n.value = '3.5';
        done(r.value === '42' && n.value === '3.5' && Number(n.value) === 3.5,
          'range=' + r.value + ' number=' + n.value + ' valueAsNumber=' + n.valueAsNumber);
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-form-submit', category: 'Input', name: 'Form submit + preventDefault', kind: 'auto',
    run: function (done) {
      try {
        var form = document.createElement('form');
        var inp = document.createElement('input'); inp.name = 'q'; inp.value = 'respect';
        form.appendChild(inp);
        document.body.appendChild(form);
        var prevented = false;
        form.addEventListener('submit', function (e) { prevented = true; e.preventDefault(); });
        if (typeof form.requestSubmit === 'function') form.requestSubmit();
        else {
          var b = document.createElement('button'); b.type = 'submit'; form.appendChild(b); b.click();
        }
        setTimeout(function () {
          if (form.parentNode) form.parentNode.removeChild(form);
          done(prevented, 'submit event terpicu & preventDefault=' + prevented);
        }, 50);
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-formdata', category: 'Input', name: 'FormData extraction', kind: 'auto',
    run: function (done) {
      if (typeof FormData !== 'function') return done(false, 'FormData tidak tersedia');
      try {
        var form = document.createElement('form');
        var a = document.createElement('input'); a.name = 'alpha'; a.value = 'AAA';
        var b = document.createElement('input'); b.name = 'beta'; b.value = 'BBB';
        form.appendChild(a); form.appendChild(b);
        var fd = new FormData(form);
        done(fd.get('alpha') === 'AAA' && fd.get('beta') === 'BBB',
          'alpha=' + fd.get('alpha') + ' beta=' + fd.get('beta'));
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-contenteditable', category: 'Input', name: 'contentEditable typing', kind: 'auto',
    run: function (done) {
      try {
        var d = document.createElement('div');
        d.contentEditable = 'true';
        d.textContent = 'editable-respect';
        var ok = d.isContentEditable === true && d.textContent === 'editable-respect';
        done(ok, 'isContentEditable=' + d.isContentEditable + ' text="' + d.textContent + '"');
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-file-field', category: 'Input', name: 'File input field presence', kind: 'auto',
    run: function (done) {
      try {
        var f = document.createElement('input');
        f.type = 'file';
        f.multiple = true;
        f.accept = '.png,.jpg,.txt';
        var ok = ('files' in f) && f.multiple === true && f.accept.indexOf('png') !== -1;
        done(ok, 'type=' + f.type + ' files=' + ('files' in f) + ' multiple=' + f.multiple + ' accept="' + f.accept + '"');
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-clipboard-write', category: 'Input', name: 'Clipboard writeText', kind: 'auto', timeoutMs: 8000,
    run: function (done) {
      if (!navigator.clipboard || typeof navigator.clipboard.writeText !== 'function') {
        return done(false, 'navigator.clipboard.writeText tidak tersedia (butuh secure context)');
      }
      navigator.clipboard.writeText('respect-clipboard-probe').then(function () {
        done(true, 'writeText sukses');
      }).catch(function (e) { done(false, 'writeText ditolak: ' + (e && e.message ? e.message : String(e))); });
    }
  });

  define({
    id: 'input-clipboard-read', category: 'Input', name: 'Clipboard readText', kind: 'manual', timeoutMs: 8000,
    run: function (done) {
      if (!navigator.clipboard || typeof navigator.clipboard.readText !== 'function') {
        return done(false, 'navigator.clipboard.readText tidak tersedia');
      }
      navigator.clipboard.readText().then(function (t) {
        done(true, 'readText => "' + t + '"');
      }).catch(function (e) { done(false, 'readText ditolak (izin user?): ' + (e && e.message ? e.message : String(e))); });
    }
  });

  define({
    id: 'input-exec-copy', category: 'Input', name: 'document.execCommand("copy")', kind: 'auto',
    run: function (done) {
      try {
        var ta = document.createElement('textarea');
        ta.value = 'respect-exec-copy';
        ta.style.cssText = 'position:fixed;left:-9999px;top:0;';
        document.body.appendChild(ta);
        ta.focus(); ta.select();
        var ok = false;
        try { ok = document.execCommand('copy'); } catch (e) { ok = false; }
        document.body.removeChild(ta);
        done(ok, 'execCommand("copy") = ' + ok);
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-drag-drop', category: 'Input', name: 'Drag & Drop events + DataTransfer', kind: 'auto',
    run: function (done) {
      try {
        if (typeof DataTransfer !== 'function') return done(false, 'DataTransfer tidak tersedia');
        var zone = document.createElement('div');
        document.body.appendChild(zone);
        var got = { dragstart: false, drop: false, data: null };
        zone.addEventListener('dragstart', function (e) {
          got.dragstart = true;
          try { e.dataTransfer.setData('text/plain', 'respect-drag'); } catch (x) {}
        });
        zone.addEventListener('drop', function (e) {
          got.drop = true;
          e.preventDefault();
          try { got.data = e.dataTransfer.getData('text/plain'); } catch (x) {}
        });
        var dt = new DataTransfer();
        var dragEv = new Event('dragstart', { bubbles: true, cancelable: true });
        dragEv.dataTransfer = dt;
        zone.dispatchEvent(dragEv);
        var dropEv = new Event('drop', { bubbles: true, cancelable: true });
        dropEv.dataTransfer = dt;
        zone.dispatchEvent(dropEv);
        document.body.removeChild(zone);
        done(got.dragstart && got.drop, 'dragstart=' + got.dragstart + ' drop=' + got.drop + ' data="' + got.data + '"');
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-mouse-coords', category: 'Input', name: 'MouseEvent client coords dispatch', kind: 'auto',
    run: function (done) {
      try {
        var box = document.createElement('div');
        box.style.cssText = 'width:20px;height:20px;';
        document.body.appendChild(box);
        var got = null;
        box.addEventListener('mousemove', function (e) { got = e.clientX + ',' + e.clientY; });
        box.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, clientX: 123, clientY: 45 }));
        var clickFired = false;
        box.addEventListener('click', function () { clickFired = true; });
        box.dispatchEvent(new MouseEvent('click', { bubbles: true }));
        document.body.removeChild(box);
        done(got === '123,45' && clickFired, 'mousemove coords=' + got + ' click=' + clickFired);
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-pointer-touch', category: 'Input', name: 'PointerEvent / TouchEvent support', kind: 'auto',
    run: function (done) {
      var parts = [];
      parts.push('PointerEvent=' + (typeof window.PointerEvent === 'function'));
      parts.push('TouchEvent=' + (typeof window.TouchEvent === 'function'));
      parts.push('ontouchstart=' + ('ontouchstart' in window));
      var pass = (typeof window.PointerEvent === 'function') || ('ontouchstart' in window);
      done(pass, parts.join(' '));
    }
  });

  define({
    id: 'input-focus-blur', category: 'Input', name: 'focus / blur events', kind: 'auto',
    run: function (done) {
      try {
        var i = document.createElement('input');
        document.body.appendChild(i);
        var got = { focus: 0, blur: 0 };
        i.addEventListener('focus', function () { got.focus++; });
        i.addEventListener('blur', function () { got.blur++; });
        i.focus();
        i.blur();
        document.body.removeChild(i);
        done(got.focus === 1 && got.blur === 1, safeStr(got));
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-contextmenu', category: 'Input', name: 'contextmenu event + preventDefault', kind: 'auto',
    run: function (done) {
      try {
        var d = document.createElement('div');
        document.body.appendChild(d);
        var fired = false, prevented = false;
        d.addEventListener('contextmenu', function (e) { fired = true; e.preventDefault(); prevented = e.defaultPrevented; });
        d.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
        document.body.removeChild(d);
        done(fired && prevented, 'fired=' + fired + ' defaultPrevented=' + prevented);
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'input-selection-range', category: 'Input', name: 'Selection & Range API', kind: 'auto',
    run: function (done) {
      try {
        if (typeof document.createRange !== 'function' || !window.getSelection) return done(false, 'Range/Selection tidak tersedia');
        var p = document.createElement('p');
        p.textContent = 'Respect selection probe';
        p.style.cssText = 'position:fixed;left:-9999px;';
        document.body.appendChild(p);
        var range = document.createRange();
        range.selectNodeContents(p);
        var sel = window.getSelection();
        sel.removeAllRanges();
        sel.addRange(range);
        var count = sel.rangeCount;
        var text = String(sel);
        sel.removeAllRanges();
        document.body.removeChild(p);
        var ok = count > 0 && text.indexOf('selection') !== -1;
        done(ok, 'rangeCount=' + count + ' text="' + text + '"');
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  // -------------------------------------------------------------------------
  // 3. DIALOG (manual - dapat memblokir UI / memicu native dialog)
  // -------------------------------------------------------------------------
  define({
    id: 'dialog-alert', category: 'Dialog', name: 'alert() native dialog', kind: 'manual', danger: true, timeoutMs: 60000,
    run: function (done) {
      try {
        window.alert('Respect Stress Test\n\nIni dialog alert(). Klik OK untuk melanjutkan.');
        done(true, 'alert() kembali normal (tidak crash / tidak hang)');
      } catch (e) { done(false, 'alert() melempar exception: ' + e.message); }
    }
  });

  define({
    id: 'dialog-confirm', category: 'Dialog', name: 'confirm() return value', kind: 'manual', danger: true, timeoutMs: 60000,
    run: function (done) {
      try {
        var r = window.confirm('Respect Stress Test\n\nKlik OK atau Cancel untuk menguji nilai kembalian confirm().');
        done(true, 'confirm() mengembalikan: ' + r);
      } catch (e) { done(false, 'confirm() melempar exception: ' + e.message); }
    }
  });

  define({
    id: 'dialog-prompt', category: 'Dialog', name: 'prompt() return value', kind: 'manual', danger: true, timeoutMs: 60000,
    run: function (done) {
      try {
        var r = window.prompt('Respect Stress Test\n\nKetik sesuatu untuk menguji prompt().', 'respect');
        done(true, 'prompt() mengembalikan: ' + (r === null ? 'null (dibatalkan)' : '"' + r + '"'));
      } catch (e) { done(false, 'prompt() melempar exception: ' + e.message); }
    }
  });

  define({
    id: 'dialog-print', category: 'Dialog', name: 'window.print()', kind: 'manual', danger: true, timeoutMs: 60000,
    run: function (done) {
      try { window.print(); done(true, 'print() dipanggil tanpa exception'); }
      catch (e) { done(false, 'print() exception: ' + e.message); }
    }
  });

  define({
    id: 'dialog-window-open', category: 'Dialog', name: 'window.open() popup (HAZARD)', kind: 'manual', danger: true, hazard: true, timeoutMs: 30000,
    run: function (done) {
      try {
        var w = window.open('about:blank', 'respect_probe_popup', 'width=420,height=320,noopener');
        setTimeout(function () {
          if (w && w !== window) {
            try { w.close(); } catch (e) {}
            done(true, 'Popup terpisah terbuka (window.open non-null, target window berbeda)');
          } else if (w) {
            done(true, 'window.open mengembalikan objek, TAPI engine ini dapat menimpa window yang sama (berpotensi freeze - perlu restart)');
          } else {
            done(false, 'Popup diblokir (window.open mengembalikan null), window utama aman');
          }
        }, 800);
      } catch (e) { done(false, 'window.open exception: ' + e.message); }
    }
  });

  // -------------------------------------------------------------------------
  // 4. JS BRIDGE (mbQuery / ipc)
  // -------------------------------------------------------------------------
  define({
    id: 'bridge-detect', category: 'Bridge', name: 'Deteksi hook native (mbQuery/ipc/chrome)', kind: 'auto',
    run: function (done) {
      var info = {
        mbQuery: typeof window.mbQuery,
        ipc: typeof window.ipc,
        chrome: typeof window.chrome,
        respect: typeof window.respect,
        mbQueryArgCount: (typeof window.mbQuery === 'function') ? window.mbQuery.length : -1
      };
      var present = info.mbQuery === 'function' || info.ipc !== 'undefined';
      done(present, safeStr(info));
    }
  });

  define({
    id: 'bridge-mbquery-call', category: 'Bridge', name: 'mbQuery() round-trip (Modern)', kind: 'manual', timeoutMs: 8000,
    run: function (done) {
      if (typeof window.mbQuery !== 'function') return done(false, 'window.mbQuery tidak tersedia (bukan engine Modern / handler belum dipasang)');
      var settled = false;
      function fin(pass, detail) { if (settled) return; settled = true; done(pass, detail); }
      try {
        var ret = window.mbQuery('respect_stress_ping', 'hello-from-suite', function (resp) {
          fin(true, 'Callback mbQuery dipanggil: ' + safeStr(resp));
        });
        setTimeout(function () {
          fin(true, 'mbQuery dieksekusi tanpa callback (return=' + safeStr(ret) + ')');
        }, 3000);
      } catch (e) {
        fin(false, 'mbQuery melempar exception: ' + e.message);
      }
    }
  });

  define({
    id: 'bridge-ipc-call', category: 'Bridge', name: 'ipc.invoke() round-trip (Lite)', kind: 'manual', timeoutMs: 8000,
    run: function (done) {
      if (!window.ipc || typeof window.ipc.invoke !== 'function') return done(false, 'window.ipc.invoke tidak tersedia (bukan engine Lite)');
      try {
        var p = window.ipc.invoke('respect_stress_ping', { hello: 'suite' });
        if (p && typeof p.then === 'function') {
          p.then(function (r) { done(true, 'ipc.invoke resolve: ' + safeStr(r)); })
           .catch(function (e) { done(false, 'ipc.invoke reject: ' + (e && e.message ? e.message : String(e))); });
        } else {
          done(true, 'ipc.invoke dipanggil, return=' + safeStr(p));
        }
      } catch (e) { done(false, 'ipc.invoke exception: ' + e.message); }
    }
  });

  // -------------------------------------------------------------------------
  // 5. STORAGE (auto)
  // -------------------------------------------------------------------------
  define({
    id: 'store-local', category: 'Storage', name: 'localStorage round-trip', kind: 'auto',
    run: function (done) {
      try {
        var k = '__respect_fl_local__';
        localStorage.setItem(k, 'v-' + Date.now());
        var v = localStorage.getItem(k);
        localStorage.removeItem(k);
        done(!!v, 'nilai dibaca kembali: ' + v);
      } catch (e) { done(false, 'localStorage exception: ' + e.message); }
    }
  });

  define({
    id: 'store-session', category: 'Storage', name: 'sessionStorage round-trip', kind: 'auto',
    run: function (done) {
      try {
        var k = '__respect_fl_session__';
        sessionStorage.setItem(k, 'sess-1');
        var v = sessionStorage.getItem(k);
        sessionStorage.removeItem(k);
        done(v === 'sess-1', 'nilai dibaca kembali: ' + v);
      } catch (e) { done(false, 'sessionStorage exception: ' + e.message); }
    }
  });

  define({
    id: 'store-cookie', category: 'Storage', name: 'document.cookie round-trip', kind: 'auto',
    run: function (done) {
      try {
        var name = '__respect_probe_cookie__';
        document.cookie = name + '=ok; path=/';
        var has = document.cookie.indexOf(name + '=ok') !== -1;
        document.cookie = name + '=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/';
        done(has, 'cookie terpasang=' + has + ' | cookieEnabled=' + navigator.cookieEnabled);
      } catch (e) { done(false, 'cookie exception: ' + e.message); }
    }
  });

  define({
    id: 'store-indexeddb', category: 'Storage', name: 'IndexedDB put/get round-trip', kind: 'auto', timeoutMs: 10000,
    run: function (done) {
      if (!window.indexedDB) return done(false, 'IndexedDB tidak tersedia');
      try {
        var req = indexedDB.open('__respect_fl_db__', 1);
        req.onupgradeneeded = function (e) {
          var db = e.target.result;
          if (!db.objectStoreNames.contains('kv')) db.createObjectStore('kv');
        };
        req.onerror = function () { done(false, 'open error'); };
        req.onsuccess = function (e) {
          var db = e.target.result;
          try {
            var tx = db.transaction('kv', 'readwrite');
            tx.objectStore('kv').put('idb-value', 'key1');
            tx.oncomplete = function () {
              var tx2 = db.transaction('kv', 'readonly');
              var g = tx2.objectStore('kv').get('key1');
              g.onsuccess = function () {
                var v = g.result;
                db.close();
                done(v === 'idb-value', 'IndexedDB get => "' + v + '"');
              };
              g.onerror = function () { db.close(); done(false, 'get error'); };
            };
            tx.onerror = function () { db.close(); done(false, 'write transaction error'); };
          } catch (err) { db.close(); done(false, 'transaction exception: ' + err.message); }
        };
      } catch (e) { done(false, 'indexedDB exception: ' + e.message); }
    }
  });

  define({
    id: 'store-cache', category: 'Storage', name: 'CacheStorage API round-trip', kind: 'auto', timeoutMs: 10000,
    run: function (done) {
      if (!('caches' in window)) return done(false, 'CacheStorage (caches) tidak tersedia');
      try {
        caches.open('__respect_fl_cache__').then(function (cache) {
          return cache.put('probe-key', new Response('cache-ok'));
        }).then(function () {
          return caches.open('__respect_fl_cache__');
        }).then(function (cache) {
          return cache.match('probe-key');
        }).then(function (resp) {
          if (!resp) return done(false, 'cache.match tidak menemukan entry');
          return resp.text().then(function (t) {
            caches.delete('__respect_fl_cache__');
            done(t === 'cache-ok', 'Cache entry dibaca => "' + t + '"');
          });
        }).catch(function (e) { done(false, 'CacheStorage error: ' + (e && e.message ? e.message : String(e))); });
      } catch (e) { done(false, 'CacheStorage exception: ' + e.message); }
    }
  });

  define({
    id: 'store-opfs', category: 'Storage', name: 'OPFS getDirectory + write/read', kind: 'auto', timeoutMs: 10000,
    run: function (done) {
      if (!navigator.storage || typeof navigator.storage.getDirectory !== 'function') {
        return done(false, 'Origin Private File System tidak tersedia');
      }
      navigator.storage.getDirectory().then(function (root) {
        return root.getFileHandle('respect-probe.txt', { create: true }).then(function (fh) {
          return fh.createWritable().then(function (w) {
            return w.write('opfs-ok').then(function () { return w.close(); });
          }).then(function () {
            return fh.getFile().then(function (file) {
              return file.text().then(function (t) {
                done(/opfs-ok/.test(t), 'OPFS baca kembali: "' + t + '"');
              });
            });
          });
        });
      }).catch(function (e) { done(false, 'OPFS error: ' + (e && e.message ? e.message : String(e))); });
    }
  });

  // -------------------------------------------------------------------------
  // 6. WEB WORKER (auto)
  // -------------------------------------------------------------------------
  define({
    id: 'worker-ping', category: 'Concurrency', name: 'Web Worker ping/pong', kind: 'auto', timeoutMs: 8000,
    run: function (done) {
      if (typeof Worker !== 'function') return done(false, 'Worker tidak tersedia');
      try {
        var w = new Worker('worker.js');
        var t = setTimeout(function () { try { w.terminate(); } catch (e) {} done(false, 'TIMEOUT worker ping'); }, 6000);
        w.onmessage = function (e) {
          clearTimeout(t);
          try { w.terminate(); } catch (x) {}
          done(e.data && e.data.cmd === 'pong', 'pong diterima: ' + safeStr(e.data));
        };
        w.onerror = function (ev) { clearTimeout(t); try { w.terminate(); } catch (x) {} done(false, 'worker error: ' + (ev.message || 'unknown')); };
        w.postMessage({ cmd: 'ping', taskId: 'fl_ping' });
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  define({
    id: 'worker-fetch', category: 'Concurrency', name: 'Web Worker relative fetch (routing di worker)', kind: 'auto', timeoutMs: 10000,
    run: function (done) {
      if (typeof Worker !== 'function') return done(false, 'Worker tidak tersedia');
      try {
        var w = new Worker('worker.js');
        var t = setTimeout(function () { try { w.terminate(); } catch (e) {} done(false, 'TIMEOUT worker fetch'); }, 8000);
        w.onmessage = function (e) {
          clearTimeout(t);
          try { w.terminate(); } catch (x) {}
          var d = e.data || {};
          if (d.cmd === 'fetch_result') done(!!d.ok, d.ok ? ('status=' + d.status + ' bytes=' + d.length) : ('error: ' + d.error));
          else done(false, 'respons tidak dikenal: ' + safeStr(d));
        };
        w.onerror = function (ev) { clearTimeout(t); try { w.terminate(); } catch (x) {} done(false, 'worker error: ' + (ev.message || 'unknown')); };
        w.postMessage({ cmd: 'fetch_probe', taskId: 'fl_fetch', url: 'data.json' });
      } catch (e) { done(false, 'Exception: ' + e.message); }
    }
  });

  // -------------------------------------------------------------------------
  // 7. MEDIA & HARDWARE (kamera/mic/synth/TTS/hardware)
  // -------------------------------------------------------------------------
  define({
    id: 'media-getusermedia', category: 'Media', name: 'navigator.mediaDevices.getUserMedia presence', kind: 'auto',
    run: function (done) {
      var md = navigator.mediaDevices;
      if (!md || typeof md.getUserMedia !== 'function') {
        return done(false, 'getUserMedia tidak tersedia (secure context / engine non-Mediasi)');
      }
      var devs = typeof md.enumerateDevices === 'function';
      done(true, 'getUserMedia tersedia, enumerateDevices=' + devs);
    }
  });

  define({
    id: 'media-camera-cycle', category: 'Media', name: 'Camera start -> snapshot -> stop (anti-crash)', kind: 'manual', timeoutMs: 12000,
    run: function (done) {
      if (typeof MediaLabEngine === 'undefined') return done(false, 'MediaLabEngine tidak tersedia');
      var videoEl = el('cameraVideo');
      var statusEl = el('cameraStatus');
      var canvasEl = el('cameraCanvas');
      var settled = false;
      function fin(pass, detail) { if (settled) return; settled = true; done(pass, detail); }
      try {
        // Pastikan state bersih dulu agar tidak berhenti di kondisi "already running".
        try { MediaLabEngine.stopCamera(videoEl, statusEl); } catch (e) {}
        MediaLabEngine.startCamera(videoEl, statusEl, function (ok, res) {
          if (!ok) return fin(false, (res && res.message) || 'kamera ditolak');
          setTimeout(function () {
            try { MediaLabEngine.snapshotCamera(videoEl, canvasEl, 'grayscale'); } catch (e) {}
            try {
              MediaLabEngine.stopCamera(videoEl, statusEl);
              fin(true, 'start + snapshot + stop selesai tanpa crash' + (res && res.isVirtual ? ' (virtual feed)' : '') + (res && res.alreadyRunning ? ' [mode already-running]' : ''));
            } catch (e) {
              fin(false, 'stopCamera exception: ' + e.message);
            }
          }, 700);
        });
      } catch (e) {
        fin(false, 'startCamera exception: ' + e.message);
      }
    }
  });

  define({
    id: 'media-mic-cycle', category: 'Media', name: 'Microphone start -> stop', kind: 'manual', timeoutMs: 20000,
    run: function (done) {
      if (typeof MediaLabEngine === 'undefined') return done(false, 'MediaLabEngine tidak tersedia');
      var canvasEl = el('micCanvas');
      var meterEl = el('micMeterFill');
      var statusEl = el('micStatus');
      var settled = false;
      function fin(pass, detail) { if (settled) return; settled = true; done(pass, detail); }
      try {
        MediaLabEngine.startMicrophone(canvasEl, meterEl, statusEl, function (ok, err) {
          if (!ok) return fin(false, (err && err.message) || 'mikrofon ditolak');
          setTimeout(function () {
            try { MediaLabEngine.stopMicrophone(statusEl); fin(true, 'mic start + stop selesai'); }
            catch (e) { fin(false, 'stopMicrophone exception: ' + e.message); }
          }, 700);
        });
      } catch (e) { fin(false, 'startMicrophone exception: ' + e.message); }
    }
  });

  define({
    id: 'media-synth-cycle', category: 'Media', name: 'Web Audio synth start -> stop', kind: 'manual', timeoutMs: 15000,
    run: function (done) {
      if (typeof MediaLabEngine === 'undefined') return done(false, 'MediaLabEngine tidak tersedia');
      var AC = window.AudioContext || window.webkitAudioContext;
      if (typeof AC !== 'function') return done(false, 'Web Audio API (AudioContext) tidak tersedia');
      // Verifikasi AudioContext bukan mock: createOscillator harus ada.
      var probeCtx = null;
      try { probeCtx = new AC(); } catch (e) { return done(false, 'AudioContext gagal dibuat: ' + e.message); }
      var hasOsc = typeof probeCtx.createOscillator === 'function';
      var hasGain = typeof probeCtx.createGain === 'function';
      try { if (probeCtx.close) probeCtx.close(); } catch (e) {}
      if (!hasOsc) return done(false, 'createOscillator() tidak tersedia (AudioContext mock/no-op di engine ini)');
      var statusEl = el('synthStatus');
      try {
        MediaLabEngine.startSynth(440, 'sine', 0.05, statusEl);
        setTimeout(function () {
          try { MediaLabEngine.stopSynth(statusEl); done(true, 'synth start + stop selesai (createOscillator=' + hasOsc + ' createGain=' + hasGain + ')'); }
          catch (e) { done(false, 'stopSynth exception: ' + e.message); }
        }, 700);
      } catch (e) { done(false, 'startSynth exception: ' + e.message); }
    }
  });

  define({
    id: 'media-tts', category: 'Media', name: 'Speech Synthesis speak + cancel', kind: 'manual', timeoutMs: 15000,
    run: function (done) {
      if (typeof MediaLabEngine === 'undefined') return done(false, 'MediaLabEngine tidak tersedia');
      if (!('speechSynthesis' in window) || typeof window.speechSynthesis.speak !== 'function') {
        return done(false, 'speechSynthesis tidak tersedia di engine ini');
      }
      var statusEl = el('ttsStatus');
      try {
        MediaLabEngine.speakText('Respect stress testing speech probe.', 1.0, 1.0, statusEl);
        setTimeout(function () {
          try { MediaLabEngine.stopSpeech(); done(true, 'speakText dipanggil & cancel sukses'); }
          catch (e) { done(false, 'stopSpeech exception: ' + e.message); }
        }, 1200);
      } catch (e) { done(false, 'speakText exception: ' + e.message); }
    }
  });

  define({
    id: 'media-voices', category: 'Media', name: 'speechSynthesis voice list', kind: 'auto',
    run: function (done) {
      if (!('speechSynthesis' in window)) return done(false, 'speechSynthesis tidak tersedia');
      try {
        var voices = window.speechSynthesis.getVoices();
        done(true, 'voices=' + (voices ? voices.length : 0) + ' (bisa 0 saat daftar belum dimuat)');
      } catch (e) { done(false, 'getVoices exception: ' + e.message); }
    }
  });

  define({
    id: 'media-fullscreen', category: 'Media', name: 'Fullscreen request/exit', kind: 'manual', timeoutMs: 15000,
    run: function (done) {
      var root = document.documentElement;
      var fn = root.requestFullscreen || root.webkitRequestFullscreen || root.msRequestFullscreen;
      if (!fn) return done(false, 'Fullscreen API tidak tersedia');
      try {
        var p = fn.call(root);
        setTimeout(function () {
          var ex = document.exitFullscreen || document.webkitExitFullscreen || document.msExitFullscreen;
          var active = !!(document.fullscreenElement || document.webkitFullscreenElement);
          if (ex) {
            try {
              var exRes = ex.call(document);
              if (exRes && typeof exRes.then === 'function') exRes.catch(function () {});
            } catch (e) {}
          }
          done(true, 'Fullscreen aktif=' + active + ' lalu exit dijalankan');
        }, 900);
        if (p && typeof p.then === 'function') p.catch(function () {});
      } catch (e) { done(false, 'Fullscreen exception: ' + e.message); }
    }
  });

  define({
    id: 'media-pip', category: 'Media', name: 'Picture-in-Picture (testVideo)', kind: 'manual', timeoutMs: 15000,
    run: function (done) {
      var v = el('testVideo');
      if (!v) return done(false, 'Elemen video tidak ditemukan');
      if (typeof v.requestPictureInPicture !== 'function') return done(false, 'PiP tidak didukung');
      try {
        var pip = v.requestPictureInPicture();
        if (!pip || typeof pip.then !== 'function') {
          return done(true, 'requestPictureInPicture() ada tetapi tidak mengembalikan Promise (mock/no-op di engine ini)');
        }
        pip.then(function () {
          if (document.exitPictureInPicture) {
            try {
              var exPip = document.exitPictureInPicture();
              if (exPip && typeof exPip.then === 'function') exPip.catch(function () {});
            } catch (e) {}
          }
          done(true, 'Picture-in-Picture aktif lalu ditutup');
        }).catch(function (e) { done(false, 'PiP ditolak: ' + (e && e.message ? e.message : String(e))); });
      } catch (e) { done(false, 'PiP exception: ' + e.message); }
    }
  });

  define({
    id: 'media-notification', category: 'Media', name: 'Notification permission request', kind: 'manual', timeoutMs: 30000,
    run: function (done) {
      if (typeof Notification === 'undefined') return done(false, 'Notification API tidak tersedia');
      try {
        Notification.requestPermission().then(function (perm) {
          done(true, 'Notification permission => ' + perm);
        }).catch(function (e) { done(false, 'requestPermission error: ' + (e && e.message ? e.message : String(e))); });
      } catch (e) { done(false, 'Notification exception: ' + e.message); }
    }
  });

  define({
    id: 'media-geolocation', category: 'Media', name: 'Geolocation getCurrentPosition', kind: 'manual', timeoutMs: 30000,
    run: function (done) {
      if (!navigator.geolocation || typeof navigator.geolocation.getCurrentPosition !== 'function') {
        return done(false, 'Geolocation tidak tersedia');
      }
      try {
        navigator.geolocation.getCurrentPosition(function (pos) {
          done(true, 'lat=' + pos.coords.latitude.toFixed(4) + ' lon=' + pos.coords.longitude.toFixed(4));
        }, function (err) {
          done(false, 'Geolocation error: ' + err.code + ' ' + err.message + ' (ditolak/timeout)');
        }, { timeout: 12000, maximumAge: 60000 });
      } catch (e) { done(false, 'Geolocation exception: ' + e.message); }
    }
  });

  define({
    id: 'media-vibrate', category: 'Media', name: 'Vibration API', kind: 'auto',
    run: function (done) {
      if (!navigator.vibrate) return done(false, 'navigator.vibrate tidak tersedia');
      try { var r = navigator.vibrate(150); done(true, 'navigator.vibrate(150) => ' + r); }
      catch (e) { done(false, 'vibrate exception: ' + e.message); }
    }
  });

  define({
    id: 'media-wakelock', category: 'Media', name: 'Screen Wake Lock request', kind: 'manual', timeoutMs: 15000,
    run: function (done) {
      if (!navigator.wakeLock || typeof navigator.wakeLock.request !== 'function') return done(false, 'Wake Lock tidak tersedia');
      try {
        navigator.wakeLock.request('screen').then(function (sentinel) {
          try { if (sentinel && sentinel.release) sentinel.release(); } catch (e) {}
          done(true, 'Wake Lock aktif lalu dirilis');
        }).catch(function (e) { done(false, 'Wake Lock ditolak: ' + (e && e.message ? e.message : String(e))); });
      } catch (e) { done(false, 'Wake Lock exception: ' + e.message); }
    }
  });

  // -------------------------------------------------------------------------
  // 8. DIAGNOSTIC / TELEMETRY (auto)
  // -------------------------------------------------------------------------
  define({
    id: 'diag-timing', category: 'Diagnostics', name: 'Performance navigation timing', kind: 'auto',
    run: function (done) {
      try {
        if (!window.performance || !performance.timing) return done(false, 'performance.timing tidak tersedia');
        var t = performance.timing;
        var load = t.loadEventEnd - t.navigationStart;
        done(true, 'DOMContentLoaded=' + (t.domContentLoadedEventEnd - t.navigationStart) + 'ms, load=' + load + 'ms, now=' + performance.now().toFixed(1) + 'ms');
      } catch (e) { done(false, 'performance exception: ' + e.message); }
    }
  });

  define({
    id: 'diag-screen', category: 'Diagnostics', name: 'Screen & viewport geometry', kind: 'auto',
    run: function (done) {
      try {
        var s = window.screen;
        done(true, safeStr({
          screen: s.width + 'x' + s.height, avail: s.availWidth + 'x' + s.availHeight,
          colorDepth: s.colorDepth, dpr: window.devicePixelRatio,
          inner: window.innerWidth + 'x' + window.innerHeight,
          outer: window.outerWidth + 'x' + window.outerHeight,
          orientation: (s.orientation && s.orientation.type) || 'n/a'
        }));
      } catch (e) { done(false, 'screen exception: ' + e.message); }
    }
  });

  define({
    id: 'diag-observer', category: 'Diagnostics', name: 'MutationObserver live callback', kind: 'auto', timeoutMs: 5000,
    run: function (done) {
      if (typeof MutationObserver !== 'function') return done(false, 'MutationObserver tidak tersedia');
      try {
        var target = document.createElement('div');
        document.body.appendChild(target);
        var fired = false;
        var mo = new MutationObserver(function () { fired = true; });
        mo.observe(target, { childList: true });
        target.appendChild(document.createElement('span'));
        setTimeout(function () {
          mo.disconnect();
          document.body.removeChild(target);
          done(fired, 'MutationObserver callback fired=' + fired);
        }, 60);
      } catch (e) { done(false, 'MutationObserver exception: ' + e.message); }
    }
  });

  // -------------------------------------------------------------------------
  // PUBLIC API
  // -------------------------------------------------------------------------
  return {
    define: define,
    list: list,
    get: get,
    run: run,
    runAll: runAll,
    abort: abort,
    resetAbort: resetAbort,
    isAborted: isAborted
  };
})();

if (typeof window !== 'undefined') {
  window.FeatureLabEngine = FeatureLabEngine;
}
