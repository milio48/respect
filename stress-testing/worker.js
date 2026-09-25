/**
 * Respect Browser Stress Testing Suite - Dedicated Web Worker
 * Menguji kinerja komputasi multithread, transfer ArrayBuffer, dan isolasi background task.
 */

self.onmessage = function (e) {
  var data = e.data || {};
  var cmd = data.cmd;
  var taskId = data.taskId;
  var startTime = performance.now();

  try {
    switch (cmd) {
      case 'ping':
        self.postMessage({
          cmd: 'pong',
          taskId: taskId,
          elapsedMs: performance.now() - startTime,
          timestamp: Date.now()
        });
        break;

      case 'prime_benchmark':
        var maxLimit = data.limit || 1000000;
        var count = 0;
        // Sieve of Eratosthenes
        var sieve = new Uint8Array(maxLimit + 1);
        for (var p = 2; p * p <= maxLimit; p++) {
          if (sieve[p] === 0) {
            for (var i = p * p; i <= maxLimit; i += p) {
              sieve[i] = 1;
            }
          }
        }
        for (var n = 2; n <= maxLimit; n++) {
          if (sieve[n] === 0) count++;
        }
        var elapsed = performance.now() - startTime;
        self.postMessage({
          cmd: 'prime_result',
          taskId: taskId,
          limit: maxLimit,
          primesFound: count,
          elapsedMs: elapsed
        });
        break;

      case 'matrix_benchmark':
        var size = data.size || 250;
        var A = new Float64Array(size * size);
        var B = new Float64Array(size * size);
        var C = new Float64Array(size * size);

        // Fill with arbitrary numbers
        for (var m = 0; m < size * size; m++) {
          A[m] = (m % 10) * 1.5;
          B[m] = ((m + 3) % 10) * 2.1;
        }

        // Standard matrix multiplication
        for (var r = 0; r < size; r++) {
          for (var c = 0; c < size; c++) {
            var sum = 0.0;
            for (var k = 0; k < size; k++) {
              sum += A[r * size + k] * B[k * size + c];
            }
            C[r * size + c] = sum;
          }
        }
        var matElapsed = performance.now() - startTime;
        self.postMessage({
          cmd: 'matrix_result',
          taskId: taskId,
          matrixSize: size + 'x' + size,
          checksum: C[0] + C[C.length - 1],
          elapsedMs: matElapsed
        });
        break;

      case 'fetch_probe':
        var url = data.url || 'data.json';
        if (typeof fetch !== 'function') {
          self.postMessage({ cmd: 'fetch_result', taskId: taskId, ok: false, error: 'fetch() tidak tersedia di worker', elapsedMs: performance.now() - startTime });
          break;
        }
        fetch(url, { cache: 'no-store' }).then(function (resp) {
          return resp.text().then(function (text) {
            self.postMessage({
              cmd: 'fetch_result',
              taskId: taskId,
              ok: resp.ok && text.length > 0,
              status: resp.status,
              length: text.length,
              elapsedMs: performance.now() - startTime
            });
          });
        }).catch(function (err) {
          self.postMessage({
            cmd: 'fetch_result',
            taskId: taskId,
            ok: false,
            error: (err && err.message) ? err.message : String(err),
            elapsedMs: performance.now() - startTime
          });
        });
        break;

      case 'transfer_test':
        var bufferSize = data.sizeBytes || (10 * 1024 * 1024); // 10MB default
        var buffer = new ArrayBuffer(bufferSize);
        var view = new Uint8Array(buffer);
        view[0] = 0xAA;
        view[view.length - 1] = 0x55;

        // Transfer ownership back to main thread (Zero-copy)
        self.postMessage({
          cmd: 'transfer_result',
          taskId: taskId,
          sizeBytes: bufferSize,
          elapsedMs: performance.now() - startTime,
          buffer: buffer
        }, [buffer]);
        break;

      default:
        self.postMessage({
          cmd: 'error',
          taskId: taskId,
          message: 'Unknown worker command: ' + cmd
        });
    }
  } catch (err) {
    self.postMessage({
      cmd: 'error',
      taskId: taskId,
      message: err.message || String(err)
    });
  }
};
