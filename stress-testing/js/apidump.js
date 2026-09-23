/**
 * Respect Browser Stress Testing Suite - Window & Global JS API Dump
 * Mengaudit seluruh properti, fungsi, class/constructor, prototype chain,
 * dan injeksi khusus engine Respect yang terpasang di global context.
 */

var ApiDumpEngine = (function () {
  'use strict';

  var cachedDump = null;

  function dumpGlobalApis() {
    if (cachedDump) return cachedDump;

    var seen = {};
    var apiList = [];
    var counts = {
      total: 0,
      constructors: 0,
      functions: 0,
      objects: 0,
      values: 0,
      respectInjected: 0
    };

    var targets = [
      { name: 'window', obj: window },
      { name: 'Window.prototype', obj: window.Window ? Window.prototype : null },
      { name: 'EventTarget.prototype', obj: window.EventTarget ? EventTarget.prototype : null }
    ];

    for (var t = 0; t < targets.length; t++) {
      var target = targets[t];
      if (!target.obj) continue;

      var props = [];
      try {
        props = Object.getOwnPropertyNames(target.obj);
      } catch (e) {
        continue;
      }

      for (var i = 0; i < props.length; i++) {
        var key = props[i];
        if (seen[key]) continue;
        seen[key] = true;

        var val = undefined;
        var valType = 'undefined';
        var isCallable = false;
        var isConstructor = false;
        var isRespect = (key === 'mbQuery' || key === 'ipc' || key === 'chrome' || key.indexOf('respect') !== -1);

        try {
          val = window[key];
          valType = typeof val;
        } catch (err) {
          valType = 'restricted-getter';
        }

        if (valType === 'function') {
          isCallable = true;
          // Heuristic for constructor / class: starts with uppercase or has prototype.constructor === val
          if (/^[A-Z]/.test(key) || (val.prototype && val.prototype.constructor === val)) {
            isConstructor = true;
            counts.constructors++;
          } else {
            counts.functions++;
          }
        } else if (valType === 'object' && val !== null) {
          counts.objects++;
        } else {
          counts.values++;
        }

        if (isRespect) {
          counts.respectInjected++;
        }

        counts.total++;

        apiList.push({
          name: key,
          origin: target.name,
          type: valType,
          isConstructor: isConstructor,
          isCallable: isCallable,
          isRespect: isRespect,
          summary: isConstructor ? 'Class / Constructor' :
                   isCallable ? 'Callable Function' :
                   valType === 'object' ? 'Object Namespace' : ('Primitive (' + valType + ')')
        });
      }
    }

    // Sort alphabetically
    apiList.sort(function (a, b) {
      return a.name.localeCompare(b.name);
    });

    cachedDump = {
      counts: counts,
      apis: apiList
    };

    return cachedDump;
  }

  function filterApis(query, category) {
    var data = dumpGlobalApis();
    var q = (query || '').toLowerCase().trim();

    return data.apis.filter(function (item) {
      if (q && item.name.toLowerCase().indexOf(q) === -1) {
        return false;
      }
      if (!category || category === 'all') return true;
      if (category === 'constructor') return item.isConstructor;
      if (category === 'function') return item.isCallable && !item.isConstructor;
      if (category === 'object') return item.type === 'object';
      if (category === 'value') return !item.isCallable && item.type !== 'object';
      if (category === 'respect') return item.isRespect;
      return true;
    });
  }

  return {
    dumpGlobalApis: dumpGlobalApis,
    filterApis: filterApis
  };
})();
