/**
 * Respect Browser Stress Testing Suite - Stress Benchmarks Rig
 * Menguji ketahanan beban tinggi: DOM Thrashing, Canvas Particle Physics 20K FPS,
 * Web Worker multithreading compute, Memory Allocation chunking, dan Storage I/O.
 */

var StressBenchmarkEngine = (function () {
  'use strict';

  var particleCanvas = null;
  var particleCtx = null;
  var particleAnimId = null;
  var particles = [];
  var particleCount = 5000;
  var isParticleRunning = false;

  var fpsLastTime = performance.now();
  var fpsFrames = 0;
  var currentFps = 60;
  var minFps = 60;
  var maxFps = 0;

  var workerInstance = null;

  // =========================================================================
  // 1. DOM THRASHING & MUTATION STRESS TEST
  // =========================================================================
  function runDomStress(nodeCount, containerEl, callback) {
    if (!containerEl) return;
    nodeCount = nodeCount || 3000;

    var startInsert = performance.now();
    var fragment = document.createDocumentFragment();

    for (var i = 0; i < nodeCount; i++) {
      var item = document.createElement('div');
      item.className = 'dom-stress-node';
      item.style.cssText = 'display:inline-block; width:6px; height:6px; margin:1px; background:hsl(' + (i % 360) + ',70%,60%); border-radius:2px;';
      fragment.appendChild(item);
    }
    containerEl.innerHTML = '';
    containerEl.appendChild(fragment);
    var insertElapsed = performance.now() - startInsert;

    // Trigger reflow & style update
    var startReflow = performance.now();
    var children = containerEl.children;
    for (var j = 0; j < children.length; j += 10) {
      children[j].style.transform = 'scale(1.2)';
      var forceReflow = children[j].offsetWidth; // Force layout calculation
    }
    var reflowElapsed = performance.now() - startReflow;

    // Cleanup latency
    var startCleanup = performance.now();
    containerEl.innerHTML = '';
    var cleanupElapsed = performance.now() - startCleanup;

    var result = {
      nodeCount: nodeCount,
      insertTimeMs: insertElapsed.toFixed(2),
      reflowTimeMs: reflowElapsed.toFixed(2),
      cleanupTimeMs: cleanupElapsed.toFixed(2),
      totalDurationMs: (insertElapsed + reflowElapsed + cleanupElapsed).toFixed(2),
      opsPerSec: Math.round((nodeCount / (insertElapsed + reflowElapsed)) * 1000)
    };

    if (callback) callback(result);
    return result;
  }

  // =========================================================================
  // 2. 2D CANVAS PARTICLE PHYSICS STRESS TEST (Up to 20,000 Particles)
  // =========================================================================
  function initParticleCanvas(canvasEl, count) {
    particleCanvas = canvasEl;
    particleCtx = canvasEl.getContext('2d');
    particleCount = count || 5000;
    particles = [];

    var w = canvasEl.width;
    var h = canvasEl.height;

    for (var i = 0; i < particleCount; i++) {
      particles.push({
        x: Math.random() * w,
        y: Math.random() * h,
        vx: (Math.random() - 0.5) * 4,
        vy: (Math.random() - 0.5) * 4,
        size: Math.random() * 2.5 + 1,
        hue: Math.floor(Math.random() * 360)
      });
    }
  }

  function startParticleSimulation(onFpsUpdate) {
    if (isParticleRunning) return;
    isParticleRunning = true;
    fpsLastTime = performance.now();
    fpsFrames = 0;
    minFps = 60;
    maxFps = 0;

    function loop() {
      if (!isParticleRunning) return;
      particleAnimId = requestAnimationFrame(loop);

      var w = particleCanvas.width;
      var h = particleCanvas.height;

      // Dark fade trail
      particleCtx.fillStyle = 'rgba(10, 14, 23, 0.25)';
      particleCtx.fillRect(0, 0, w, h);

      particleCtx.globalCompositeOperation = 'lighter';

      for (var i = 0; i < particles.length; i++) {
        var p = particles[i];
        p.x += p.vx;
        p.y += p.vy;

        if (p.x < 0 || p.x > w) p.vx = -p.vx;
        if (p.y < 0 || p.y > h) p.vy = -p.vy;

        particleCtx.fillStyle = 'hsl(' + p.hue + ', 90%, 60%)';
        particleCtx.beginPath();
        particleCtx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
        particleCtx.fill();
      }

      particleCtx.globalCompositeOperation = 'source-over';

      // Measure FPS
      fpsFrames++;
      var now = performance.now();
      var delta = now - fpsLastTime;

      if (delta >= 1000) {
        currentFps = Math.round((fpsFrames * 1000) / delta);
        if (currentFps < minFps) minFps = currentFps;
        if (currentFps > maxFps) maxFps = currentFps;
        fpsFrames = 0;
        fpsLastTime = now;

        if (onFpsUpdate) {
          onFpsUpdate({
            fps: currentFps,
            minFps: minFps,
            maxFps: maxFps,
            particles: particles.length
          });
        }
      }
    }

    loop();
  }

  function setParticleCount(newCount) {
    initParticleCanvas(particleCanvas, newCount);
  }

  function stopParticleSimulation() {
    isParticleRunning = false;
    if (particleAnimId) {
      cancelAnimationFrame(particleAnimId);
      particleAnimId = null;
    }
  }

  // =========================================================================
  // 3. COMPUTATION BENCHMARK (MAIN THREAD VS WEB WORKER)
  // =========================================================================
  function runComputeBenchmark(limit, useWorker, callback) {
    limit = limit || 1500000;
    var start = performance.now();

    if (useWorker) {
      try {
        if (!workerInstance) {
          workerInstance = new Worker('worker.js');
        }

        var taskId = 'task_' + Math.random().toString(36).substr(2, 9);

        var onMsg = function (e) {
          var data = e.data;
          if (data && data.taskId === taskId) {
            workerInstance.removeEventListener('message', onMsg);
            if (callback) {
              callback({
                mode: 'Dedicated Web Worker (Background Thread)',
                limit: limit,
                primesFound: data.primesFound,
                elapsedMs: data.elapsedMs.toFixed(2),
                totalLatencyMs: (performance.now() - start).toFixed(2),
                workerSupported: true
              });
            }
          }
        };

        workerInstance.addEventListener('message', onMsg);
        workerInstance.postMessage({
          cmd: 'prime_benchmark',
          taskId: taskId,
          limit: limit
        });
      } catch (err) {
        if (callback) {
          callback({
            mode: 'Web Worker Failed (Fallback)',
            error: err.message,
            workerSupported: false
          });
        }
      }
    } else {
      // Main Thread execution (Sieve of Eratosthenes)
      var count = 0;
      var sieve = new Uint8Array(limit + 1);
      for (var p = 2; p * p <= limit; p++) {
        if (sieve[p] === 0) {
          for (var i = p * p; i <= limit; i += p) {
            sieve[i] = 1;
          }
        }
      }
      for (var n = 2; n <= limit; n++) {
        if (sieve[n] === 0) count++;
      }
      var elapsed = performance.now() - start;

      if (callback) {
        callback({
          mode: 'Main Thread (Synchronous)',
          limit: limit,
          primesFound: count,
          elapsedMs: elapsed.toFixed(2),
          workerSupported: true
        });
      }
    }
  }

  // =========================================================================
  // 4. MEMORY ALLOCATION & GC STRESS TEST
  // =========================================================================
  function runMemoryStress(targetMb, callback) {
    targetMb = targetMb || 150;
    var chunkSizeMb = 25;
    var chunksCount = Math.floor(targetMb / chunkSizeMb);
    var allocatedBuffers = [];

    var start = performance.now();
    var successMb = 0;

    try {
      for (var i = 0; i < chunksCount; i++) {
        var bytes = chunkSizeMb * 1024 * 1024;
        var buf = new Uint8Array(bytes);
        // Write pattern to force physical OS page faulting
        buf[0] = 0xAA;
        buf[Math.floor(bytes / 2)] = 0x55;
        buf[bytes - 1] = 0xFF;
        allocatedBuffers.push(buf);
        successMb += chunkSizeMb;
      }
    } catch (e) {
      // Out of memory or allocation failure
    }

    var allocTime = performance.now() - start;

    // Dereference to allow GC
    var startGc = performance.now();
    allocatedBuffers = null;
    var derefTime = performance.now() - startGc;

    var result = {
      requestedMb: targetMb,
      allocatedMb: successMb,
      allocTimeMs: allocTime.toFixed(2),
      derefTimeMs: derefTime.toFixed(2),
      throughputMbSec: Math.round((successMb / (allocTime / 1000)))
    };

    if (callback) callback(result);
    return result;
  }

  // =========================================================================
  // 5. STORAGE I/O STRESS TEST
  // =========================================================================
  function runStorageIoStress(recordCount, callback) {
    recordCount = recordCount || 500;
    var start = performance.now();
    var prefix = '__stress_';

    // 1. Bulk Write
    for (var i = 0; i < recordCount; i++) {
      try {
        localStorage.setItem(prefix + i, 'payload_data_hash_stress_testing_respect_desktop_' + i);
      } catch (e) {
        break;
      }
    }
    var writeTime = performance.now() - start;

    // 2. Bulk Read
    var startRead = performance.now();
    var readCount = 0;
    for (var j = 0; j < recordCount; j++) {
      var val = localStorage.getItem(prefix + j);
      if (val) readCount++;
    }
    var readTime = performance.now() - startRead;

    // 3. Bulk Delete
    var startDel = performance.now();
    for (var k = 0; k < recordCount; k++) {
      localStorage.removeItem(prefix + k);
    }
    var delTime = performance.now() - startDel;

    var result = {
      recordCount: recordCount,
      writeTimeMs: writeTime.toFixed(2),
      readTimeMs: readTime.toFixed(2),
      deleteTimeMs: delTime.toFixed(2),
      totalMs: (performance.now() - start).toFixed(2),
      writeOpsPerSec: Math.round((recordCount / (writeTime / 1000))),
      readOpsPerSec: Math.round((readCount / (readTime / 1000)))
    };

    if (callback) callback(result);
    return result;
  }

  return {
    runDomStress: runDomStress,
    initParticleCanvas: initParticleCanvas,
    startParticleSimulation: startParticleSimulation,
    setParticleCount: setParticleCount,
    stopParticleSimulation: stopParticleSimulation,
    runComputeBenchmark: runComputeBenchmark,
    runMemoryStress: runMemoryStress,
    runStorageIoStress: runStorageIoStress
  };
})();
