// Respect Stress Testing - ES Module Probe
// Dipakai untuk menguji dukungan <script type="module"> dan dynamic import()
// pada Virtual Host (http://app/) maupun Local Server (http://127.0.0.1:port/).

if (typeof window !== 'undefined') {
  window.__respectModuleLoaded = true;
  window.__respectModuleInfo = 'module-probe.mjs loaded';
}

export var MODULE_VERSION = 1;

export function probe() {
  return 'module-ok';
}

export function add(a, b) {
  return a + b;
}
