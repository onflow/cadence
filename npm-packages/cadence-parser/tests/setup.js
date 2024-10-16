Object.defineProperty(global, 'performance', {
  writable: true,
});

Object.defineProperty(global, 'fetch', {
  writable: true,
});

global.performance = require("perf_hooks").performance;
global.fetch = require("node-fetch");
