/**
 * Respect Browser Stress Testing Suite - Window & Global JS API Dump
 * Mengaudit seluruh properti, fungsi, class/constructor, prototype chain,
 * descriptor, dan nilai evaluasi nyata dari objek global (window, origin, navigator, dll).
 */

var ApiDumpEngine = (function () {
  'use strict';

  var cachedDump = null;

  function safeSerializeVal(val, depth) {
    depth = depth || 0;
    if (depth > 2) return '[Deep Object]';
    if (val === null) return 'null';
    if (val === undefined) return 'undefined';
    var t = typeof val;
    if (t === 'string') return '"' + val.substring(0, 100) + (val.length > 100 ? '...' : '') + '"';
    if (t === 'number' || t === 'boolean') return String(val);
    if (t === 'symbol') return val.toString();
    if (t === 'bigint') return val.toString() + 'n';
    if (t === 'function') {
      var fnName = val.name ? (' ' + val.name) : '';
      return 'function' + fnName + '(' + (val.length ? ('...' + val.length + ' args') : '') + ')';
    }
    if (Array.isArray(val)) {
      return '[Array(' + val.length + ')]';
    }
    try {
      if (val.constructor && val.constructor.name) {
        return '[' + val.constructor.name + ']';
      }
    } catch (e) {}
    return '[Object]';
  }

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
        var sampleVal = '';
        var descriptorInfo = '';
        var subKeys = [];

        try {
          var desc = Object.getOwnPropertyDescriptor(target.obj, key);
          if (desc) {
            descriptorInfo = (desc.writable ? 'W' : '-') +
                             (desc.enumerable ? 'E' : '-') +
                             (desc.configurable ? 'C' : '-') +
                             (desc.get ? 'G' : '-') +
                             (desc.set ? 'S' : '-');
          }
        } catch (e) {}

        try {
          val = window[key];
          valType = typeof val;
          sampleVal = safeSerializeVal(val, 0);

          // If object has sub-properties (e.g. location, navigator, screen), probe top-level keys
          if (valType === 'object' && val !== null) {
            try {
              var ownP = Object.getOwnPropertyNames(val);
              subKeys = ownP.slice(0, 15);
            } catch (e) {}
          }
        } catch (err) {
          valType = 'restricted';
          sampleVal = '[Restricted / Security Access Denied]';
        }

        if (valType === 'function') {
          isCallable = true;
          // Heuristic for constructor / class: starts with uppercase or has prototype.constructor === val
          try {
            if (/^[A-Z]/.test(key) || (val && val.prototype && val.prototype.constructor === val)) {
              isConstructor = true;
              counts.constructors++;
            } else {
              counts.functions++;
            }
          } catch (e) {
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
          descriptor: descriptorInfo,
          sampleValue: sampleVal,
          subProperties: subKeys,
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

  function getNamespaceDetails(name) {
    try {
      var obj = window[name];
      if (!obj || typeof obj !== 'object') return null;

      var result = {};
      var keys = Object.getOwnPropertyNames(obj);
      for (var i = 0; i < keys.length; i++) {
        var k = keys[i];
        try {
          result[k] = safeSerializeVal(obj[k], 1);
        } catch (e) {
          result[k] = '[Access Error]';
        }
      }
      return result;
    } catch (e) {
      return null;
    }
  }

  function filterApis(query, category) {
    var data = dumpGlobalApis();
    var q = (query || '').toLowerCase().trim();

    return data.apis.filter(function (item) {
      if (q) {
        var matchName = item.name.toLowerCase().indexOf(q) !== -1;
        var matchVal = (item.sampleValue || '').toLowerCase().indexOf(q) !== -1;
        if (!matchName && !matchVal) return false;
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
    getNamespaceDetails: getNamespaceDetails,
    filterApis: filterApis
  };
})();
