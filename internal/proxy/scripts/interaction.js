// User interaction tracking module for DevTool
// Tracks mouse, keyboard, and form interactions

(function() {
  'use strict';

  var utils = window.__devtool_utils;
  var core = window.__devtool_core;

  // Configuration
  var config = {
    maxHistorySize: 500,       // Max interactions in local buffer
    debounceScroll: 100,       // ms to debounce scroll events
    debounceInput: 300,        // ms to debounce input events
    truncateText: 100,         // Max chars for text content
    mouseMoveWindow: 60000,    // Sample mousemove for 1 minute around interactions
    mouseMoveInterval: 100,    // Sample mousemove every 100ms (interaction time)
    sendBatchSize: 10,         // Send in batches
    sendInterval: 1000         // ms between batch sends
  };

  // Mousemove sampling state
  var mouseMoveBuffer = [];           // Recent mousemove samples
  var lastInteractionTime = 0;        // Wall time of last meaningful interaction
  var lastMouseSampleTime = 0;        // Interaction-time of last mousemove sample
  var interactionTimeBase = 0;        // Base for interaction-time calculation

  // Local interaction buffer (circular)
  var interactions = [];
  var interactionIndex = 0;

  // Pending batch for server
  var pendingBatch = [];

  // Track last scroll/input to debounce
  var lastScroll = 0;
  var inputDebounce = {};

  // Generate target info from element
  function getTargetInfo(el) {
    if (!el || !(el instanceof HTMLElement)) return null;

    var text = (el.innerText || '').substring(0, config.truncateText);
    var attrs = {};

    // Only include relevant attributes
    ['href', 'src', 'type', 'name', 'placeholder', 'role', 'aria-label'].forEach(function(attr) {
      if (el.hasAttribute(attr)) {
        attrs[attr] = el.getAttribute(attr);
      }
    });

    return {
      selector: utils.generateSelector(el),
      tag: el.tagName.toLowerCase(),
      id: el.id || undefined,
      classes: Array.prototype.slice.call(el.classList).slice(0, 5),
      text: text || undefined,
      attributes: Object.keys(attrs).length > 0 ? attrs : undefined
    };
  }

  // Record interaction
  function recordInteraction(eventType, event, extra) {
    var target = event.target || event.srcElement;
    var targetInfo = getTargetInfo(target);
    if (!targetInfo) return;

    var interaction = {
      event_type: eventType,
      target: targetInfo,
      timestamp: Date.now()
    };

    // Add position for mouse events
    if (event.clientX !== undefined) {
      interaction.position = {
        client_x: event.clientX,
        client_y: event.clientY,
        page_x: event.pageX,
        page_y: event.pageY
      };
    }

    // Add key info for keyboard events
    if (event.key !== undefined) {
      interaction.key = {
        key: event.key,
        code: event.code,
        ctrl: event.ctrlKey || undefined,
        alt: event.altKey || undefined,
        shift: event.shiftKey || undefined,
        meta: event.metaKey || undefined
      };
    }

    // Add extra data
    if (extra) {
      for (var key in extra) {
        interaction[key] = extra[key];
      }
    }

    // Store locally
    if (interactions.length < config.maxHistorySize) {
      interactions.push(interaction);
    } else {
      interactions[interactionIndex % config.maxHistorySize] = interaction;
    }
    interactionIndex++;

    // Queue for server
    pendingBatch.push(interaction);
  }

  // Reset interaction time base on meaningful interactions
  function resetInteractionTime() {
    var now = Date.now();
    lastInteractionTime = now;
    interactionTimeBase = now;
    lastMouseSampleTime = 0;
  }

  // Event handlers
  function handleClick(e) {
    resetInteractionTime();
    recordInteraction('click', e);
  }

  function handleDblClick(e) {
    recordInteraction('dblclick', e);
  }

  function handleKeyDown(e) {
    // Only track meaningful keys (skip modifiers alone)
    if (['Control', 'Alt', 'Shift', 'Meta'].indexOf(e.key) !== -1) return;
    resetInteractionTime();
    recordInteraction('keydown', e);
  }

  function handleInput(e) {
    var target = e.target;
    var key = utils.generateSelector(target);

    // Debounce per-element
    clearTimeout(inputDebounce[key]);
    inputDebounce[key] = setTimeout(function() {
      // Sanitize value (don't send passwords)
      var value = '';
      if (target.type !== 'password') {
        value = (target.value || '').substring(0, config.truncateText);
      }

      recordInteraction('input', e, { value: value });
    }, config.debounceInput);
  }

  function handleFocus(e) {
    recordInteraction('focus', e);
  }

  function handleBlur(e) {
    recordInteraction('blur', e);
  }

  function handleScroll(e) {
    var now = Date.now();
    if (now - lastScroll < config.debounceScroll) return;
    lastScroll = now;

    recordInteraction('scroll', {
      target: e.target === document ? document.documentElement : e.target,
      clientX: undefined,
      clientY: undefined
    }, {
      scroll_position: {
        x: window.scrollX || document.documentElement.scrollLeft,
        y: window.scrollY || document.documentElement.scrollTop
      }
    });
  }

  function handleSubmit(e) {
    recordInteraction('submit', e);
  }

  function handleContextMenu(e) {
    recordInteraction('contextmenu', e);
  }

  // Mousemove handler - samples based on interaction time
  function handleMouseMove(e) {
    var now = Date.now();

    // Only sample if within window of last meaningful interaction
    if (now - lastInteractionTime > config.mouseMoveWindow) {
      return;
    }

    // Calculate interaction time (time since last interaction, not wall time)
    var interactionTime = now - interactionTimeBase;

    // Sample at configured interval based on interaction time
    if (interactionTime - lastMouseSampleTime < config.mouseMoveInterval) {
      return;
    }
    lastMouseSampleTime = interactionTime;

    // Record the mousemove sample
    var sample = {
      event_type: 'mousemove',
      position: {
        client_x: e.clientX,
        client_y: e.clientY,
        page_x: e.pageX,
        page_y: e.pageY
      },
      wall_time: now,
      interaction_time: interactionTime,
      timestamp: now
    };

    // Get element under cursor
    var target = document.elementFromPoint(e.clientX, e.clientY);
    if (target) {
      sample.target = getTargetInfo(target);
    }

    mouseMoveBuffer.push(sample);

    // Trim buffer to last minute of samples
    var cutoff = now - config.mouseMoveWindow;
    while (mouseMoveBuffer.length > 0 && mouseMoveBuffer[0].wall_time < cutoff) {
      mouseMoveBuffer.shift();
    }
  }

  // Attach listeners
  function attachListeners() {
    document.addEventListener('click', handleClick, true);
    document.addEventListener('dblclick', handleDblClick, true);
    document.addEventListener('keydown', handleKeyDown, true);
    document.addEventListener('input', handleInput, true);
    document.addEventListener('focus', handleFocus, true);
    document.addEventListener('blur', handleBlur, true);
    document.addEventListener('scroll', handleScroll, true);
    document.addEventListener('submit', handleSubmit, true);
    document.addEventListener('contextmenu', handleContextMenu, true);
    document.addEventListener('mousemove', handleMouseMove, true);
  }

  // Send batch to server
  function sendBatch() {
    if (pendingBatch.length === 0) return;

    var batch = pendingBatch.splice(0, config.sendBatchSize);
    core.send('interactions', { events: batch });
  }

  // Start batch sender
  setInterval(sendBatch, config.sendInterval);

  // Initialize
  attachListeners();

  // Export interactions API
  window.__devtool_interactions = {
    getHistory: function(count) {
      count = count || 50;
      var start = Math.max(0, interactions.length - count);
      return interactions.slice(start);
    },

    getLastClick: function() {
      for (var i = interactions.length - 1; i >= 0; i--) {
        if (interactions[i].event_type === 'click') {
          return interactions[i];
        }
      }
      return null;
    },

    getClicksOn: function(selector) {
      return interactions.filter(function(i) {
        return i.event_type === 'click' &&
               i.target.selector.indexOf(selector) !== -1;
      });
    },

    // Get mousemove samples around a specific interaction
    getMouseTrail: function(interactionTimestamp, windowMs) {
      windowMs = windowMs || 5000;
      var start = interactionTimestamp - windowMs;
      var end = interactionTimestamp + windowMs;

      return mouseMoveBuffer.filter(function(m) {
        return m.wall_time >= start && m.wall_time <= end;
      });
    },

    // Get all mousemove samples in buffer
    getMouseBuffer: function() {
      return mouseMoveBuffer.slice();
    },

    // Get context around last click (click + mouse trail)
    getLastClickContext: function(trailMs) {
      var click = this.getLastClick();
      if (!click) return null;

      return {
        click: click,
        mouseTrail: this.getMouseTrail(click.timestamp, trailMs || 2000)
      };
    },

    clear: function() {
      interactions = [];
      interactionIndex = 0;
      mouseMoveBuffer = [];
    },

    config: config
  };
})();
