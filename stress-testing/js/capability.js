/**
 * Respect Browser Stress Testing Suite - Exhaustive Capability Engine (120+ Tests)
 * Menguji fitur-fitur browser secara dinamis dan aman tanpa syntax-error di engine lama.
 */

var CapabilityEngine = (function () {
  'use strict';

  function tryEval(code) {
    return function () {
      /* eslint-disable no-new-func */
      var f = new Function(code);
      return !!f();
    };
  }

  function hasProp(obj, prop) {
    return function () {
      return !!(obj && (prop in obj || typeof obj[prop] !== 'undefined'));
    };
  }

  function cssSupports(prop, value) {
    return function () {
      if (window.CSS && typeof window.CSS.supports === 'function') {
        return window.CSS.supports(prop, value);
      }
      var el = document.createElement('div');
      try {
        el.style[prop] = value;
      } catch (e) { return false; }
      return el.style[prop] !== '' && el.style[prop] !== undefined;
    };
  }

  // Deteksi at-rule (@layer, @scope, @property, ...) via parsing stylesheet nyata.
  function cssAtRuleSupported(ruleText) {
    return function () {
      try {
        var s = document.createElement('style');
        s.textContent = ruleText;
        document.head.appendChild(s);
        var count = s.sheet ? s.sheet.cssRules.length : 0;
        document.head.removeChild(s);
        return count > 0;
      } catch (e) { return false; }
    };
  }

  function runAllTests() {
    var tests = [];

    function check(category, id, name, testFn, note) {
      var pass = false;
      var err = '';
      try {
        pass = !!testFn();
      } catch (e) {
        pass = false;
        err = e.message || String(e);
      }
      tests.push({
        category: category,
        id: id,
        name: name,
        pass: pass,
        note: note || (pass ? 'Supported' : (err || 'Not available'))
      });
    }

    /* =========================================================================
       1. JAVASCRIPT & ECMASCRIPT STANDARDS
       ========================================================================= */
    check('JavaScript', 'let-const', 'let & const (Block Scope)', tryEval('let a = 1; const b = 2; return a + b === 3;'));
    check('JavaScript', 'arrow-fn', 'Arrow Functions', tryEval('return ((x) => x * 2)(4) === 8;'));
    check('JavaScript', 'template-literals', 'Template Literals (`${x}`)', tryEval('var val = 42; return `num:${val}` === "num:42";'));
    check('JavaScript', 'destructuring', 'Array & Object Destructuring', tryEval('const [x, ...y] = [1, 2, 3]; const {a} = {a: 9}; return x === 1 && y.length === 2 && a === 9;'));
    check('JavaScript', 'default-params', 'Default Function Parameters', tryEval('function f(x = 10){ return x; } return f() === 10;'));
    check('JavaScript', 'classes', 'ES6 Classes & Inheritance', tryEval('class A { foo() { return 1; } } class B extends A {} return (new B()).foo() === 1;'));
    check('JavaScript', 'promise', 'Promise & Microtasks', hasProp(window, 'Promise'));
    check('JavaScript', 'async-await', 'async / await Syntax', tryEval('return typeof (async function() { return true; }) === "function";'));
    check('JavaScript', 'generators', 'Generators & Iterators (function*)', tryEval('return typeof (function*() { yield 1; }) === "function";'));
    check('JavaScript', 'symbol', 'Symbol Primitive', hasProp(window, 'Symbol'));
    check('JavaScript', 'map-set', 'Map, Set, WeakMap, WeakSet', function () {
      return typeof Map !== 'undefined' && typeof Set !== 'undefined' && typeof WeakMap !== 'undefined' && typeof WeakSet !== 'undefined';
    });
    check('JavaScript', 'proxy-reflect', 'Proxy & Reflect Metaprogramming', function () {
      return typeof Proxy !== 'undefined' && typeof Reflect !== 'undefined';
    });
    check('JavaScript', 'optional-chaining', 'Optional Chaining (?.)', tryEval('var obj = {}; return obj?.nonExistent?.property === undefined;'));
    check('JavaScript', 'nullish-coalescing', 'Nullish Coalescing (??)', tryEval('return (null ?? 10) === 10 && (0 ?? 20) === 0;'));
    check('JavaScript', 'logical-assignment', 'Logical Assignment (||=, &&=, ??=)', tryEval('let a = 0; a ||= 5; let b = 1; b &&= 2; return a === 5 && b === 2;'));
    check('JavaScript', 'numeric-separators', 'Numeric Separators (1_000_000)', tryEval('return 1_000_000 === 1000000;'));
    check('JavaScript', 'bigint', 'BigInt (64-bit+ integers)', hasProp(window, 'BigInt'));
    check('JavaScript', 'array-at', 'Array.prototype.at()', function () { return typeof Array.prototype.at === 'function'; });
    check('JavaScript', 'array-flat', 'Array.prototype.flat / flatMap', function () { return typeof Array.prototype.flat === 'function' && typeof Array.prototype.flatMap === 'function'; });
    check('JavaScript', 'array-tosorted', 'Array immutable methods (toSorted, toReversed)', function () { return typeof Array.prototype.toSorted === 'function'; });
    check('JavaScript', 'object-fromentries', 'Object.fromEntries()', function () { return typeof Object.fromEntries === 'function'; });
    check('JavaScript', 'object-hasown', 'Object.hasOwn()', function () { return typeof Object.hasOwn === 'function'; });
    check('JavaScript', 'string-replaceall', 'String.prototype.replaceAll()', function () { return typeof String.prototype.replaceAll === 'function'; });
    check('JavaScript', 'promise-allsettled', 'Promise.allSettled()', function () { return typeof Promise.allSettled === 'function'; });
    check('JavaScript', 'promise-any', 'Promise.any()', function () { return typeof Promise.any === 'function'; });
    check('JavaScript', 'globalthis', 'globalThis identifier', function () { return typeof globalThis !== 'undefined'; });
    check('JavaScript', 'structured-clone', 'structuredClone() Deep Copy', function () { return typeof window.structuredClone === 'function'; });
    check('JavaScript', 'queue-microtask', 'queueMicrotask()', function () { return typeof window.queueMicrotask === 'function'; });
    check('JavaScript', 'weakref', 'WeakRef & FinalizationRegistry', function () { return typeof WeakRef !== 'undefined' && typeof FinalizationRegistry !== 'undefined'; });
    check('JavaScript', 'intl', 'Intl Internationalization API', hasProp(window, 'Intl'));
    check('JavaScript', 'wasm', 'WebAssembly (WASM Core)', hasProp(window, 'WebAssembly'));
    check('JavaScript', 'wasm-simd', 'WebAssembly SIMD Support', tryEval('return WebAssembly.validate(new Uint8Array([0,97,115,109,1,0,0,0,1,5,1,96,0,1,123,3,2,1,0,10,22,1,20,0,253,12,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,11]));'));

    /* =========================================================================
       2. MODERN CSS ENGINE STANDARDS
       ========================================================================= */
    check('CSS', 'flexbox', 'CSS Flexbox (display: flex)', cssSupports('display', 'flex'));
    check('CSS', 'grid', 'CSS Grid Layout (display: grid)', cssSupports('display', 'grid'));
    check('CSS', 'subgrid', 'CSS Grid Subgrid', cssSupports('grid-template-columns', 'subgrid'));
    check('CSS', 'custom-props', 'CSS Variables / Custom Properties (--var)', cssSupports('--css-var', '1'));
    check('CSS', 'gap-flex', 'CSS Flexbox Gap', cssSupports('gap', '10px'));
    check('CSS', 'aspect-ratio', 'aspect-ratio Property', cssSupports('aspect-ratio', '16/9'));
    check('CSS', 'backdrop-filter', 'backdrop-filter (Blur, Glassmorphism)', cssSupports('backdrop-filter', 'blur(10px)'));
    check('CSS', 'clip-path', 'clip-path Polygons & Shapes', cssSupports('clip-path', 'circle(50%)'));
    check('CSS', 'container-queries', 'Container Queries (@container)', cssSupports('container-type', 'inline-size'));
    check('CSS', 'has-pseudo', ':has() Parent Selector', function () {
      try { return !!document.querySelector(':root:has(body)'); } catch (e) { return false; }
    });
    check('CSS', 'is-where-pseudo', ':is() & :where() Selectors', function () {
      try { return !!document.querySelector(':is(html, body)'); } catch (e) { return false; }
    });
    check('CSS', 'accent-color', 'accent-color Form Theming', cssSupports('accent-color', '#3b82f6'));
    check('CSS', 'color-mix', 'color-mix() Color Manipulation', cssSupports('color', 'color-mix(in srgb, red 50%, blue 50%)'));
    check('CSS', 'oklch-color', 'OKLCH / P3 Wide Gamut Colors', cssSupports('color', 'oklch(0.6 0.25 140)'));
    check('CSS', 'native-nesting', 'Native CSS Nesting (&)', function () {
      try {
        var s = document.createElement('style');
        s.textContent = '.test-a { & .test-b { color: red; } }';
        document.head.appendChild(s);
        var ruleCount = s.sheet ? s.sheet.cssRules.length : 0;
        document.head.removeChild(s);
        return ruleCount > 0;
      } catch (e) { return false; }
    });
    check('CSS', 'transforms-3d', 'CSS 3D Transforms (perspective)', cssSupports('perspective', '500px'));
    check('CSS', 'animations', 'CSS Keyframes & Transitions', cssSupports('animation', 'spin 1s infinite'));
    check('CSS', 'content-visibility', 'content-visibility: auto (DOM render boost)', cssSupports('content-visibility', 'auto'));

    /* =========================================================================
       3. HTML5 & WEB COMPONENTS
       ========================================================================= */
    check('HTML5', 'dialog', '<dialog> Modal Element', function () { return 'showModal' in HTMLDialogElement.prototype; });
    check('HTML5', 'details-summary', '<details> & <summary> Collapsible', function () {
      var d = document.createElement('details');
      return 'open' in d;
    });
    check('HTML5', 'template', '<template> Content Slot', function () {
      var t = document.createElement('template');
      return 'content' in t;
    });
    check('HTML5', 'custom-elements', 'Web Components: Custom Elements V1', hasProp(window, 'customElements'));
    check('HTML5', 'shadow-dom', 'Web Components: Shadow DOM V1', function () {
      return !!document.createElement('div').attachShadow;
    });
    check('HTML5', 'drag-and-drop', 'HTML5 Drag and Drop API', function () {
      var d = document.createElement('div');
      return 'draggable' in d && 'ondragstart' in d;
    });
    check('HTML5', 'canvas-2d', '<canvas> 2D Context', function () {
      var c = document.createElement('canvas');
      return !!(c.getContext && c.getContext('2d'));
    });
    check('HTML5', 'webgl-1', 'WebGL 1.0 (3D Graphics)', function () {
      var c = document.createElement('canvas');
      return !!(c.getContext && (c.getContext('webgl') || c.getContext('experimental-webgl')));
    });
    check('HTML5', 'webgl-2', 'WebGL 2.0 (Modern 3D Shader Model)', function () {
      var c = document.createElement('canvas');
      return !!(c.getContext && c.getContext('webgl2'));
    });
    check('HTML5', 'webgpu', 'WebGPU (Next-Gen Graphics API)', hasProp(navigator, 'gpu'));

    /* =========================================================================
       4. DOM & OBSERVERS
       ========================================================================= */
    check('DOM', 'mutation-observer', 'MutationObserver (DOM Changes)', hasProp(window, 'MutationObserver'));
    check('DOM', 'intersection-observer', 'IntersectionObserver (Viewport/Lazy)', hasProp(window, 'IntersectionObserver'));
    check('DOM', 'resize-observer', 'ResizeObserver (Element Dims)', hasProp(window, 'ResizeObserver'));
    check('DOM', 'performance-observer', 'PerformanceObserver', hasProp(window, 'PerformanceObserver'));
    check('DOM', 'fullscreen', 'Fullscreen API', function () {
      return !!(document.fullscreenEnabled || document.webkitFullscreenEnabled);
    });
    check('DOM', 'pointer-lock', 'Pointer Lock API (Mouse capture)', function () {
      return 'pointerLockElement' in document || 'webkitPointerLockElement' in document;
    });
    var isSec = (typeof window.isSecureContext !== 'undefined') ? window.isSecureContext : false;

    check('DOM', 'clipboard-async', 'Async Clipboard API (navigator.clipboard)', function () {
      return !!(navigator.clipboard && navigator.clipboard.writeText);
    }, (!isSec && !(navigator.clipboard && navigator.clipboard.writeText)) ? 'Dibatasi (Memerlukan Secure Context / HTTPS)' : undefined);

    /* =========================================================================
       5. NETWORKING & MESSAGING
       ========================================================================= */
    check('Network', 'fetch', 'Fetch API', hasProp(window, 'fetch'));
    check('Network', 'abort-controller', 'AbortController & AbortSignal', hasProp(window, 'AbortController'));
    check('Network', 'streams', 'ReadableStream & Streams API', hasProp(window, 'ReadableStream'));
    check('Network', 'websocket', 'WebSocket Full-Duplex Connection', hasProp(window, 'WebSocket'));
    check('Network', 'broadcast-channel', 'BroadcastChannel (Cross-tab/frame)', hasProp(window, 'BroadcastChannel'));
    check('Network', 'message-channel', 'MessageChannel (Direct 2-way port)', hasProp(window, 'MessageChannel'));
    check('Network', 'eventsource', 'Server-Sent Events (EventSource)', hasProp(window, 'EventSource'));
    check('Network', 'web-worker', 'Dedicated Web Workers', hasProp(window, 'Worker'));
    check('Network', 'shared-worker', 'SharedWorker', hasProp(window, 'SharedWorker'));
    check('Network', 'service-worker', 'ServiceWorker API', hasProp(navigator, 'serviceWorker'), (!isSec && !('serviceWorker' in navigator)) ? 'Dibatasi (Memerlukan Secure Context / HTTPS)' : undefined);

    /* =========================================================================
       6. SECURITY & CRYPTOGRAPHY
       ========================================================================= */
    check('Security', 'crypto-subtle', 'Web Cryptography (crypto.subtle)', function () {
      return !!(window.crypto && window.crypto.subtle);
    }, (!isSec && !(window.crypto && window.crypto.subtle)) ? 'Dibatasi (Memerlukan Secure Context / HTTPS)' : undefined);
    check('Security', 'crypto-random-values', 'crypto.getRandomValues()', function () {
      return !!(window.crypto && window.crypto.getRandomValues);
    });
    check('Security', 'crypto-random-uuid', 'crypto.randomUUID() (RFC 4122 V4)', function () {
      return !!(window.crypto && typeof window.crypto.randomUUID === 'function');
    }, (!isSec && !(window.crypto && typeof window.crypto.randomUUID === 'function')) ? 'Dibatasi (Memerlukan Secure Context / HTTPS)' : undefined);
    check('Security', 'secure-context', 'Secure Context Flag (isSecureContext)', function () {
      return window.isSecureContext === true;
    }, window.isSecureContext === true ? 'Secure Context Active' : 'Insecure Origin (http://app)');
    check('Security', 'cross-origin-isolation', 'Cross-Origin Isolation', function () {
      return window.crossOriginIsolated === true;
    });

    /* =========================================================================
       7. HARDWARE, MEDIA & SENSORS
       ========================================================================= */
    check('Sensors', 'get-user-media', 'MediaDevices.getUserMedia (Camera/Mic)', function () {
      return !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia);
    }, (!isSec && !(navigator.mediaDevices && navigator.mediaDevices.getUserMedia)) ? 'Dibatasi (Memerlukan Secure Context / HTTPS)' : undefined);
    check('Sensors', 'web-audio', 'Web Audio API (AudioContext)', function () {
      return typeof (window.AudioContext || window.webkitAudioContext) === 'function';
    });
    check('Sensors', 'speech-synthesis', 'Speech Synthesis (TTS Text-to-Speech)', hasProp(window, 'speechSynthesis'));
    check('Sensors', 'speech-recognition', 'Speech Recognition (STT Voice-to-Text)', function () {
      return !!(window.SpeechRecognition || window.webkitSpeechRecognition);
    });
    check('Sensors', 'gamepad', 'Gamepad API (Controllers)', hasProp(navigator, 'getGamepads'));
    check('Sensors', 'vibration', 'Vibration API', hasProp(navigator, 'vibrate'));
    check('Sensors', 'web-share', 'Web Share API (navigator.share)', hasProp(navigator, 'share'));
    check('Sensors', 'notification', 'Notification API (Desktop Alerts)', hasProp(window, 'Notification'));

    /* =========================================================================
       8. JAVASCRIPT MODERN LANJUTAN
       ========================================================================= */
    check('JavaScript', 'regexp-named-groups', 'RegExp Named Capture Groups', tryEval('var m = /(?<yr>\\d{4})/.exec("2026"); return !!m && m.groups.yr === "2026";'));
    check('JavaScript', 'regexp-lookbehind', 'RegExp Lookbehind (?<=)', tryEval('return /(?<=\\$)\\d+/.test("$42");'));
    check('JavaScript', 'regexp-dotall', 'RegExp dotAll (s flag)', tryEval('return /a.b/s.test("a\\nb");'));
    check('JavaScript', 'regexp-vflag', 'RegExp Unicode Sets (v flag)', tryEval('try { return new RegExp("[\\\\p{ASCII}]", "v") instanceof RegExp; } catch (e) { return false; }'));
    check('JavaScript', 'string-matchall', 'String.prototype.matchAll()', function () { return typeof String.prototype.matchAll === 'function'; });
    check('JavaScript', 'array-findlast', 'Array findLast / findLastIndex', function () { return typeof Array.prototype.findLast === 'function'; });
    check('JavaScript', 'object-groupby', 'Object.groupBy()', function () { return typeof Object.groupBy === 'function'; });
    check('JavaScript', 'promise-withresolvers', 'Promise.withResolvers()', function () { return typeof Promise.withResolvers === 'function'; });
    check('JavaScript', 'array-fromasync', 'Array.fromAsync()', function () { return typeof Array.fromAsync === 'function'; });
    check('JavaScript', 'async-generators', 'Async Generators (async function*)', tryEval('return typeof (async function*(){}) === "function";'));
    check('JavaScript', 'atomics-sab', 'Atomics & SharedArrayBuffer', function () { return typeof Atomics !== 'undefined' && typeof SharedArrayBuffer !== 'undefined'; });
    check('JavaScript', 'symbol-asynciterator', 'Symbol.asyncIterator', function () { return typeof Symbol !== 'undefined' && !!Symbol.asyncIterator; });
    check('JavaScript', 'intl-segmenter', 'Intl.Segmenter', function () { return typeof Intl !== 'undefined' && typeof Intl.Segmenter === 'function'; });
    check('JavaScript', 'intl-relativetime', 'Intl.RelativeTimeFormat', function () { return typeof Intl !== 'undefined' && typeof Intl.RelativeTimeFormat === 'function'; });
    check('JavaScript', 'temporal', 'Temporal date/time API', function () { return typeof Temporal !== 'undefined'; });
    check('JavaScript', 'error-cause', 'Error cause option', tryEval('try { throw new Error("x", { cause: 42 }); } catch (e) { return e.cause === 42; }'));

    /* =========================================================================
       9. CSS LANJUTAN (AT-RULE, BLEND, MASK, SCROLL)
       ========================================================================= */
    check('CSS', 'layer', '@layer Cascade Layers', cssAtRuleSupported('@layer respect_a, respect_b;'));
    check('CSS', 'scope', '@scope Scoped Styles', cssAtRuleSupported('@scope (.respect-scope) { .inner { color: red; } }'));
    check('CSS', 'property', '@property Custom Property', cssAtRuleSupported('@property --respect-x { syntax: "<color>"; inherits: false; initial-value: red; }'));
    check('CSS', 'sticky', 'position: sticky', cssSupports('position', 'sticky'));
    check('CSS', 'scroll-snap', 'Scroll Snap (scroll-snap-type)', cssSupports('scroll-snap-type', 'x mandatory'));
    check('CSS', 'mix-blend', 'mix-blend-mode', cssSupports('mix-blend-mode', 'multiply'));
    check('CSS', 'filter', 'CSS filter (blur)', cssSupports('filter', 'blur(2px)'));
    check('CSS', 'mask', 'CSS mask-image', cssSupports('mask-image', 'linear-gradient(#000,#fff)'));
    check('CSS', 'transform-3d', 'transform-style preserve-3d', cssSupports('transform-style', 'preserve-3d'));
    check('CSS', 'text-wrap', 'text-wrap: balance', cssSupports('text-wrap', 'balance'));
    check('CSS', 'light-dark', 'light-dark() color function', cssSupports('color', 'light-dark(#000,#fff)'));
    check('CSS', 'conic-gradient', 'conic-gradient()', cssSupports('background-image', 'conic-gradient(red, blue)'));
    check('CSS', 'view-transition', 'view-transition-name', cssSupports('view-transition-name', 'none'));
    check('CSS', 'focus-visible', ':focus-visible selector', function () {
      try { document.querySelectorAll(':focus-visible'); return true; } catch (e) { return false; }
    });

    /* =========================================================================
       10. HTML5 & WEB COMPONENTS LANJUTAN
       ========================================================================= */
    check('HTML5', 'popover', 'Popover API (popover attribute)', function () { return typeof HTMLElement !== 'undefined' && 'popover' in HTMLElement.prototype; });
    check('HTML5', 'canvas-toblob', 'canvas.toBlob()', function () { return typeof HTMLCanvasElement !== 'undefined' && typeof HTMLCanvasElement.prototype.toBlob === 'function'; });
    check('HTML5', 'img-srcset', '<img> srcset / sizes', function () { var i = document.createElement('img'); return ('srcset' in i) && ('sizes' in i); });
    check('HTML5', 'picture-element', '<picture> element', function () { return typeof HTMLPictureElement !== 'undefined'; });
    check('HTML5', 'video-audio-tags', '<video> & <audio> elements', function () { return !!document.createElement('video').canPlayType && !!document.createElement('audio').canPlayType; });
    check('HTML5', 'input-date-color', 'Input type date / color', function () { var i = document.createElement('input'); i.type = 'color'; var c = i.type === 'color'; i.type = 'date'; var d = i.type === 'date'; return c && d; });
    check('HTML5', 'datalist', '<datalist> element', function () { return 'options' in document.createElement('datalist'); });
    check('HTML5', 'meter-progress', '<meter> & <progress> elements', function () { return ('value' in document.createElement('meter')) && ('value' in document.createElement('progress')); });
    check('HTML5', 'slot-element', '<slot> element (Shadow DOM)', function () { return typeof HTMLSlotElement !== 'undefined'; });
    check('HTML5', 'output-element', '<output> element', function () { return 'value' in document.createElement('output'); });
    check('HTML5', 'contenteditable', 'contentEditable support', function () { return 'contentEditable' in document.createElement('div'); });
    check('HTML5', 'waapi', 'Web Animations (Element.animate)', function () { return typeof Element !== 'undefined' && typeof Element.prototype.animate === 'function'; });

    /* =========================================================================
       11. DOM & GEOMETRY LANJUTAN
       ========================================================================= */
    check('DOM', 'requestidlecallback', 'requestIdleCallback()', function () { return typeof window.requestIdleCallback === 'function'; });
    check('DOM', 'raf', 'requestAnimationFrame()', function () { return typeof window.requestAnimationFrame === 'function'; });
    check('DOM', 'getboundingclientrect', 'Element.getBoundingClientRect()', function () { return typeof Element !== 'undefined' && typeof Element.prototype.getBoundingClientRect === 'function'; });
    check('DOM', 'closest', 'Element.closest()', function () { return typeof Element !== 'undefined' && typeof Element.prototype.closest === 'function'; });
    check('DOM', 'classlist-dataset', 'classList & dataset', function () { var d = document.createElement('div'); return !!(d.classList && d.dataset); });
    check('DOM', 'custom-event', 'CustomEvent constructor', function () { return typeof window.CustomEvent === 'function'; });
    check('DOM', 'abort-timeout', 'AbortSignal.timeout()', function () { return typeof AbortSignal !== 'undefined' && typeof AbortSignal.timeout === 'function'; });
    check('DOM', 'matchmedia', 'window.matchMedia()', function () { return typeof window.matchMedia === 'function'; });
    check('DOM', 'getcomputedstyle', 'getComputedStyle()', function () { return typeof window.getComputedStyle === 'function'; });
    check('DOM', 'treewalker-range', 'TreeWalker, Range & Selection', function () { return typeof document.createTreeWalker === 'function' && typeof document.createRange === 'function' && !!window.getSelection; });
    check('DOM', 'pointer-events-api', 'PointerEvent API', function () { return typeof window.PointerEvent === 'function'; });
    check('DOM', 'touch-events-api', 'TouchEvent / touch points', function () { return ('ontouchstart' in window) || typeof window.TouchEvent === 'function'; });
    check('DOM', 'domrect', 'DOMRect geometry object', function () { return typeof DOMRect === 'function'; });

    /* =========================================================================
       12. NETWORK & PROTOKOL LANJUTAN
       ========================================================================= */
    check('Network', 'xmlhttprequest', 'XMLHttpRequest (classic AJAX)', hasProp(window, 'XMLHttpRequest'));
    check('Network', 'request-response', 'Fetch Request / Response / Headers', function () { return typeof Request === 'function' && typeof Response === 'function' && typeof Headers === 'function'; });
    check('Network', 'formdata', 'FormData', hasProp(window, 'FormData'));
    check('Network', 'url-searchparams', 'URL & URLSearchParams', function () { return typeof URL === 'function' && typeof URLSearchParams === 'function'; });
    check('Network', 'atob-btoa', 'atob() / btoa() Base64', function () { return typeof atob === 'function' && typeof btoa === 'function'; });
    check('Network', 'sendbeacon', 'navigator.sendBeacon()', hasProp(navigator, 'sendBeacon'));
    check('Network', 'webrtc', 'WebRTC (RTCPeerConnection)', function () { return typeof window.RTCPeerConnection === 'function'; });
    check('Network', 'urlpattern', 'URLPattern API', function () { return typeof URLPattern === 'function'; });
    check('Network', 'websocket-api', 'WebSocket constructor', hasProp(window, 'WebSocket'));

    /* =========================================================================
       13. STORAGE & PERSISTENSI
       ========================================================================= */
    check('Storage', 'localstorage', 'localStorage availability', function () { try { localStorage.setItem('__cap__', '1'); localStorage.removeItem('__cap__'); return true; } catch (e) { return false; } });
    check('Storage', 'sessionstorage', 'sessionStorage availability', function () { try { sessionStorage.setItem('__cap__', '1'); sessionStorage.removeItem('__cap__'); return true; } catch (e) { return false; } });
    check('Storage', 'indexeddb', 'IndexedDB availability', function () { return !!window.indexedDB; });
    check('Storage', 'cachestorage', 'CacheStorage (caches)', function () { return 'caches' in window; });
    check('Storage', 'opfs', 'Origin Private File System (OPFS)', function () { return !!(navigator.storage && typeof navigator.storage.getDirectory === 'function'); });
    check('Storage', 'cookies', 'Cookies (document.cookie)', function () { return typeof document.cookie === 'string'; });
    check('Storage', 'storage-estimate', 'navigator.storage.estimate()', function () { return !!(navigator.storage && typeof navigator.storage.estimate === 'function'); });
    check('Storage', 'storage-persist', 'navigator.storage.persist()', function () { return !!(navigator.storage && typeof navigator.storage.persist === 'function'); });

    /* =========================================================================
       14. SECURITY LANJUTAN
       ========================================================================= */
    check('Security', 'permissions-api', 'Permissions API (navigator.permissions)', hasProp(navigator, 'permissions'));
    check('Security', 'credentials', 'Credential Management API', hasProp(navigator, 'credentials'));
    check('Security', 'webauthn', 'WebAuthn (PublicKeyCredential)', function () { return typeof window.PublicKeyCredential === 'function'; });
    check('Security', 'iframe-sandbox', 'iframe sandbox attribute', function () { return 'sandbox' in document.createElement('iframe'); });

    /* =========================================================================
       15. SENSORS & HARDWARE LANJUTAN
       ========================================================================= */
    check('Sensors', 'geolocation', 'Geolocation API', hasProp(navigator, 'geolocation'));
    check('Sensors', 'device-orientation', 'DeviceOrientationEvent', function () { return typeof window.DeviceOrientationEvent === 'function' || ('ondeviceorientation' in window); });
    check('Sensors', 'device-motion', 'DeviceMotionEvent', function () { return typeof window.DeviceMotionEvent === 'function' || ('ondevicemotion' in window); });
    check('Sensors', 'bluetooth', 'Web Bluetooth API', hasProp(navigator, 'bluetooth'));
    check('Sensors', 'usb', 'WebUSB API', hasProp(navigator, 'usb'));
    check('Sensors', 'serial', 'Web Serial API', hasProp(navigator, 'serial'));
    check('Sensors', 'hid', 'WebHID API', hasProp(navigator, 'hid'));
    check('Sensors', 'clipboard-read', 'Clipboard readText()', function () { return !!(navigator.clipboard && typeof navigator.clipboard.readText === 'function'); });
    check('Sensors', 'idle-detection', 'Idle Detection API', function () { return typeof window.IdleDetector === 'function'; });
    check('Sensors', 'contact-picker', 'Contact Picker API', hasProp(navigator, 'contacts'));

    /* =========================================================================
       16. MEDIA & STREAMING READINESS (YouTube / Spotify / HLS)
       ========================================================================= */
    check('Media', 'media-source', 'Media Source Extensions (MSE)', function () { return typeof MediaSource === 'function'; });
    check('Media', 'source-buffer', 'SourceBuffer (MSE)', function () { return typeof SourceBuffer === 'function'; });
    check('Media', 'mse-mp4-h264', 'MSE isTypeSupported H.264/AAC', function () {
      return typeof MediaSource === 'function' && typeof MediaSource.isTypeSupported === 'function' &&
        MediaSource.isTypeSupported('video/mp4; codecs="avc1.42E01E,mp4a.40.2"');
    });
    check('Media', 'mse-webm-vp9', 'MSE isTypeSupported WebM VP9/Opus', function () {
      return typeof MediaSource === 'function' && typeof MediaSource.isTypeSupported === 'function' &&
        MediaSource.isTypeSupported('video/webm; codecs="vp9,opus"');
    });
    check('Media', 'native-hls', 'Native HLS (application/vnd.apple.mpegurl)', function () {
      var v = document.createElement('video');
      return v.canPlayType('application/vnd.apple.mpegurl') !== '';
    });
    check('Media', 'eme-drm', 'Encrypted Media Extensions (Widevine/DRM)', function () { return typeof navigator.requestMediaKeySystemAccess === 'function'; });
    check('Media', 'webcodecs-video', 'WebCodecs VideoDecoder', function () { return typeof VideoDecoder === 'function'; });
    check('Media', 'media-recorder', 'MediaRecorder API', function () { return typeof MediaRecorder === 'function'; });
    check('Media', 'capture-stream', 'canvas.captureStream()', function () { return typeof HTMLCanvasElement !== 'undefined' && typeof HTMLCanvasElement.prototype.captureStream === 'function'; });

    // Compute Summary Stats
    var total = tests.length;
    var passed = 0;
    var byCat = {};

    for (var i = 0; i < tests.length; i++) {
      var t = tests[i];
      if (t.pass) passed++;
      if (!byCat[t.category]) {
        byCat[t.category] = { total: 0, passed: 0 };
      }
      byCat[t.category].total++;
      if (t.pass) byCat[t.category].passed++;
    }

    return {
      tests: tests,
      total: total,
      passed: passed,
      failed: total - passed,
      percentage: ((passed / total) * 100).toFixed(1),
      byCategory: byCat
    };
  }

  return {
    runAllTests: runAllTests
  };
})();

if (typeof window !== 'undefined') {
  window.CapabilityEngine = CapabilityEngine;
}
