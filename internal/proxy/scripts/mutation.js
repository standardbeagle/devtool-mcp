// DOM mutation tracking module for DevTool
// Tracks added, removed, and modified elements with visual highlighting

(function() {
  'use strict';

  var utils = window.__devtool_utils;
  var core = window.__devtool_core;

  var config = {
    maxHistorySize: 200,
    highlightDuration: 2000,
    highlightAddedColor: 'rgba(0, 255, 0, 0.2)',
    highlightRemovedColor: 'rgba(255, 0, 0, 0.2)',
    highlightModifiedColor: 'rgba(255, 255, 0, 0.2)',
    trackAttributes: true,
    trackCharacterData: false,
    ignoreSelectors: ['.__devtool', '#__devtool-overlays', 'script', 'style', 'link'],
    sendBatchSize: 10,
    sendInterval: 1000
  };

  var mutations = [];
  var pendingBatch = [];
  var highlightElements = new Map();
  var observer = null;

  // Check if element should be ignored
  function shouldIgnore(node) {
    if (!(node instanceof HTMLElement)) return true;

    for (var i = 0; i < config.ignoreSelectors.length; i++) {
      try {
        if (node.matches && node.matches(config.ignoreSelectors[i])) return true;
        if (node.closest && node.closest(config.ignoreSelectors[i])) return true;
      } catch (e) {}
    }
    return false;
  }

  // Highlight element temporarily
  function highlightMutation(element, color) {
    if (!element || !(element instanceof HTMLElement)) return;
    if (shouldIgnore(element)) return;

    var originalBg = element.style.backgroundColor;
    var originalOutline = element.style.outline;
    var originalTransition = element.style.transition;

    element.style.transition = 'background-color 0.3s, outline 0.3s';
    element.style.backgroundColor = color;
    element.style.outline = '2px solid ' + color.replace('0.2', '0.8');

    var id = 'mutation-' + Date.now() + '-' + Math.random();
    highlightElements.set(id, {
      element: element,
      originalBg: originalBg,
      originalOutline: originalOutline,
      originalTransition: originalTransition
    });

    setTimeout(function() {
      var info = highlightElements.get(id);
      if (info && info.element) {
        info.element.style.backgroundColor = info.originalBg;
        info.element.style.outline = info.originalOutline;
        setTimeout(function() {
          if (info.element) {
            info.element.style.transition = info.originalTransition;
          }
        }, 300);
        highlightElements.delete(id);
      }
    }, config.highlightDuration);
  }

  // Mutation observer callback
  function handleMutations(mutationsList) {
    for (var i = 0; i < mutationsList.length; i++) {
      var mutation = mutationsList[i];

      if (mutation.type === 'childList') {
        // Added nodes
        mutation.addedNodes.forEach(function(node) {
          if (shouldIgnore(node)) return;

          var record = {
            mutation_type: 'added',
            target: {
              selector: utils.generateSelector(mutation.target),
              tag: mutation.target.tagName ? mutation.target.tagName.toLowerCase() : 'unknown',
              id: mutation.target.id || undefined
            },
            added: [{
              selector: node.nodeType === 1 ? utils.generateSelector(node) : null,
              tag: node.nodeName.toLowerCase(),
              id: node.id || undefined,
              html: node.outerHTML ? node.outerHTML.substring(0, 500) : undefined
            }],
            timestamp: Date.now()
          };

          mutations.push(record);
          pendingBatch.push(record);

          if (node instanceof HTMLElement) {
            highlightMutation(node, config.highlightAddedColor);
          }
        });

        // Removed nodes
        mutation.removedNodes.forEach(function(node) {
          if (shouldIgnore(node)) return;

          var record = {
            mutation_type: 'removed',
            target: {
              selector: utils.generateSelector(mutation.target),
              tag: mutation.target.tagName ? mutation.target.tagName.toLowerCase() : 'unknown',
              id: mutation.target.id || undefined
            },
            removed: [{
              tag: node.nodeName.toLowerCase(),
              id: node.id || undefined,
              html: node.outerHTML ? node.outerHTML.substring(0, 200) : undefined
            }],
            timestamp: Date.now()
          };

          mutations.push(record);
          pendingBatch.push(record);
        });
      }

      if (mutation.type === 'attributes' && config.trackAttributes) {
        var target = mutation.target;
        if (shouldIgnore(target)) continue;

        var record = {
          mutation_type: 'attributes',
          target: {
            selector: utils.generateSelector(target),
            tag: target.tagName ? target.tagName.toLowerCase() : 'unknown',
            id: target.id || undefined
          },
          attribute: {
            name: mutation.attributeName,
            old_value: mutation.oldValue,
            new_value: target.getAttribute(mutation.attributeName)
          },
          timestamp: Date.now()
        };

        mutations.push(record);
        pendingBatch.push(record);
        highlightMutation(target, config.highlightModifiedColor);
      }
    }

    // Trim history
    if (mutations.length > config.maxHistorySize) {
      mutations = mutations.slice(-config.maxHistorySize);
    }
  }

  // Send batch to server
  function sendBatch() {
    if (pendingBatch.length === 0) return;

    var batch = pendingBatch.splice(0, config.sendBatchSize);
    core.send('mutations', { events: batch });
  }

  // Start observer
  function startObserver() {
    if (observer) return;

    observer = new MutationObserver(handleMutations);

    observer.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: config.trackAttributes,
      attributeOldValue: true,
      characterData: config.trackCharacterData,
      characterDataOldValue: true
    });
  }

  // Stop observer
  function stopObserver() {
    if (observer) {
      observer.disconnect();
      observer = null;
    }
  }

  // Start batch sender
  setInterval(sendBatch, config.sendInterval);

  // Initialize when DOM is ready
  if (document.body) {
    startObserver();
  } else {
    document.addEventListener('DOMContentLoaded', startObserver);
  }

  // Export mutations API
  window.__devtool_mutations = {
    getHistory: function(count) {
      count = count || 50;
      var start = Math.max(0, mutations.length - count);
      return mutations.slice(start);
    },

    getAdded: function(since) {
      since = since || 0;
      return mutations.filter(function(m) {
        return m.mutation_type === 'added' && m.timestamp > since;
      });
    },

    getRemoved: function(since) {
      since = since || 0;
      return mutations.filter(function(m) {
        return m.mutation_type === 'removed' && m.timestamp > since;
      });
    },

    getModified: function(since) {
      since = since || 0;
      return mutations.filter(function(m) {
        return m.mutation_type === 'attributes' && m.timestamp > since;
      });
    },

    highlightRecent: function(duration) {
      duration = duration || 5000;
      var since = Date.now() - duration;

      mutations.forEach(function(m) {
        if (m.timestamp > since && m.mutation_type === 'added' && m.added) {
          m.added.forEach(function(node) {
            if (node.selector) {
              var el = document.querySelector(node.selector);
              if (el) highlightMutation(el, config.highlightAddedColor);
            }
          });
        }
      });
    },

    clear: function() {
      mutations = [];
    },

    pause: function() {
      stopObserver();
    },

    resume: function() {
      startObserver();
    },

    config: config
  };
})();
