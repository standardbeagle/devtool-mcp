package proxy

import (
	"bytes"
	"strings"
	"sync"
)

var (
	// Cache the instrumentation script since it never changes
	cachedScript     string
	cachedScriptOnce sync.Once
)

// instrumentationScript returns JavaScript code for error and performance monitoring.
// The script is cached after first call for performance.
func instrumentationScript() string {
	cachedScriptOnce.Do(func() {
		cachedScript = generateInstrumentationScript()
	})
	return cachedScript
}

// generateInstrumentationScript creates the instrumentation JavaScript.
func generateInstrumentationScript() string {
	return `
<script src="https://cdn.jsdelivr.net/npm/html2canvas@1.4.1/dist/html2canvas.min.js"></script>
<script>
(function() {
  'use strict';

  // Configuration
  // Use relative WebSocket URL to automatically match the current connection
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const WS_URL = protocol + '//' + window.location.host + '/__devtool_metrics';
  let ws = null;
  let reconnectAttempts = 0;
  const MAX_RECONNECT_ATTEMPTS = 5;
  let pendingExecutions = new Map(); // Track pending JS executions

  // WebSocket connection
  function connect() {
    try {
      ws = new WebSocket(WS_URL);

      ws.onopen = function() {
        console.log('[DevTool] Metrics connection established');
        reconnectAttempts = 0;
        sendPageLoad();
      };

      ws.onmessage = function(event) {
        try {
          const message = JSON.parse(event.data);
          handleServerMessage(message);
        } catch (err) {
          console.error('[DevTool] Failed to parse server message:', err);
        }
      };

      ws.onclose = function() {
        console.log('[DevTool] Metrics connection closed');
        if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttempts++;
          setTimeout(connect, 1000 * reconnectAttempts);
        }
      };

      ws.onerror = function(err) {
        console.error('[DevTool] Metrics connection error:', err);
      };
    } catch (err) {
      console.error('[DevTool] Failed to create WebSocket:', err);
    }
  }

  // Handle messages from server
  function handleServerMessage(message) {
    if (message.type === 'execute') {
      executeJavaScript(message.id, message.code);
    }
  }

  // Execute JavaScript sent from server
  function executeJavaScript(execId, code) {
    const startTime = performance.now();
    let result, error;

    try {
      result = eval(code);
      // Convert result to string representation
      if (result === undefined) {
        result = 'undefined';
      } else if (result === null) {
        result = 'null';
      } else if (typeof result === 'function') {
        result = result.toString();
      } else if (typeof result === 'object') {
        try {
          result = JSON.stringify(result, null, 2);
        } catch (e) {
          result = String(result);
        }
      } else {
        result = String(result);
      }
    } catch (err) {
      error = err.toString();
      if (err.stack) {
        error += '\n' + err.stack;
      }
    }

    const duration = performance.now() - startTime;

    send('execution', {
      exec_id: execId,
      result: result || '',
      error: error || '',
      duration: duration,
      timestamp: Date.now()
    });
  }

  // Send metric to server
  function send(type, data) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      try {
        ws.send(JSON.stringify({ type: type, data: data, url: window.location.href }));
      } catch (err) {
        console.error('[DevTool] Failed to send metric:', err);
      }
    }
  }

  // Error tracking
  window.addEventListener('error', function(event) {
    send('error', {
      message: event.message,
      source: event.filename,
      lineno: event.lineno,
      colno: event.colno,
      error: event.error ? event.error.toString() : '',
      stack: event.error ? event.error.stack : '',
      timestamp: Date.now()
    });
  });

  // Promise rejection tracking
  window.addEventListener('unhandledrejection', function(event) {
    send('error', {
      message: 'Unhandled Promise Rejection: ' + event.reason,
      source: '',
      lineno: 0,
      colno: 0,
      error: event.reason ? event.reason.toString() : '',
      stack: event.reason && event.reason.stack ? event.reason.stack : '',
      timestamp: Date.now()
    });
  });

  // Performance tracking
  function sendPageLoad() {
    // Wait for load event
    if (document.readyState === 'complete') {
      capturePerformance();
    } else {
      window.addEventListener('load', capturePerformance);
    }
  }

  function capturePerformance() {
    // Use setTimeout to ensure all metrics are available
    setTimeout(function() {
      try {
        const perf = window.performance;
        if (!perf || !perf.timing) return;

        const timing = perf.timing;
        const navigation = perf.navigation;

        const metrics = {
          navigation_start: timing.navigationStart,
          dom_content_loaded: timing.domContentLoadedEventEnd - timing.navigationStart,
          load_event_end: timing.loadEventEnd - timing.navigationStart,
          dom_interactive: timing.domInteractive - timing.navigationStart,
          dom_complete: timing.domComplete - timing.navigationStart,
          timestamp: Date.now()
        };

        // Paint timing (if available)
        if (perf.getEntriesByType) {
          const paintEntries = perf.getEntriesByType('paint');
          paintEntries.forEach(function(entry) {
            if (entry.name === 'first-paint') {
              metrics.first_paint = Math.round(entry.startTime);
            } else if (entry.name === 'first-contentful-paint') {
              metrics.first_contentful_paint = Math.round(entry.startTime);
            }
          });

          // Resource timing (summary)
          const resources = perf.getEntriesByType('resource');
          if (resources && resources.length > 0) {
            metrics.resources = resources.slice(0, 50).map(function(r) {
              return {
                name: r.name,
                duration: Math.round(r.duration),
                size: r.transferSize || 0
              };
            });
          }
        }

        send('performance', metrics);
      } catch (err) {
        console.error('[DevTool] Failed to capture performance:', err);
      }
    }, 100);
  }

  // ============================================================================
  // PHASE 1: CORE INFRASTRUCTURE UTILITIES
  // ============================================================================

  // Resolve selector, element, or array to element
  function resolveElement(input) {
    if (!input) return null;
    if (input instanceof HTMLElement) return input;
    if (typeof input === 'string') {
      try {
        return document.querySelector(input);
      } catch (e) {
        return null;
      }
    }
    return null;
  }

  // Generate unique CSS selector for element
  function generateSelector(element) {
    if (!element || !(element instanceof HTMLElement)) return '';

    // Try ID first
    if (element.id) {
      return '#' + element.id;
    }

    // Build path from element to root
    var path = [];
    var current = element;

    while (current && current.nodeType === Node.ELEMENT_NODE) {
      var selector = current.nodeName.toLowerCase();

      // Add nth-of-type if needed
      if (current.parentNode) {
        var siblings = Array.prototype.filter.call(current.parentNode.children, function(el) {
          return el.nodeName === current.nodeName;
        });

        if (siblings.length > 1) {
          var index = siblings.indexOf(current) + 1;
          selector += ':nth-of-type(' + index + ')';
        }
      }

      path.unshift(selector);

      if (current.parentNode && current.parentNode.nodeType === Node.ELEMENT_NODE) {
        current = current.parentNode;
      } else {
        break;
      }
    }

    return path.join(' > ');
  }

  // Safe getComputedStyle wrapper
  function safeGetComputed(element, properties) {
    if (!element || !(element instanceof HTMLElement)) {
      return { error: 'Invalid element' };
    }

    try {
      var computed = window.getComputedStyle(element);
      var result = {};

      if (properties && Array.isArray(properties)) {
        // Get specific properties
        for (var i = 0; i < properties.length; i++) {
          var prop = properties[i];
          result[prop] = computed.getPropertyValue(prop) || computed[prop];
        }
      } else {
        // Get all common properties
        var commonProps = [
          'display', 'position', 'zIndex', 'opacity', 'visibility',
          'width', 'height', 'top', 'left', 'right', 'bottom',
          'margin', 'padding', 'border', 'backgroundColor', 'color'
        ];
        for (var j = 0; j < commonProps.length; j++) {
          var key = commonProps[j];
          result[key] = computed[key];
        }
      }

      return result;
    } catch (e) {
      return { error: e.message };
    }
  }

  // Parse CSS value to number (strips 'px', 'em', etc)
  function parseValue(value) {
    if (typeof value === 'number') return value;
    if (typeof value !== 'string') return 0;
    return parseFloat(value) || 0;
  }

  // Get element's bounding box with caching
  function getRect(element) {
    if (!element || !(element instanceof HTMLElement)) return null;
    try {
      return element.getBoundingClientRect();
    } catch (e) {
      return null;
    }
  }

  // Check if element is in viewport
  function isElementInViewport(element) {
    var rect = getRect(element);
    if (!rect) return false;

    return (
      rect.top >= 0 &&
      rect.left >= 0 &&
      rect.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
      rect.right <= (window.innerWidth || document.documentElement.clientWidth)
    );
  }

  // Find stacking context parent
  function getStackingContext(element) {
    if (!element || element === document.documentElement) return null;

    var parent = element.parentElement;
    while (parent && parent !== document.documentElement) {
      var computed = window.getComputedStyle(parent);

      // Check conditions that create stacking context
      if (
        computed.position !== 'static' && computed.zIndex !== 'auto' ||
        parseFloat(computed.opacity) < 1 ||
        computed.transform !== 'none' ||
        computed.filter !== 'none' ||
        computed.perspective !== 'none' ||
        computed.willChange === 'transform' || computed.willChange === 'opacity'
      ) {
        return parent;
      }

      parent = parent.parentElement;
    }

    return document.documentElement;
  }

  // Overlay management system
  var overlayState = {
    container: null,
    overlays: {},
    highlights: {},
    labels: {},
    nextId: 1
  };

  function initOverlayContainer() {
    if (overlayState.container) return overlayState.container;

    var container = document.createElement('div');
    container.id = '__devtool-overlays';
    container.style.cssText = [
      'position: fixed',
      'top: 0',
      'left: 0',
      'right: 0',
      'bottom: 0',
      'pointer-events: none',
      'z-index: 2147483647',
      'overflow: hidden'
    ].join(';');

    document.documentElement.appendChild(container);
    overlayState.container = container;
    return container;
  }

  function removeOverlayContainer() {
    if (overlayState.container && overlayState.container.parentNode) {
      overlayState.container.parentNode.removeChild(overlayState.container);
      overlayState.container = null;
    }
  }

  function createOverlayElement(type, config) {
    var el = document.createElement('div');
    el.className = '__devtool-overlay-' + type;
    el.style.position = 'absolute';
    el.style.pointerEvents = 'none';
    return el;
  }

  // ============================================================================
  // PHASE 2: ELEMENT INSPECTION PRIMITIVES
  // ============================================================================

  function getElementInfo(selector) {
    var el = resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var attrs = {};
      for (var i = 0; i < el.attributes.length; i++) {
        var attr = el.attributes[i];
        attrs[attr.name] = attr.value;
      }

      return {
        element: el,
        selector: generateSelector(el),
        tag: el.tagName.toLowerCase(),
        id: el.id || null,
        classes: Array.prototype.slice.call(el.classList),
        attributes: attrs
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  function getPosition(selector) {
    var el = resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var rect = getRect(el);
      if (!rect) return { error: 'Failed to get bounding rect' };

      return {
        rect: {
          x: rect.x,
          y: rect.y,
          width: rect.width,
          height: rect.height,
          top: rect.top,
          right: rect.right,
          bottom: rect.bottom,
          left: rect.left
        },
        viewport: {
          x: rect.left,
          y: rect.top
        },
        document: {
          x: rect.left + window.scrollX,
          y: rect.top + window.scrollY
        },
        scroll: {
          x: window.scrollX,
          y: window.scrollY
        }
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  function getComputed(selector, properties) {
    var el = resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    return safeGetComputed(el, properties);
  }

  function getBox(selector) {
    var el = resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var computed = window.getComputedStyle(el);

      return {
        margin: {
          top: parseValue(computed.marginTop),
          right: parseValue(computed.marginRight),
          bottom: parseValue(computed.marginBottom),
          left: parseValue(computed.marginLeft)
        },
        border: {
          top: parseValue(computed.borderTopWidth),
          right: parseValue(computed.borderRightWidth),
          bottom: parseValue(computed.borderBottomWidth),
          left: parseValue(computed.borderLeftWidth)
        },
        padding: {
          top: parseValue(computed.paddingTop),
          right: parseValue(computed.paddingRight),
          bottom: parseValue(computed.paddingBottom),
          left: parseValue(computed.paddingLeft)
        },
        content: {
          width: el.clientWidth - parseValue(computed.paddingLeft) - parseValue(computed.paddingRight),
          height: el.clientHeight - parseValue(computed.paddingTop) - parseValue(computed.paddingBottom)
        }
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  function getLayout(selector) {
    var el = resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var computed = window.getComputedStyle(el);
      var display = computed.display;

      var result = {
        display: display,
        position: computed.position,
        float: computed.float,
        clear: computed.clear
      };

      // Flexbox information
      if (display.indexOf('flex') !== -1) {
        result.flexbox = {
          container: true,
          direction: computed.flexDirection,
          wrap: computed.flexWrap,
          justifyContent: computed.justifyContent,
          alignItems: computed.alignItems,
          alignContent: computed.alignContent
        };
      } else if (el.parentElement && window.getComputedStyle(el.parentElement).display.indexOf('flex') !== -1) {
        result.flexbox = {
          container: false,
          flex: computed.flex,
          flexGrow: computed.flexGrow,
          flexShrink: computed.flexShrink,
          flexBasis: computed.flexBasis,
          alignSelf: computed.alignSelf,
          order: computed.order
        };
      }

      // Grid information
      if (display.indexOf('grid') !== -1) {
        result.grid = {
          container: true,
          templateColumns: computed.gridTemplateColumns,
          templateRows: computed.gridTemplateRows,
          gap: computed.gap,
          autoFlow: computed.gridAutoFlow
        };
      } else if (el.parentElement && window.getComputedStyle(el.parentElement).display.indexOf('grid') !== -1) {
        result.grid = {
          container: false,
          column: computed.gridColumn,
          row: computed.gridRow,
          area: computed.gridArea
        };
      }

      return result;
    } catch (e) {
      return { error: e.message };
    }
  }

  function getContainer(selector) {
    var el = resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var computed = window.getComputedStyle(el);

      return {
        type: computed.containerType || 'normal',
        name: computed.containerName || null,
        contain: computed.contain || 'none'
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  function getStacking(selector) {
    var el = resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var computed = window.getComputedStyle(el);
      var context = getStackingContext(el);

      return {
        zIndex: computed.zIndex,
        position: computed.position,
        context: context ? generateSelector(context) : null,
        opacity: parseFloat(computed.opacity),
        transform: computed.transform,
        filter: computed.filter
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  function getTransform(selector) {
    var el = resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var computed = window.getComputedStyle(el);
      var transform = computed.transform;

      if (!transform || transform === 'none') {
        return {
          matrix: null,
          translate: { x: 0, y: 0 },
          rotate: 0,
          scale: { x: 1, y: 1 }
        };
      }

      return {
        matrix: transform,
        transform: transform,
        transformOrigin: computed.transformOrigin
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  function getOverflow(selector) {
    var el = resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var computed = window.getComputedStyle(el);

      return {
        x: computed.overflowX,
        y: computed.overflowY,
        scrollWidth: el.scrollWidth,
        scrollHeight: el.scrollHeight,
        clientWidth: el.clientWidth,
        clientHeight: el.clientHeight,
        scrollTop: el.scrollTop,
        scrollLeft: el.scrollLeft,
        hasOverflow: el.scrollWidth > el.clientWidth || el.scrollHeight > el.clientHeight
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  // Global __devtool API
  window.__devtool = {
    // ========================================================================
    // LOGGING API
    // ========================================================================

    // Send custom log message
    log: function(message, level, data) {
      level = level || 'info';
      send('custom_log', {
        level: level,
        message: String(message),
        data: data || {},
        timestamp: Date.now()
      });
    },

    // Convenience methods
    debug: function(message, data) {
      this.log(message, 'debug', data);
    },

    info: function(message, data) {
      this.log(message, 'info', data);
    },

    warn: function(message, data) {
      this.log(message, 'warn', data);
    },

    error: function(message, data) {
      this.log(message, 'error', data);
    },

    // ========================================================================
    // ELEMENT INSPECTION PRIMITIVES
    // ========================================================================

    getElementInfo: getElementInfo,
    getPosition: getPosition,
    getComputed: getComputed,
    getBox: getBox,
    getLayout: getLayout,
    getContainer: getContainer,
    getStacking: getStacking,
    getTransform: getTransform,
    getOverflow: getOverflow,

    // ========================================================================
    // TREE WALKING PRIMITIVES
    // ========================================================================

    walkChildren: function(selector, depth, filter) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      depth = depth || 1;
      var results = [];

      function walk(element, currentDepth) {
        if (currentDepth > depth) return;

        var children = Array.prototype.slice.call(element.children);
        for (var i = 0; i < children.length; i++) {
          var child = children[i];

          if (!filter || filter(child)) {
            results.push({
              element: child,
              selector: generateSelector(child),
              depth: currentDepth
            });
          }

          if (currentDepth < depth) {
            walk(child, currentDepth + 1);
          }
        }
      }

      try {
        walk(el, 1);
        return { elements: results, count: results.length };
      } catch (e) {
        return { error: e.message };
      }
    },

    walkParents: function(selector) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      var parents = [];
      var current = el.parentElement;

      while (current) {
        parents.push({
          element: current,
          selector: generateSelector(current),
          tag: current.tagName.toLowerCase()
        });
        current = current.parentElement;
      }

      return { parents: parents, count: parents.length };
    },

    findAncestor: function(selector, condition) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      if (typeof condition !== 'function') {
        return { error: 'Condition must be a function' };
      }

      var current = el.parentElement;
      while (current) {
        if (condition(current)) {
          return {
            element: current,
            selector: generateSelector(current)
          };
        }
        current = current.parentElement;
      }

      return { found: false };
    },

    // ========================================================================
    // VISUAL STATE PRIMITIVES
    // ========================================================================

    isVisible: function(selector) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      try {
        var computed = window.getComputedStyle(el);
        var rect = getRect(el);

        if (!rect) {
          return { visible: false, reason: 'No bounding rect' };
        }

        if (computed.display === 'none') {
          return { visible: false, reason: 'display: none' };
        }

        if (computed.visibility === 'hidden') {
          return { visible: false, reason: 'visibility: hidden' };
        }

        if (parseFloat(computed.opacity) === 0) {
          return { visible: false, reason: 'opacity: 0' };
        }

        if (rect.width === 0 || rect.height === 0) {
          return { visible: false, reason: 'zero size' };
        }

        return { visible: true, area: rect.width * rect.height };
      } catch (e) {
        return { error: e.message };
      }
    },

    isInViewport: function(selector) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      try {
        var rect = getRect(el);
        if (!rect) return { error: 'Failed to get bounding rect' };

        var viewportHeight = window.innerHeight || document.documentElement.clientHeight;
        var viewportWidth = window.innerWidth || document.documentElement.clientWidth;

        var intersecting = !(
          rect.bottom < 0 ||
          rect.top > viewportHeight ||
          rect.right < 0 ||
          rect.left > viewportWidth
        );

        var visibleWidth = Math.min(rect.right, viewportWidth) - Math.max(rect.left, 0);
        var visibleHeight = Math.min(rect.bottom, viewportHeight) - Math.max(rect.top, 0);
        var visibleArea = Math.max(0, visibleWidth) * Math.max(0, visibleHeight);
        var totalArea = rect.width * rect.height;
        var ratio = totalArea > 0 ? visibleArea / totalArea : 0;

        return {
          intersecting: intersecting,
          ratio: ratio,
          rect: rect,
          fullyVisible: ratio === 1
        };
      } catch (e) {
        return { error: e.message };
      }
    },

    checkOverlap: function(selector1, selector2) {
      var el1 = resolveElement(selector1);
      var el2 = resolveElement(selector2);

      if (!el1 || !el2) return { error: 'Element not found' };

      try {
        var rect1 = getRect(el1);
        var rect2 = getRect(el2);

        if (!rect1 || !rect2) return { error: 'Failed to get bounding rects' };

        var overlaps = !(
          rect1.right < rect2.left ||
          rect1.left > rect2.right ||
          rect1.bottom < rect2.top ||
          rect1.top > rect2.bottom
        );

        if (!overlaps) {
          return { overlaps: false };
        }

        var overlapLeft = Math.max(rect1.left, rect2.left);
        var overlapRight = Math.min(rect1.right, rect2.right);
        var overlapTop = Math.max(rect1.top, rect2.top);
        var overlapBottom = Math.min(rect1.bottom, rect2.bottom);

        var overlapArea = (overlapRight - overlapLeft) * (overlapBottom - overlapTop);

        return {
          overlaps: true,
          area: overlapArea,
          rect: {
            left: overlapLeft,
            right: overlapRight,
            top: overlapTop,
            bottom: overlapBottom
          }
        };
      } catch (e) {
        return { error: e.message };
      }
    },

    // ========================================================================
    // LAYOUT DIAGNOSTIC PRIMITIVES
    // ========================================================================

    findOverflows: function() {
      var elements = document.querySelectorAll('*');
      var results = [];

      for (var i = 0; i < elements.length; i++) {
        var el = elements[i];
        var overflow = getOverflow(el);

        if (overflow && overflow.hasOverflow) {
          results.push({
            element: el,
            selector: generateSelector(el),
            type: overflow.x === 'hidden' || overflow.y === 'hidden' ? 'hidden' : 'scrollable',
            scrollWidth: overflow.scrollWidth,
            scrollHeight: overflow.scrollHeight,
            clientWidth: overflow.clientWidth,
            clientHeight: overflow.clientHeight
          });
        }
      }

      return { overflows: results, count: results.length };
    },

    findStackingContexts: function() {
      var elements = document.querySelectorAll('*');
      var contexts = [];

      for (var i = 0; i < elements.length; i++) {
        var el = elements[i];
        var computed = window.getComputedStyle(el);

        var isContext = (
          (computed.position !== 'static' && computed.zIndex !== 'auto') ||
          parseFloat(computed.opacity) < 1 ||
          computed.transform !== 'none' ||
          computed.filter !== 'none' ||
          computed.perspective !== 'none'
        );

        if (isContext) {
          contexts.push({
            element: el,
            selector: generateSelector(el),
            zIndex: computed.zIndex,
            reason: []
          });

          var last = contexts[contexts.length - 1];
          if (computed.position !== 'static' && computed.zIndex !== 'auto') {
            last.reason.push('positioned');
          }
          if (parseFloat(computed.opacity) < 1) {
            last.reason.push('opacity');
          }
          if (computed.transform !== 'none') {
            last.reason.push('transform');
          }
          if (computed.filter !== 'none') {
            last.reason.push('filter');
          }
        }
      }

      return { contexts: contexts, count: contexts.length };
    },

    findOffscreen: function() {
      var elements = document.querySelectorAll('*');
      var results = [];

      for (var i = 0; i < elements.length; i++) {
        var el = elements[i];
        var viewport = this.isInViewport(el);

        if (viewport && !viewport.intersecting) {
          var rect = viewport.rect;
          var direction = [];

          if (rect.bottom < 0) direction.push('above');
          if (rect.top > window.innerHeight) direction.push('below');
          if (rect.right < 0) direction.push('left');
          if (rect.left > window.innerWidth) direction.push('right');

          results.push({
            element: el,
            selector: generateSelector(el),
            direction: direction,
            rect: rect
          });
        }
      }

      return { offscreen: results, count: results.length };
    },

    // ========================================================================
    // VISUAL OVERLAY SYSTEM
    // ========================================================================

    highlight: function(selector, config) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      config = config || {};
      var color = config.color || 'rgba(0, 123, 255, 0.3)';
      var duration = config.duration;
      var id = 'highlight-' + overlayState.nextId++;

      try {
        initOverlayContainer();
        var rect = getRect(el);

        var highlight = createOverlayElement('highlight', config);
        highlight.id = id;
        highlight.style.cssText += [
          'top: ' + rect.top + 'px',
          'left: ' + rect.left + 'px',
          'width: ' + rect.width + 'px',
          'height: ' + rect.height + 'px',
          'background-color: ' + color,
          'border: 2px solid ' + (config.borderColor || '#007bff'),
          'box-sizing: border-box'
        ].join(';');

        overlayState.container.appendChild(highlight);
        overlayState.highlights[id] = highlight;

        if (duration) {
          setTimeout(function() {
            window.__devtool.removeHighlight(id);
          }, duration);
        }

        return { highlightId: id };
      } catch (e) {
        return { error: e.message };
      }
    },

    removeHighlight: function(highlightId) {
      var highlight = overlayState.highlights[highlightId];
      if (highlight && highlight.parentNode) {
        highlight.parentNode.removeChild(highlight);
        delete overlayState.highlights[highlightId];
      }
    },

    clearAllOverlays: function() {
      overlayState.overlays = {};
      overlayState.highlights = {};
      overlayState.labels = {};

      if (overlayState.container) {
        removeOverlayContainer();
      }
    },

    // ========================================================================
    // INTERACTIVE PRIMITIVES
    // ========================================================================

    measureBetween: function(selector1, selector2) {
      var el1 = resolveElement(selector1);
      var el2 = resolveElement(selector2);

      if (!el1 || !el2) return { error: 'Element not found' };

      try {
        var rect1 = getRect(el1);
        var rect2 = getRect(el2);

        if (!rect1 || !rect2) return { error: 'Failed to get bounding rects' };

        var center1 = {
          x: rect1.left + rect1.width / 2,
          y: rect1.top + rect1.height / 2
        };

        var center2 = {
          x: rect2.left + rect2.width / 2,
          y: rect2.top + rect2.height / 2
        };

        var dx = center2.x - center1.x;
        var dy = center2.y - center1.y;
        var diagonal = Math.sqrt(dx * dx + dy * dy);

        return {
          distance: {
            x: Math.abs(dx),
            y: Math.abs(dy),
            diagonal: diagonal
          },
          direction: {
            horizontal: dx > 0 ? 'right' : 'left',
            vertical: dy > 0 ? 'down' : 'up'
          }
        };
      } catch (e) {
        return { error: e.message };
      }
    },

    // ========================================================================
    // COMPOSITE CONVENIENCE FUNCTIONS
    // ========================================================================

    inspect: function(selector) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      var info = getElementInfo(selector);
      var position = getPosition(selector);
      var box = getBox(selector);
      var layout = getLayout(selector);
      var stacking = getStacking(selector);
      var container = getContainer(selector);
      var visibility = this.isVisible(selector);
      var viewport = this.isInViewport(selector);

      return {
        info: info,
        position: position,
        box: box,
        layout: layout,
        stacking: stacking,
        container: container,
        visibility: visibility,
        viewport: viewport
      };
    },

    diagnoseLayout: function(selector) {
      var el = resolveElement(selector);
      var overflows = this.findOverflows();
      var contexts = this.findStackingContexts();
      var offscreen = this.findOffscreen();

      var result = {
        overflows: overflows,
        stackingContexts: contexts,
        offscreen: offscreen
      };

      if (selector) {
        var stacking = getStacking(selector);
        result.element = {
          selector: generateSelector(el),
          stacking: stacking
        };
      }

      return result;
    },

    showLayout: function(config) {
      config = config || {};
      console.log('[DevTool] Layout visualization not yet fully implemented');
      console.log('[DevTool] Use highlight() to mark specific elements');
      return { message: 'Use highlight() for now' };
    },

    // ========================================================================
    // PHASE 7: INTERACTIVE PRIMITIVES (ADVANCED)
    // ========================================================================

    selectElement: function() {
      return new Promise(function(resolve, reject) {
        var overlay = document.createElement('div');
        overlay.style.cssText = [
          'position: fixed',
          'top: 0',
          'left: 0',
          'right: 0',
          'bottom: 0',
          'z-index: 2147483646',
          'cursor: crosshair',
          'background: rgba(0, 0, 0, 0.1)'
        ].join(';');

        var highlightBox = document.createElement('div');
        highlightBox.style.cssText = [
          'position: absolute',
          'border: 2px solid #007bff',
          'background: rgba(0, 123, 255, 0.1)',
          'pointer-events: none',
          'display: none'
        ].join(';');
        overlay.appendChild(highlightBox);

        var labelBox = document.createElement('div');
        labelBox.style.cssText = [
          'position: absolute',
          'background: #007bff',
          'color: white',
          'padding: 4px 8px',
          'font-size: 12px',
          'font-family: monospace',
          'border-radius: 3px',
          'pointer-events: none',
          'display: none',
          'white-space: nowrap'
        ].join(';');
        overlay.appendChild(labelBox);

        function cleanup() {
          if (overlay.parentNode) {
            overlay.parentNode.removeChild(overlay);
          }
        }

        overlay.addEventListener('mousemove', function(e) {
          var target = document.elementFromPoint(e.clientX, e.clientY);
          if (!target || target === overlay || target === highlightBox || target === labelBox) {
            highlightBox.style.display = 'none';
            labelBox.style.display = 'none';
            return;
          }

          var rect = target.getBoundingClientRect();
          highlightBox.style.cssText += [
            'display: block',
            'top: ' + rect.top + 'px',
            'left: ' + rect.left + 'px',
            'width: ' + rect.width + 'px',
            'height: ' + rect.height + 'px'
          ].join(';');

          var selector = generateSelector(target);
          labelBox.textContent = selector;
          labelBox.style.cssText += [
            'display: block',
            'top: ' + (rect.top - 25) + 'px',
            'left: ' + rect.left + 'px'
          ].join(';');
        });

        overlay.addEventListener('click', function(e) {
          e.preventDefault();
          e.stopPropagation();

          var target = document.elementFromPoint(e.clientX, e.clientY);
          if (target && target !== overlay && target !== highlightBox && target !== labelBox) {
            var selector = generateSelector(target);
            cleanup();
            resolve(selector);
          }
        });

        overlay.addEventListener('keydown', function(e) {
          if (e.key === 'Escape') {
            cleanup();
            reject(new Error('Selection cancelled'));
          }
        });

        document.body.appendChild(overlay);
        overlay.focus();
      });
    },

    waitForElement: function(selector, timeout) {
      timeout = timeout || 5000;
      var startTime = Date.now();

      return new Promise(function(resolve, reject) {
        var el = resolveElement(selector);
        if (el) {
          resolve(el);
          return;
        }

        var observer = new MutationObserver(function(mutations) {
          var el = resolveElement(selector);
          if (el) {
            observer.disconnect();
            resolve(el);
          } else if (Date.now() - startTime > timeout) {
            observer.disconnect();
            reject(new Error('Timeout waiting for element: ' + selector));
          }
        });

        observer.observe(document.body, {
          childList: true,
          subtree: true
        });

        setTimeout(function() {
          observer.disconnect();
          reject(new Error('Timeout waiting for element: ' + selector));
        }, timeout);
      });
    },

    ask: function(question, options) {
      return new Promise(function(resolve, reject) {
        var modal = document.createElement('div');
        modal.style.cssText = [
          'position: fixed',
          'top: 50%',
          'left: 50%',
          'transform: translate(-50%, -50%)',
          'background: white',
          'padding: 20px',
          'border-radius: 8px',
          'box-shadow: 0 4px 20px rgba(0,0,0,0.3)',
          'z-index: 2147483647',
          'min-width: 300px',
          'max-width: 500px'
        ].join(';');

        var overlay = document.createElement('div');
        overlay.style.cssText = [
          'position: fixed',
          'top: 0',
          'left: 0',
          'right: 0',
          'bottom: 0',
          'background: rgba(0,0,0,0.5)',
          'z-index: 2147483646'
        ].join(';');

        var title = document.createElement('h3');
        title.style.cssText = 'margin: 0 0 15px 0; color: #333;';
        title.textContent = question;
        modal.appendChild(title);

        var buttonContainer = document.createElement('div');
        buttonContainer.style.cssText = 'display: flex; gap: 10px; flex-wrap: wrap;';

        options = options || ['Yes', 'No'];
        for (var i = 0; i < options.length; i++) {
          (function(option) {
            var btn = document.createElement('button');
            btn.textContent = option;
            btn.style.cssText = [
              'padding: 10px 20px',
              'border: none',
              'border-radius: 4px',
              'background: #007bff',
              'color: white',
              'cursor: pointer',
              'font-size: 14px'
            ].join(';');

            btn.addEventListener('mouseover', function() {
              this.style.background = '#0056b3';
            });

            btn.addEventListener('mouseout', function() {
              this.style.background = '#007bff';
            });

            btn.addEventListener('click', function() {
              cleanup();
              resolve(option);
            });

            buttonContainer.appendChild(btn);
          })(options[i]);
        }

        modal.appendChild(buttonContainer);

        function cleanup() {
          if (overlay.parentNode) overlay.parentNode.removeChild(overlay);
          if (modal.parentNode) modal.parentNode.removeChild(modal);
        }

        overlay.addEventListener('click', function() {
          cleanup();
          reject(new Error('Question cancelled'));
        });

        document.body.appendChild(overlay);
        document.body.appendChild(modal);
      });
    },

    // ========================================================================
    // PHASE 8: STATE CAPTURE PRIMITIVES
    // ========================================================================

    captureDOM: function() {
      try {
        var html = document.documentElement.outerHTML;
        var hash = 0;
        for (var i = 0; i < html.length; i++) {
          var char = html.charCodeAt(i);
          hash = ((hash << 5) - hash) + char;
          hash = hash & hash;
        }

        return {
          snapshot: html,
          hash: hash.toString(16),
          timestamp: Date.now(),
          url: window.location.href,
          size: html.length
        };
      } catch (e) {
        return { error: e.message };
      }
    },

    captureStyles: function(selector) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      try {
        var computed = window.getComputedStyle(el);
        var inline = el.style.cssText;

        var computedObj = {};
        for (var i = 0; i < computed.length; i++) {
          var prop = computed[i];
          computedObj[prop] = computed.getPropertyValue(prop);
        }

        return {
          selector: generateSelector(el),
          computed: computedObj,
          inline: inline,
          timestamp: Date.now()
        };
      } catch (e) {
        return { error: e.message };
      }
    },

    captureState: function(keys) {
      try {
        var state = {
          timestamp: Date.now(),
          url: window.location.href
        };

        if (!keys || keys.indexOf('localStorage') !== -1) {
          state.localStorage = {};
          try {
            for (var i = 0; i < localStorage.length; i++) {
              var key = localStorage.key(i);
              state.localStorage[key] = localStorage.getItem(key);
            }
          } catch (e) {
            state.localStorage = { error: 'Access denied' };
          }
        }

        if (!keys || keys.indexOf('sessionStorage') !== -1) {
          state.sessionStorage = {};
          try {
            for (var j = 0; j < sessionStorage.length; j++) {
              var skey = sessionStorage.key(j);
              state.sessionStorage[skey] = sessionStorage.getItem(skey);
            }
          } catch (e) {
            state.sessionStorage = { error: 'Access denied' };
          }
        }

        if (!keys || keys.indexOf('cookies') !== -1) {
          state.cookies = document.cookie;
        }

        return state;
      } catch (e) {
        return { error: e.message };
      }
    },

    captureNetwork: function() {
      try {
        var resources = [];
        if (window.performance && window.performance.getEntriesByType) {
          var entries = window.performance.getEntriesByType('resource');
          for (var i = 0; i < entries.length; i++) {
            var entry = entries[i];
            resources.push({
              name: entry.name,
              type: entry.initiatorType,
              duration: entry.duration,
              size: entry.transferSize || 0,
              startTime: entry.startTime
            });
          }
        }

        return {
          resources: resources,
          count: resources.length,
          timestamp: Date.now()
        };
      } catch (e) {
        return { error: e.message };
      }
    },

    // ========================================================================
    // PHASE 9: ACCESSIBILITY PRIMITIVES
    // ========================================================================

    getA11yInfo: function(selector) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      try {
        var computed = window.getComputedStyle(el);

        var ariaAttrs = {};
        for (var i = 0; i < el.attributes.length; i++) {
          var attr = el.attributes[i];
          if (attr.name.startsWith('aria-')) {
            ariaAttrs[attr.name] = attr.value;
          }
        }

        return {
          role: el.getAttribute('role') || el.tagName.toLowerCase(),
          aria: ariaAttrs,
          tabindex: el.tabIndex,
          focusable: el.tabIndex >= 0 || ['A', 'BUTTON', 'INPUT', 'SELECT', 'TEXTAREA'].indexOf(el.tagName) !== -1,
          label: el.getAttribute('aria-label') || el.getAttribute('aria-labelledby') || el.textContent.trim().substring(0, 50),
          hidden: computed.display === 'none' || computed.visibility === 'hidden' || el.getAttribute('aria-hidden') === 'true'
        };
      } catch (e) {
        return { error: e.message };
      }
    },

    getContrast: function(selector) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      try {
        var computed = window.getComputedStyle(el);

        function parseColor(color) {
          var match = color.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
          if (match) {
            return {
              r: parseInt(match[1]),
              g: parseInt(match[2]),
              b: parseInt(match[3])
            };
          }
          return null;
        }

        function getLuminance(rgb) {
          var rsRGB = rgb.r / 255;
          var gsRGB = rgb.g / 255;
          var bsRGB = rgb.b / 255;

          var r = rsRGB <= 0.03928 ? rsRGB / 12.92 : Math.pow((rsRGB + 0.055) / 1.055, 2.4);
          var g = gsRGB <= 0.03928 ? gsRGB / 12.92 : Math.pow((gsRGB + 0.055) / 1.055, 2.4);
          var b = bsRGB <= 0.03928 ? bsRGB / 12.92 : Math.pow((bsRGB + 0.055) / 1.055, 2.4);

          return 0.2126 * r + 0.7152 * g + 0.0722 * b;
        }

        function getContrastRatio(fg, bg) {
          var l1 = getLuminance(fg);
          var l2 = getLuminance(bg);
          var lighter = Math.max(l1, l2);
          var darker = Math.min(l1, l2);
          return (lighter + 0.05) / (darker + 0.05);
        }

        var fgColor = parseColor(computed.color);
        var bgColor = parseColor(computed.backgroundColor);

        if (!fgColor || !bgColor) {
          return { error: 'Could not parse colors' };
        }

        var ratio = getContrastRatio(fgColor, bgColor);

        return {
          foreground: computed.color,
          background: computed.backgroundColor,
          ratio: ratio.toFixed(2),
          passes: {
            AA: ratio >= 4.5,
            AALarge: ratio >= 3,
            AAA: ratio >= 7,
            AAALarge: ratio >= 4.5
          }
        };
      } catch (e) {
        return { error: e.message };
      }
    },

    getTabOrder: function(container) {
      var root = container ? resolveElement(container) : document.body;
      if (!root) return { error: 'Container not found' };

      try {
        var focusable = root.querySelectorAll(
          'a[href], button, input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );

        var elements = [];
        for (var i = 0; i < focusable.length; i++) {
          var el = focusable[i];
          elements.push({
            element: el,
            selector: generateSelector(el),
            tabindex: el.tabIndex,
            tag: el.tagName.toLowerCase()
          });
        }

        elements.sort(function(a, b) {
          if (a.tabindex === 0 && b.tabindex === 0) return 0;
          if (a.tabindex === 0) return 1;
          if (b.tabindex === 0) return -1;
          return a.tabindex - b.tabindex;
        });

        return { elements: elements, count: elements.length };
      } catch (e) {
        return { error: e.message };
      }
    },

    getScreenReaderText: function(selector) {
      var el = resolveElement(selector);
      if (!el) return { error: 'Element not found' };

      try {
        var ariaLabel = el.getAttribute('aria-label');
        if (ariaLabel) return ariaLabel;

        var ariaLabelledBy = el.getAttribute('aria-labelledby');
        if (ariaLabelledBy) {
          var labelEl = document.getElementById(ariaLabelledBy);
          if (labelEl) return labelEl.textContent.trim();
        }

        if (el.tagName === 'IMG') {
          return el.alt || '(No alt text)';
        }

        if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
          var label = document.querySelector('label[for="' + el.id + '"]');
          if (label) return label.textContent.trim();
        }

        return el.textContent.trim();
      } catch (e) {
        return { error: e.message };
      }
    },

    auditAccessibility: function() {
      try {
        var errors = [];
        var warnings = [];

        var imgs = document.querySelectorAll('img');
        for (var i = 0; i < imgs.length; i++) {
          var img = imgs[i];
          if (!img.alt) {
            errors.push({
              rule: 'img-alt',
              element: generateSelector(img),
              message: 'Image missing alt text',
              fix: 'Add alt attribute to image'
            });
          }
        }

        var buttons = document.querySelectorAll('button');
        for (var j = 0; j < buttons.length; j++) {
          var btn = buttons[j];
          if (!btn.textContent.trim() && !btn.getAttribute('aria-label')) {
            errors.push({
              rule: 'button-name',
              element: generateSelector(btn),
              message: 'Button has no accessible name',
              fix: 'Add text content or aria-label'
            });
          }
        }

        var inputs = document.querySelectorAll('input, textarea, select');
        for (var k = 0; k < inputs.length; k++) {
          var input = inputs[k];
          if (input.id) {
            var label = document.querySelector('label[for="' + input.id + '"]');
            if (!label && !input.getAttribute('aria-label')) {
              warnings.push({
                rule: 'label-missing',
                element: generateSelector(input),
                message: 'Form control missing label',
                fix: 'Add <label> or aria-label'
              });
            }
          }
        }

        var score = Math.max(0, 100 - (errors.length * 10) - (warnings.length * 5));

        return {
          errors: errors,
          warnings: warnings,
          score: score,
          summary: {
            errors: errors.length,
            warnings: warnings.length
          }
        };
      } catch (e) {
        return { error: e.message };
      }
    },

    // Take screenshot and save to server
    // Usage:
    //   screenshot() - captures entire page
    //   screenshot('my-name') - captures entire page with custom name
    //   screenshot('my-name', '#selector') - captures specific element
    //   screenshot(null, '.class') - captures element with auto-generated name
    screenshot: function(name, selector) {
      return new Promise(function(resolve, reject) {
        if (typeof html2canvas === 'undefined') {
          reject(new Error('html2canvas not loaded'));
          return;
        }

        // Handle different parameter combinations
        // screenshot(selector) where selector is a string starting with . or #
        if (typeof name === 'string' && !selector && (name.startsWith('.') || name.startsWith('#') || name.startsWith('['))) {
          selector = name;
          name = null;
        }

        name = name || 'screenshot_' + Date.now();

        // Determine target element
        var targetElement = document.body;
        if (selector) {
          try {
            targetElement = document.querySelector(selector);
            if (!targetElement) {
              reject(new Error('Element not found: ' + selector));
              return;
            }
          } catch (err) {
            reject(new Error('Invalid selector: ' + selector + ' - ' + err.message));
            return;
          }
        }

        html2canvas(targetElement, {
          allowTaint: true,
          useCORS: true,
          logging: false,
          scrollY: -window.scrollY,
          scrollX: -window.scrollX,
          windowWidth: targetElement === document.body ? document.documentElement.scrollWidth : undefined,
          windowHeight: targetElement === document.body ? document.documentElement.scrollHeight : undefined
        }).then(function(canvas) {
          const dataUrl = canvas.toDataURL('image/png');
          const width = canvas.width;
          const height = canvas.height;

          send('screenshot', {
            name: name,
            data: dataUrl,
            width: width,
            height: height,
            format: 'png',
            selector: selector || 'body',
            timestamp: Date.now()
          });

          resolve({
            name: name,
            width: width,
            height: height,
            selector: selector || 'body'
          });
        }).catch(function(err) {
          reject(err);
        });
      });
    },

    // Check if connected
    isConnected: function() {
      return ws && ws.readyState === WebSocket.OPEN;
    },

    // Get connection status
    getStatus: function() {
      if (!ws) return 'not_initialized';
      switch (ws.readyState) {
        case WebSocket.CONNECTING: return 'connecting';
        case WebSocket.OPEN: return 'connected';
        case WebSocket.CLOSING: return 'closing';
        case WebSocket.CLOSED: return 'closed';
        default: return 'unknown';
      }
    },

    // ========================================================================
    // LAYOUT ROBUSTNESS & FRAGILITY DETECTION
    // ========================================================================

    // Check for text truncation, overflow, and font size issues
    checkTextFragility: function(selector) {
      var root = selector ? resolveElement(selector) : document.body;
      if (!root) return { error: 'Element not found' };

      var issues = [];
      var elements = root.querySelectorAll('*');

      for (var i = 0; i < elements.length; i++) {
        var el = elements[i];

        // Skip non-visible elements
        var computed = window.getComputedStyle(el);
        if (computed.display === 'none' || computed.visibility === 'hidden') continue;

        // Check for text truncation with ellipsis
        var hasEllipsis = computed.textOverflow === 'ellipsis';
        var hasOverflowHidden = computed.overflow === 'hidden' ||
                                computed.overflowX === 'hidden';
        var isTruncatedH = el.scrollWidth > el.clientWidth;
        var isTruncatedV = el.scrollHeight > el.clientHeight;

        if (hasEllipsis && hasOverflowHidden && isTruncatedH) {
          issues.push({
            selector: generateSelector(el),
            type: 'truncation-ellipsis',
            severity: 'error',
            message: 'Text is truncated with ellipsis - content loss (WCAG 1.4.10)',
            details: {
              scrollWidth: el.scrollWidth,
              clientWidth: el.clientWidth,
              excess: el.scrollWidth - el.clientWidth,
              textContent: el.textContent.substring(0, 100) + (el.textContent.length > 100 ? '...' : ''),
              overflowStyle: computed.overflow,
              textOverflow: computed.textOverflow
            },
            wcag: '1.4.10',
            fix: 'Allow text to wrap or expand container; avoid text-overflow: ellipsis'
          });
        } else if (hasOverflowHidden && isTruncatedH && el.textContent.trim()) {
          // Hidden overflow clipping text without ellipsis
          issues.push({
            selector: generateSelector(el),
            type: 'overflow-clipped',
            severity: 'warning',
            message: 'Text content is clipped by overflow: hidden',
            details: {
              scrollWidth: el.scrollWidth,
              clientWidth: el.clientWidth,
              excess: el.scrollWidth - el.clientWidth,
              overflowStyle: computed.overflow
            },
            wcag: '1.4.10',
            fix: 'Allow container to expand or text to wrap'
          });
        }

        // Check for white-space: nowrap with potential overflow
        if (computed.whiteSpace === 'nowrap' && el.textContent.trim()) {
          var parentWidth = el.parentElement ? el.parentElement.clientWidth : window.innerWidth;
          var textWidth = el.scrollWidth;
          var margin = parentWidth * 0.1; // 10% margin before warning

          if (textWidth > parentWidth - margin && !hasOverflowHidden) {
            issues.push({
              selector: generateSelector(el),
              type: 'nowrap-risk',
              severity: 'warning',
              message: 'white-space: nowrap may cause overflow with longer content',
              details: {
                textWidth: textWidth,
                containerWidth: parentWidth,
                whiteSpace: computed.whiteSpace
              },
              fix: 'Consider allowing text wrap or using overflow handling'
            });
          }
        }

        // Check for viewport unit fonts that may become unreadable
        var fontSize = computed.fontSize;
        var fontSizeValue = parseFloat(fontSize);

        // Check if original font-size uses viewport units (check inline or stylesheet)
        var inlineFontSize = el.style.fontSize || '';
        if (inlineFontSize.match(/v[wh]|vmin|vmax/)) {
          // Calculate what size this would be at 320px viewport
          var vwMatch = inlineFontSize.match(/([\d.]+)vw/);
          var vhMatch = inlineFontSize.match(/([\d.]+)vh/);
          var minSize = fontSizeValue;

          if (vwMatch) {
            minSize = (parseFloat(vwMatch[1]) / 100) * 320;
          } else if (vhMatch) {
            minSize = (parseFloat(vhMatch[1]) / 100) * 568; // iPhone SE height
          }

          if (minSize < 12) {
            issues.push({
              selector: generateSelector(el),
              type: 'viewport-font',
              severity: 'warning',
              message: 'Font uses viewport units - may become unreadable at small sizes',
              details: {
                fontSize: inlineFontSize,
                computedSize: fontSize,
                estimatedMinSize: minSize.toFixed(1) + 'px',
                minRecommended: '12px'
              },
              wcag: '1.4.4',
              fix: 'Use clamp() with minimum pixel size: clamp(14px, ' + inlineFontSize + ', 48px)'
            });
          }
        }

        // Check for fixed height containers with text that might overflow
        var heightValue = computed.height;
        var hasFixedHeight = heightValue !== 'auto' &&
                            !heightValue.match(/%|vh|vmin|vmax/) &&
                            computed.minHeight === '0px';

        if (hasFixedHeight && isTruncatedV && el.textContent.trim().length > 50) {
          issues.push({
            selector: generateSelector(el),
            type: 'fixed-height-text',
            severity: 'warning',
            message: 'Fixed height container may clip variable-length text',
            details: {
              height: heightValue,
              scrollHeight: el.scrollHeight,
              clientHeight: el.clientHeight,
              overflowY: computed.overflowY
            },
            fix: 'Use min-height instead of height, or ensure overflow is visible'
          });
        }

        // Check for small line-height that may cause text overlap
        var lineHeight = parseFloat(computed.lineHeight);
        if (!isNaN(lineHeight) && fontSizeValue > 0) {
          var lineHeightRatio = lineHeight / fontSizeValue;
          if (lineHeightRatio < 1.1 && el.textContent.trim().length > 20) {
            issues.push({
              selector: generateSelector(el),
              type: 'tight-line-height',
              severity: 'info',
              message: 'Very tight line-height may cause text overlap with descenders/ascenders',
              details: {
                lineHeight: computed.lineHeight,
                fontSize: fontSize,
                ratio: lineHeightRatio.toFixed(2)
              },
              wcag: '1.4.12',
              fix: 'Use line-height of at least 1.5 for body text'
            });
          }
        }
      }

      // Categorize issues
      var summary = {
        truncations: issues.filter(function(i) { return i.type === 'truncation-ellipsis'; }).length,
        overflows: issues.filter(function(i) { return i.type === 'overflow-clipped'; }).length,
        nowrapRisks: issues.filter(function(i) { return i.type === 'nowrap-risk'; }).length,
        viewportFonts: issues.filter(function(i) { return i.type === 'viewport-font'; }).length,
        fixedHeights: issues.filter(function(i) { return i.type === 'fixed-height-text'; }).length,
        lineHeightIssues: issues.filter(function(i) { return i.type === 'tight-line-height'; }).length,
        errors: issues.filter(function(i) { return i.severity === 'error'; }).length,
        warnings: issues.filter(function(i) { return i.severity === 'warning'; }).length,
        total: issues.length
      };

      return {
        issues: issues,
        summary: summary,
        timestamp: Date.now()
      };
    },

    // Check for responsive layout risks
    checkResponsiveRisk: function(selector) {
      var root = selector ? resolveElement(selector) : document.body;
      if (!root) return { error: 'Element not found' };

      var issues = [];
      var elements = root.querySelectorAll('*');
      var viewportWidth = window.innerWidth;

      for (var i = 0; i < elements.length; i++) {
        var el = elements[i];
        var computed = window.getComputedStyle(el);

        // Skip non-visible elements
        if (computed.display === 'none') continue;

        var rect = el.getBoundingClientRect();
        if (rect.width === 0 || rect.height === 0) continue;

        // Check for fixed pixel widths in fluid containers
        var width = computed.width;
        var maxWidth = computed.maxWidth;
        var parentEl = el.parentElement;
        var parentComputed = parentEl ? window.getComputedStyle(parentEl) : null;

        var hasFixedWidth = width.match(/^\d+(\.\d+)?px$/) && parseFloat(width) > 100;
        var hasNoMaxWidth = maxWidth === 'none' || maxWidth === '';
        var parentIsFluid = parentComputed &&
                          (parentComputed.width === 'auto' ||
                           parentComputed.width.match(/%/) ||
                           parentComputed.display.indexOf('flex') !== -1 ||
                           parentComputed.display.indexOf('grid') !== -1);

        if (hasFixedWidth && hasNoMaxWidth && parentIsFluid) {
          var fixedWidthPx = parseFloat(width);
          var breaksAt = fixedWidthPx + 32; // Account for padding/margins

          issues.push({
            selector: generateSelector(el),
            type: 'fixed-width-in-fluid',
            severity: 'warning',
            message: 'Fixed pixel width in flexible container',
            details: {
              elementWidth: width,
              parentDisplay: parentComputed.display,
              breaksAtViewport: breaksAt + 'px'
            },
            breakpoints: {
              '320px': fixedWidthPx > 288,
              '375px': fixedWidthPx > 343,
              '768px': fixedWidthPx > 736
            },
            fix: 'Use max-width: 100% or responsive units (%, vw, clamp())'
          });
        }

        // Check for images without max-width constraint
        if (el.tagName === 'IMG') {
          var imgMaxWidth = computed.maxWidth;
          var naturalWidth = el.naturalWidth || 0;

          if ((imgMaxWidth === 'none' || imgMaxWidth === '') && naturalWidth > viewportWidth * 0.5) {
            issues.push({
              selector: generateSelector(el),
              type: 'unbounded-image',
              severity: 'warning',
              message: 'Image without max-width constraint may overflow on small screens',
              details: {
                naturalWidth: naturalWidth,
                displayWidth: rect.width,
                maxWidth: imgMaxWidth
              },
              fix: 'Add max-width: 100% to image or its container'
            });
          }
        }

        // Check for flex containers with nowrap that might overflow
        if (computed.display.indexOf('flex') !== -1) {
          var flexWrap = computed.flexWrap;
          var childCount = el.children.length;

          if (flexWrap === 'nowrap' && childCount > 2) {
            var childrenWidth = 0;
            for (var j = 0; j < el.children.length; j++) {
              childrenWidth += el.children[j].getBoundingClientRect().width;
            }

            if (childrenWidth > rect.width * 0.9) {
              issues.push({
                selector: generateSelector(el),
                type: 'flex-nowrap-overflow',
                severity: 'warning',
                message: 'Flex container with nowrap may overflow on smaller screens',
                details: {
                  flexWrap: flexWrap,
                  childCount: childCount,
                  childrenTotalWidth: Math.round(childrenWidth),
                  containerWidth: Math.round(rect.width),
                  fillRatio: (childrenWidth / rect.width * 100).toFixed(1) + '%'
                },
                fix: 'Consider flex-wrap: wrap or responsive breakpoints'
              });
            }
          }
        }

        // Check for grid with fixed column widths
        if (computed.display.indexOf('grid') !== -1) {
          var gridCols = computed.gridTemplateColumns;

          if (gridCols && gridCols.match(/\d+px/) && !gridCols.match(/minmax|auto-fit|auto-fill/)) {
            var colWidths = gridCols.split(' ').map(function(w) { return parseFloat(w) || 0; });
            var totalColWidth = colWidths.reduce(function(a, b) { return a + b; }, 0);

            if (totalColWidth > 320) {
              issues.push({
                selector: generateSelector(el),
                type: 'grid-fixed-columns',
                severity: 'warning',
                message: 'Grid with fixed pixel columns may not fit small screens',
                details: {
                  gridTemplateColumns: gridCols,
                  totalWidth: totalColWidth,
                  containerWidth: Math.round(rect.width)
                },
                fix: 'Use minmax(), auto-fit, or auto-fill for responsive grids'
              });
            }
          }
        }

        // Check for elements extending beyond viewport
        if (rect.right > viewportWidth + 10) {
          issues.push({
            selector: generateSelector(el),
            type: 'viewport-overflow',
            severity: 'error',
            message: 'Element extends beyond viewport causing horizontal scroll',
            details: {
              elementRight: Math.round(rect.right),
              viewportWidth: viewportWidth,
              overflow: Math.round(rect.right - viewportWidth)
            },
            fix: 'Constrain element width or use overflow handling'
          });
        }

        // Check for horizontal scroll on containers
        if (el.scrollWidth > el.clientWidth + 5 && computed.overflowX !== 'hidden') {
          var isIntentionalScroll = computed.overflowX === 'scroll' || computed.overflowX === 'auto';
          if (!isIntentionalScroll || el === document.body || el === document.documentElement) {
            issues.push({
              selector: generateSelector(el),
              type: 'horizontal-scroll',
              severity: el === document.body ? 'error' : 'info',
              message: 'Element has horizontal scrollable content',
              details: {
                scrollWidth: el.scrollWidth,
                clientWidth: el.clientWidth,
                overflowX: computed.overflowX
              },
              fix: 'Review content width and container constraints'
            });
          }
        }
      }

      // Test specific breakpoints by checking which issues would trigger
      var breakpointAnalysis = {
        '320px': { issues: [], willOverflow: false },
        '375px': { issues: [], willOverflow: false },
        '768px': { issues: [], willOverflow: false },
        '1024px': { issues: [], willOverflow: false }
      };

      issues.forEach(function(issue) {
        if (issue.breakpoints) {
          Object.keys(issue.breakpoints).forEach(function(bp) {
            if (issue.breakpoints[bp] && breakpointAnalysis[bp]) {
              breakpointAnalysis[bp].issues.push(issue.selector);
              breakpointAnalysis[bp].willOverflow = true;
            }
          });
        }
        if (issue.type === 'viewport-overflow') {
          breakpointAnalysis['320px'].willOverflow = true;
          breakpointAnalysis['375px'].willOverflow = true;
        }
      });

      var summary = {
        fixedWidthIssues: issues.filter(function(i) { return i.type === 'fixed-width-in-fluid'; }).length,
        unboundedImages: issues.filter(function(i) { return i.type === 'unbounded-image'; }).length,
        flexWrapRisks: issues.filter(function(i) { return i.type === 'flex-nowrap-overflow'; }).length,
        gridIssues: issues.filter(function(i) { return i.type === 'grid-fixed-columns'; }).length,
        viewportOverflows: issues.filter(function(i) { return i.type === 'viewport-overflow'; }).length,
        horizontalScrolls: issues.filter(function(i) { return i.type === 'horizontal-scroll'; }).length,
        errors: issues.filter(function(i) { return i.severity === 'error'; }).length,
        warnings: issues.filter(function(i) { return i.severity === 'warning'; }).length,
        total: issues.length
      };

      return {
        issues: issues,
        breakpoints: breakpointAnalysis,
        summary: summary,
        currentViewport: viewportWidth,
        timestamp: Date.now()
      };
    },

    // Capture comprehensive performance metrics
    capturePerformanceMetrics: function() {
      var result = {
        cls: null,
        longTasks: [],
        resources: {
          byType: {},
          largest: [],
          slowest: [],
          renderBlocking: []
        },
        totals: {
          pageWeight: 0,
          resourceCount: 0,
          loadTime: 0,
          domContentLoaded: 0
        },
        timestamp: Date.now()
      };

      try {
        var perf = window.performance;
        if (!perf) return { error: 'Performance API not available' };

        // Navigation timing
        if (perf.timing) {
          var timing = perf.timing;
          result.totals.loadTime = timing.loadEventEnd - timing.navigationStart;
          result.totals.domContentLoaded = timing.domContentLoadedEventEnd - timing.navigationStart;
        }

        // Try to get navigation timing from newer API
        if (perf.getEntriesByType) {
          var navEntries = perf.getEntriesByType('navigation');
          if (navEntries && navEntries.length > 0) {
            var nav = navEntries[0];
            result.totals.loadTime = Math.round(nav.loadEventEnd);
            result.totals.domContentLoaded = Math.round(nav.domContentLoadedEventEnd);
          }
        }

        // Resource timing
        if (perf.getEntriesByType) {
          var resources = perf.getEntriesByType('resource');
          var byType = {};
          var allResources = [];

          resources.forEach(function(r) {
            var type = r.initiatorType || 'other';
            var size = r.transferSize || r.encodedBodySize || 0;
            var duration = r.duration || 0;

            // Aggregate by type
            if (!byType[type]) {
              byType[type] = { count: 0, totalSize: 0, totalDuration: 0 };
            }
            byType[type].count++;
            byType[type].totalSize += size;
            byType[type].totalDuration += duration;

            result.totals.pageWeight += size;
            result.totals.resourceCount++;

            allResources.push({
              url: r.name,
              type: type,
              size: size,
              duration: Math.round(duration),
              startTime: Math.round(r.startTime)
            });

            // Check for render-blocking (loaded before DOMContentLoaded)
            if ((type === 'script' || type === 'link' || type === 'css') &&
                r.startTime < result.totals.domContentLoaded &&
                r.renderBlockingStatus === 'blocking') {
              result.resources.renderBlocking.push({
                url: r.name,
                type: type,
                size: size,
                duration: Math.round(duration)
              });
            }
          });

          result.resources.byType = byType;

          // Sort for largest and slowest
          allResources.sort(function(a, b) { return b.size - a.size; });
          result.resources.largest = allResources.slice(0, 10);

          allResources.sort(function(a, b) { return b.duration - a.duration; });
          result.resources.slowest = allResources.slice(0, 10);
        }

        // Layout shift (CLS) - check if we have any stored
        if (perf.getEntriesByType) {
          var layoutShifts = perf.getEntriesByType('layout-shift');
          if (layoutShifts && layoutShifts.length > 0) {
            var clsValue = 0;
            var clsEntries = [];

            layoutShifts.forEach(function(entry) {
              if (!entry.hadRecentInput) {
                clsValue += entry.value;
                clsEntries.push({
                  value: entry.value,
                  startTime: Math.round(entry.startTime),
                  sources: entry.sources ? entry.sources.map(function(s) {
                    return s.node ? generateSelector(s.node) : 'unknown';
                  }) : []
                });
              }
            });

            var clsRating = 'good';
            if (clsValue > 0.25) clsRating = 'poor';
            else if (clsValue > 0.1) clsRating = 'needs-improvement';

            result.cls = {
              score: parseFloat(clsValue.toFixed(4)),
              rating: clsRating,
              shifts: clsEntries.slice(0, 10)
            };
          }
        }

        // Long tasks (if available)
        if (perf.getEntriesByType) {
          var longTasks = perf.getEntriesByType('longtask');
          if (longTasks && longTasks.length > 0) {
            result.longTasks = longTasks.slice(0, 20).map(function(task) {
              return {
                duration: Math.round(task.duration),
                startTime: Math.round(task.startTime),
                name: task.name
              };
            });
          }
        }

        // Paint timing
        if (perf.getEntriesByType) {
          var paintEntries = perf.getEntriesByType('paint');
          result.paint = {};
          paintEntries.forEach(function(entry) {
            if (entry.name === 'first-paint') {
              result.paint.firstPaint = Math.round(entry.startTime);
            } else if (entry.name === 'first-contentful-paint') {
              result.paint.firstContentfulPaint = Math.round(entry.startTime);
            }
          });
        }

      } catch (e) {
        return { error: e.message };
      }

      return result;
    },

    // Load and run axe-core accessibility audit
    runAxeAudit: function(options) {
      options = options || {};
      var self = this;

      return new Promise(function(resolve, reject) {
        // Check if axe is already loaded
        if (window.axe) {
          self._executeAxeAudit(options, resolve, reject);
          return;
        }

        // Load axe-core from CDN
        var script = document.createElement('script');
        script.src = 'https://cdn.jsdelivr.net/npm/axe-core@4.10.0/axe.min.js';
        script.onload = function() {
          console.log('[DevTool] axe-core loaded successfully');
          self._executeAxeAudit(options, resolve, reject);
        };
        script.onerror = function() {
          reject({ error: 'Failed to load axe-core from CDN' });
        };
        document.head.appendChild(script);
      });
    },

    _executeAxeAudit: function(options, resolve, reject) {
      if (!window.axe) {
        reject({ error: 'axe-core not available' });
        return;
      }

      var axeOptions = {
        runOnly: options.runOnly || ['wcag2a', 'wcag2aa', 'best-practice'],
        resultTypes: ['violations', 'incomplete']
      };

      var context = options.selector ? { include: [options.selector] } : document;

      window.axe.run(context, axeOptions).then(function(results) {
        // Process violations
        var violations = results.violations.map(function(v) {
          return {
            id: v.id,
            impact: v.impact,
            description: v.description,
            help: v.help,
            helpUrl: v.helpUrl,
            tags: v.tags,
            nodes: v.nodes.slice(0, 10).map(function(n) {
              return {
                selector: n.target.join(' '),
                html: n.html.substring(0, 200),
                failureSummary: n.failureSummary,
                impact: n.impact
              };
            })
          };
        });

        // Count by impact
        var impactCounts = { critical: 0, serious: 0, moderate: 0, minor: 0 };
        violations.forEach(function(v) {
          if (impactCounts.hasOwnProperty(v.impact)) {
            impactCounts[v.impact]++;
          }
        });

        // Calculate score (rough approximation)
        var totalIssues = violations.reduce(function(sum, v) { return sum + v.nodes.length; }, 0);
        var weightedScore = impactCounts.critical * 25 +
                          impactCounts.serious * 15 +
                          impactCounts.moderate * 8 +
                          impactCounts.minor * 3;
        var score = Math.max(0, 100 - weightedScore);

        resolve({
          violations: violations,
          passes: results.passes ? results.passes.length : 0,
          incomplete: results.incomplete ? results.incomplete.length : 0,
          inapplicable: results.inapplicable ? results.inapplicable.length : 0,
          summary: {
            critical: impactCounts.critical,
            serious: impactCounts.serious,
            moderate: impactCounts.moderate,
            minor: impactCounts.minor,
            totalViolations: violations.length,
            totalNodes: totalIssues
          },
          score: score,
          testEngine: {
            name: 'axe-core',
            version: window.axe.version
          },
          timestamp: Date.now()
        });
      }).catch(function(err) {
        reject({ error: err.message || 'axe-core audit failed' });
      });
    },

    // Comprehensive layout robustness audit
    auditLayoutRobustness: function(options) {
      options = options || {};
      var self = this;
      var selector = options.selector || null;

      return new Promise(function(resolve, reject) {
        try {
          // Gather all synchronous audits
          var textFragility = self.checkTextFragility(selector);
          var responsiveRisk = self.checkResponsiveRisk(selector);
          var layoutIssues = self.diagnoseLayout(selector);
          var performance = self.capturePerformanceMetrics();
          var basicA11y = self.auditAccessibility();

          // Calculate overall scores
          var textScore = Math.max(0, 100 - (textFragility.summary.errors * 15) - (textFragility.summary.warnings * 5));
          var responsiveScore = Math.max(0, 100 - (responsiveRisk.summary.errors * 15) - (responsiveRisk.summary.warnings * 5));
          var layoutScore = Math.max(0, 100 - (layoutIssues.overflows.count * 5) - (layoutIssues.offscreen.count * 3));
          var a11yScore = basicA11y.score || 100;

          // Performance score based on key metrics
          var perfScore = 100;
          if (performance.cls && performance.cls.score > 0.25) perfScore -= 30;
          else if (performance.cls && performance.cls.score > 0.1) perfScore -= 15;
          if (performance.totals.loadTime > 5000) perfScore -= 20;
          else if (performance.totals.loadTime > 3000) perfScore -= 10;
          if (performance.longTasks.length > 5) perfScore -= 15;
          else if (performance.longTasks.length > 2) perfScore -= 5;
          perfScore = Math.max(0, perfScore);

          var overallScore = Math.round(
            (textScore * 0.2) +
            (responsiveScore * 0.25) +
            (layoutScore * 0.15) +
            (a11yScore * 0.25) +
            (perfScore * 0.15)
          );

          // Determine grade
          var grade = 'F';
          if (overallScore >= 90) grade = 'A';
          else if (overallScore >= 80) grade = 'B';
          else if (overallScore >= 70) grade = 'C';
          else if (overallScore >= 60) grade = 'D';

          // Collect critical issues
          var criticalIssues = [];

          textFragility.issues.filter(function(i) { return i.severity === 'error'; }).forEach(function(i) {
            criticalIssues.push({
              category: 'text',
              type: i.type,
              selector: i.selector,
              message: i.message,
              fix: i.fix
            });
          });

          responsiveRisk.issues.filter(function(i) { return i.severity === 'error'; }).forEach(function(i) {
            criticalIssues.push({
              category: 'responsive',
              type: i.type,
              selector: i.selector,
              message: i.message,
              fix: i.fix
            });
          });

          // Generate recommendations
          var recommendations = [];
          var priority = 1;

          if (textFragility.summary.truncations > 0) {
            recommendations.push({
              priority: priority++,
              category: 'text',
              issue: textFragility.summary.truncations + ' element(s) have truncated text with ellipsis',
              impact: 'Content loss - users cannot see full text (WCAG 1.4.10)',
              fix: 'Remove text-overflow: ellipsis or allow container to expand'
            });
          }

          if (responsiveRisk.summary.viewportOverflows > 0) {
            recommendations.push({
              priority: priority++,
              category: 'responsive',
              issue: responsiveRisk.summary.viewportOverflows + ' element(s) cause horizontal scroll',
              impact: 'Poor mobile experience, content may be inaccessible',
              fix: 'Constrain element widths with max-width: 100%'
            });
          }

          if (performance.cls && performance.cls.rating === 'poor') {
            recommendations.push({
              priority: priority++,
              category: 'performance',
              issue: 'High Cumulative Layout Shift (CLS: ' + performance.cls.score + ')',
              impact: 'Visual instability, poor Core Web Vitals score',
              fix: 'Reserve space for dynamic content, set explicit dimensions on images'
            });
          }

          if (responsiveRisk.summary.unboundedImages > 0) {
            recommendations.push({
              priority: priority++,
              category: 'responsive',
              issue: responsiveRisk.summary.unboundedImages + ' image(s) without max-width constraint',
              impact: 'Images may overflow containers on small screens',
              fix: 'Add max-width: 100% to images'
            });
          }

          if (basicA11y.summary && basicA11y.summary.errors > 0) {
            recommendations.push({
              priority: priority++,
              category: 'accessibility',
              issue: basicA11y.summary.errors + ' accessibility error(s) found',
              impact: 'Content may be inaccessible to users with disabilities',
              fix: 'Review and fix accessibility errors (run runAxeAudit for details)'
            });
          }

          var result = {
            textFragility: textFragility,
            responsiveRisk: responsiveRisk,
            layoutIssues: layoutIssues,
            accessibility: basicA11y,
            performance: performance,
            scores: {
              text: textScore,
              responsive: responsiveScore,
              layout: layoutScore,
              accessibility: a11yScore,
              performance: perfScore,
              overall: overallScore
            },
            grade: grade,
            criticalIssues: criticalIssues,
            recommendations: recommendations,
            timestamp: Date.now()
          };

          // Optionally run full axe audit
          if (options.includeAxe) {
            self.runAxeAudit({ selector: selector }).then(function(axeResults) {
              result.axeAudit = axeResults;
              result.scores.accessibility = axeResults.score;
              // Recalculate overall with axe score
              result.scores.overall = Math.round(
                (textScore * 0.2) +
                (responsiveScore * 0.25) +
                (layoutScore * 0.15) +
                (axeResults.score * 0.25) +
                (perfScore * 0.15)
              );
              if (result.scores.overall >= 90) result.grade = 'A';
              else if (result.scores.overall >= 80) result.grade = 'B';
              else if (result.scores.overall >= 70) result.grade = 'C';
              else if (result.scores.overall >= 60) result.grade = 'D';
              else result.grade = 'F';
              resolve(result);
            }).catch(function(err) {
              result.axeAudit = { error: err.error || err.message };
              resolve(result);
            });
          } else {
            resolve(result);
          }
        } catch (e) {
          reject({ error: e.message });
        }
      });
    },

    // Start observing layout shifts in real-time
    observeLayoutShifts: function(callback) {
      if (!window.PerformanceObserver) {
        return { error: 'PerformanceObserver not supported' };
      }

      var clsValue = 0;
      var entries = [];

      var observer = new PerformanceObserver(function(list) {
        for (var entry of list.getEntries()) {
          if (!entry.hadRecentInput) {
            clsValue += entry.value;
            var shiftEntry = {
              value: entry.value,
              cumulative: clsValue,
              startTime: entry.startTime,
              sources: entry.sources ? entry.sources.map(function(s) {
                return s.node ? generateSelector(s.node) : 'unknown';
              }) : []
            };
            entries.push(shiftEntry);

            if (callback) {
              callback(shiftEntry, clsValue);
            }
          }
        }
      });

      try {
        observer.observe({ type: 'layout-shift', buffered: true });
      } catch (e) {
        return { error: 'layout-shift observation not supported: ' + e.message };
      }

      return {
        stop: function() {
          observer.disconnect();
          return { finalCLS: clsValue, entries: entries };
        },
        getCurrent: function() {
          return { cls: clsValue, entryCount: entries.length };
        }
      };
    },

    // Start observing long tasks in real-time
    observeLongTasks: function(callback) {
      if (!window.PerformanceObserver) {
        return { error: 'PerformanceObserver not supported' };
      }

      var tasks = [];

      var observer = new PerformanceObserver(function(list) {
        for (var entry of list.getEntries()) {
          var taskEntry = {
            duration: entry.duration,
            startTime: entry.startTime,
            name: entry.name
          };
          tasks.push(taskEntry);

          if (callback) {
            callback(taskEntry);
          }
        }
      });

      try {
        observer.observe({ type: 'longtask', buffered: true });
      } catch (e) {
        return { error: 'longtask observation not supported: ' + e.message };
      }

      return {
        stop: function() {
          observer.disconnect();
          return { tasks: tasks, count: tasks.length };
        },
        getCurrent: function() {
          return { taskCount: tasks.length, totalBlocking: tasks.reduce(function(sum, t) { return sum + t.duration; }, 0) };
        }
      };
    },

    // ========================================================================
    // FRAME RATE & JANK DETECTION
    // ========================================================================

    // Observe frame rate and detect jank/stuttering
    observeFrameRate: function(options) {
      options = options || {};
      var duration = options.duration || 5000;
      var threshold = options.threshold || 50; // ms, frames longer than this are jank

      var frames = [];
      var jankFrames = [];
      var startTime = performance.now();
      var lastTime = startTime;
      var running = true;
      var rafId = null;

      function tick(now) {
        if (!running) return;

        var delta = now - lastTime;
        frames.push({ delta: delta, timestamp: now - startTime });

        // Detect jank: frame took longer than threshold
        if (delta > threshold) {
          var droppedFrames = Math.floor(delta / 16.67) - 1;
          jankFrames.push({
            delta: Math.round(delta),
            droppedFrames: droppedFrames,
            timestamp: Math.round(now - startTime)
          });
        }

        lastTime = now;

        if (now - startTime < duration) {
          rafId = requestAnimationFrame(tick);
        } else {
          running = false;
          calculateResults();
        }
      }

      var results = null;
      function calculateResults() {
        var totalFrames = frames.length;
        var totalTime = frames.length > 0 ? frames[frames.length - 1].timestamp : 0;
        var avgFPS = totalFrames > 0 ? (totalFrames / (totalTime / 1000)) : 0;

        var deltas = frames.map(function(f) { return f.delta; });
        var avgDelta = deltas.reduce(function(a, b) { return a + b; }, 0) / deltas.length;
        var maxDelta = Math.max.apply(null, deltas);
        var minDelta = Math.min.apply(null, deltas);

        // Calculate percentiles
        deltas.sort(function(a, b) { return a - b; });
        var p95 = deltas[Math.floor(deltas.length * 0.95)] || 0;
        var p99 = deltas[Math.floor(deltas.length * 0.99)] || 0;

        var smoothFrames = frames.filter(function(f) { return f.delta <= 16.67; }).length;
        var smoothness = totalFrames > 0 ? (smoothFrames / totalFrames * 100) : 0;

        results = {
          totalFrames: totalFrames,
          duration: Math.round(totalTime),
          avgFPS: parseFloat(avgFPS.toFixed(1)),
          avgFrameTime: parseFloat(avgDelta.toFixed(2)),
          maxFrameTime: parseFloat(maxDelta.toFixed(2)),
          minFrameTime: parseFloat(minDelta.toFixed(2)),
          p95FrameTime: parseFloat(p95.toFixed(2)),
          p99FrameTime: parseFloat(p99.toFixed(2)),
          jankFrames: jankFrames.length,
          totalDroppedFrames: jankFrames.reduce(function(sum, j) { return sum + j.droppedFrames; }, 0),
          smoothness: parseFloat(smoothness.toFixed(1)),
          rating: smoothness >= 90 ? 'smooth' : smoothness >= 70 ? 'moderate-jank' : 'janky',
          jankEvents: jankFrames.slice(0, 20),
          timestamp: Date.now()
        };
      }

      rafId = requestAnimationFrame(tick);

      return {
        stop: function() {
          running = false;
          if (rafId) cancelAnimationFrame(rafId);
          if (!results) calculateResults();
          return results;
        },
        isRunning: function() {
          return running;
        },
        getResults: function() {
          return results;
        }
      };
    },

    // Observe Long Animation Frames (LoAF) - Chrome 123+
    observeLongAnimationFrames: function(callback) {
      if (!window.PerformanceObserver) {
        return { error: 'PerformanceObserver not supported' };
      }

      var entries = [];

      var observer = new PerformanceObserver(function(list) {
        list.getEntries().forEach(function(entry) {
          var loafEntry = {
            duration: Math.round(entry.duration),
            blockingDuration: Math.round(entry.blockingDuration || 0),
            startTime: Math.round(entry.startTime),
            renderStart: Math.round(entry.renderStart || 0),
            styleAndLayoutStart: Math.round(entry.styleAndLayoutStart || 0),
            firstUIEventTimestamp: Math.round(entry.firstUIEventTimestamp || 0),
            scripts: []
          };

          // Extract script attribution if available
          if (entry.scripts && entry.scripts.length > 0) {
            loafEntry.scripts = entry.scripts.slice(0, 10).map(function(s) {
              return {
                sourceURL: s.sourceURL || '',
                sourceFunctionName: s.sourceFunctionName || '',
                invoker: s.invoker || '',
                invokerType: s.invokerType || '',
                duration: Math.round(s.duration || 0),
                executionStart: Math.round(s.executionStart || 0),
                forcedStyleAndLayoutDuration: Math.round(s.forcedStyleAndLayoutDuration || 0),
                pauseDuration: Math.round(s.pauseDuration || 0)
              };
            });
          }

          entries.push(loafEntry);

          if (callback) {
            callback(loafEntry);
          }
        });
      });

      try {
        observer.observe({ type: 'long-animation-frame', buffered: true });
      } catch (e) {
        return { error: 'long-animation-frame observation not supported: ' + e.message };
      }

      return {
        stop: function() {
          observer.disconnect();
          var totalBlocking = entries.reduce(function(sum, e) { return sum + e.blockingDuration; }, 0);
          var forcedLayouts = entries.reduce(function(sum, e) {
            return sum + e.scripts.reduce(function(s, script) {
              return s + script.forcedStyleAndLayoutDuration;
            }, 0);
          }, 0);

          return {
            entries: entries,
            count: entries.length,
            totalBlockingDuration: totalBlocking,
            totalForcedLayoutDuration: forcedLayouts,
            worstFrame: entries.length > 0 ? entries.reduce(function(worst, e) {
              return e.duration > worst.duration ? e : worst;
            }) : null
          };
        },
        getCurrent: function() {
          return {
            count: entries.length,
            totalBlocking: entries.reduce(function(sum, e) { return sum + e.blockingDuration; }, 0)
          };
        }
      };
    },

    // ========================================================================
    // CORE WEB VITALS OBSERVERS
    // ========================================================================

    // Observe Interaction to Next Paint (INP) - Core Web Vital since March 2024
    observeINP: function(callback) {
      if (!window.PerformanceObserver) {
        return { error: 'PerformanceObserver not supported' };
      }

      var interactions = [];
      var worstINP = 0;
      var worstInteraction = null;

      var observer = new PerformanceObserver(function(list) {
        list.getEntries().forEach(function(entry) {
          // Only track entries with interactionId (actual user interactions)
          if (!entry.interactionId) return;

          var interaction = {
            type: entry.name,
            duration: Math.round(entry.duration),
            startTime: Math.round(entry.startTime),
            processingStart: Math.round(entry.processingStart || 0),
            processingEnd: Math.round(entry.processingEnd || 0),
            interactionId: entry.interactionId,
            target: entry.target ? generateSelector(entry.target) : null,
            inputDelay: Math.round((entry.processingStart || entry.startTime) - entry.startTime),
            processingTime: Math.round((entry.processingEnd || 0) - (entry.processingStart || 0)),
            presentationDelay: Math.round(entry.duration - ((entry.processingEnd || 0) - entry.startTime))
          };

          interactions.push(interaction);

          if (interaction.duration > worstINP) {
            worstINP = interaction.duration;
            worstInteraction = interaction;

            if (callback) {
              callback({
                type: 'new-worst',
                inp: worstINP,
                interaction: worstInteraction,
                rating: worstINP < 200 ? 'good' : worstINP < 500 ? 'needs-improvement' : 'poor'
              });
            }
          }
        });
      });

      try {
        observer.observe({ type: 'event', buffered: true, durationThreshold: 16 });
      } catch (e) {
        return { error: 'event observation not supported: ' + e.message };
      }

      return {
        stop: function() {
          observer.disconnect();

          // Calculate p75 INP (what Google uses)
          var durations = interactions.map(function(i) { return i.duration; });
          durations.sort(function(a, b) { return a - b; });
          var p75 = durations.length > 0 ? durations[Math.floor(durations.length * 0.75)] : 0;

          return {
            interactions: interactions.slice(-50), // Last 50 interactions
            totalInteractions: interactions.length,
            worstINP: worstINP,
            worstInteraction: worstInteraction,
            p75INP: p75,
            rating: p75 < 200 ? 'good' : p75 < 500 ? 'needs-improvement' : 'poor',
            breakdown: {
              good: interactions.filter(function(i) { return i.duration < 200; }).length,
              needsImprovement: interactions.filter(function(i) { return i.duration >= 200 && i.duration < 500; }).length,
              poor: interactions.filter(function(i) { return i.duration >= 500; }).length
            }
          };
        },
        getCurrent: function() {
          return {
            worstINP: worstINP,
            interactionCount: interactions.length,
            rating: worstINP < 200 ? 'good' : worstINP < 500 ? 'needs-improvement' : 'poor'
          };
        }
      };
    },

    // Observe Largest Contentful Paint (LCP) - Core Web Vital
    observeLCP: function(callback) {
      if (!window.PerformanceObserver) {
        return { error: 'PerformanceObserver not supported' };
      }

      var lcpEntries = [];
      var currentLCP = null;

      var observer = new PerformanceObserver(function(list) {
        var entries = list.getEntries();
        entries.forEach(function(entry) {
          var lcpEntry = {
            value: Math.round(entry.startTime),
            size: entry.size,
            element: entry.element ? generateSelector(entry.element) : null,
            elementTag: entry.element ? entry.element.tagName.toLowerCase() : null,
            url: entry.url || null,
            id: entry.id || null,
            loadTime: Math.round(entry.loadTime || 0),
            renderTime: Math.round(entry.renderTime || 0),
            rating: entry.startTime < 2500 ? 'good' : entry.startTime < 4000 ? 'needs-improvement' : 'poor'
          };

          lcpEntries.push(lcpEntry);
          currentLCP = lcpEntry;

          if (callback) {
            callback(lcpEntry);
          }
        });
      });

      try {
        observer.observe({ type: 'largest-contentful-paint', buffered: true });
      } catch (e) {
        return { error: 'largest-contentful-paint observation not supported: ' + e.message };
      }

      return {
        stop: function() {
          observer.disconnect();
          return {
            finalLCP: currentLCP,
            allCandidates: lcpEntries,
            candidateCount: lcpEntries.length
          };
        },
        getCurrent: function() {
          return currentLCP;
        }
      };
    },

    // ========================================================================
    // DOM & MEMORY AUDITING
    // ========================================================================

    // Audit DOM complexity and detect bloat
    auditDOMComplexity: function() {
      var allNodes = document.querySelectorAll('*');
      var totalNodes = allNodes.length;
      var maxDepth = 0;
      var maxChildren = 0;
      var maxChildrenElement = null;
      var deepestElement = null;
      var depthDistribution = {};
      var tagCounts = {};

      for (var i = 0; i < allNodes.length; i++) {
        var node = allNodes[i];

        // Count tags
        var tag = node.tagName.toLowerCase();
        tagCounts[tag] = (tagCounts[tag] || 0) + 1;

        // Calculate depth
        var depth = 0;
        var parent = node;
        while (parent.parentElement) {
          depth++;
          parent = parent.parentElement;
        }
        depthDistribution[depth] = (depthDistribution[depth] || 0) + 1;

        if (depth > maxDepth) {
          maxDepth = depth;
          deepestElement = node;
        }

        // Check children count
        if (node.children.length > maxChildren) {
          maxChildren = node.children.length;
          maxChildrenElement = node;
        }
      }

      // Find most common tags
      var sortedTags = Object.keys(tagCounts).sort(function(a, b) {
        return tagCounts[b] - tagCounts[a];
      }).slice(0, 10).map(function(tag) {
        return { tag: tag, count: tagCounts[tag] };
      });

      // Find elements with excessive children (>60)
      var heavyParents = [];
      for (var j = 0; j < allNodes.length; j++) {
        if (allNodes[j].children.length > 60) {
          heavyParents.push({
            selector: generateSelector(allNodes[j]),
            childCount: allNodes[j].children.length
          });
        }
      }

      // Calculate scores
      var nodeScore = totalNodes < 800 ? 100 : totalNodes < 1500 ? 70 : totalNodes < 3000 ? 40 : 20;
      var depthScore = maxDepth < 15 ? 100 : maxDepth < 32 ? 70 : maxDepth < 50 ? 40 : 20;
      var childrenScore = maxChildren < 30 ? 100 : maxChildren < 60 ? 70 : maxChildren < 100 ? 40 : 20;
      var overallScore = Math.round((nodeScore + depthScore + childrenScore) / 3);

      return {
        totalNodes: totalNodes,
        maxDepth: maxDepth,
        maxChildren: maxChildren,
        deepestElement: deepestElement ? generateSelector(deepestElement) : null,
        largestParent: maxChildrenElement ? generateSelector(maxChildrenElement) : null,
        heavyParents: heavyParents.slice(0, 10),
        topTags: sortedTags,
        depthDistribution: depthDistribution,
        thresholds: {
          nodes: { value: totalNodes, limit: 1500, exceeded: totalNodes > 1500 },
          depth: { value: maxDepth, limit: 32, exceeded: maxDepth > 32 },
          children: { value: maxChildren, limit: 60, exceeded: maxChildren > 60 }
        },
        scores: {
          nodes: nodeScore,
          depth: depthScore,
          children: childrenScore,
          overall: overallScore
        },
        rating: overallScore >= 70 ? 'good' : overallScore >= 40 ? 'needs-improvement' : 'poor',
        recommendations: (function() {
          var recs = [];
          if (totalNodes > 1500) {
            recs.push('Reduce DOM nodes (current: ' + totalNodes + ', recommended: <1500). Consider virtualization for lists.');
          }
          if (maxDepth > 32) {
            recs.push('Flatten DOM structure (current depth: ' + maxDepth + ', recommended: <32).');
          }
          if (maxChildren > 60) {
            recs.push('Break up large parent elements (max children: ' + maxChildren + ', recommended: <60).');
          }
          if (heavyParents.length > 0) {
            recs.push(heavyParents.length + ' element(s) have >60 children. Consider pagination or virtualization.');
          }
          return recs;
        })(),
        timestamp: Date.now()
      };
    },

    // Capture memory metrics (Chrome only)
    captureMemoryMetrics: function() {
      var result = {
        available: false,
        timestamp: Date.now()
      };

      // Legacy API (Chrome only, approximate)
      if (performance.memory) {
        result.available = true;
        result.jsHeap = {
          usedSize: performance.memory.usedJSHeapSize,
          totalSize: performance.memory.totalJSHeapSize,
          sizeLimit: performance.memory.jsHeapSizeLimit,
          usedMB: parseFloat((performance.memory.usedJSHeapSize / 1024 / 1024).toFixed(2)),
          totalMB: parseFloat((performance.memory.totalJSHeapSize / 1024 / 1024).toFixed(2)),
          limitMB: parseFloat((performance.memory.jsHeapSizeLimit / 1024 / 1024).toFixed(2)),
          percentUsed: parseFloat(((performance.memory.usedJSHeapSize / performance.memory.jsHeapSizeLimit) * 100).toFixed(1)),
          percentAllocated: parseFloat(((performance.memory.totalJSHeapSize / performance.memory.jsHeapSizeLimit) * 100).toFixed(1))
        };

        // Memory pressure assessment
        var percentUsed = result.jsHeap.percentUsed;
        result.pressure = percentUsed < 50 ? 'low' : percentUsed < 75 ? 'moderate' : percentUsed < 90 ? 'high' : 'critical';
      }

      // Check for modern API availability
      result.measureMemoryAvailable = typeof performance.measureUserAgentSpecificMemory === 'function';
      if (result.measureMemoryAvailable) {
        result.measureMemoryNote = 'Call measureMemoryDetailed() for cross-origin isolated measurement (requires COOP/COEP headers)';
      }

      return result;
    },

    // Detailed memory measurement (async, requires cross-origin isolation)
    measureMemoryDetailed: function() {
      if (!performance.measureUserAgentSpecificMemory) {
        return Promise.resolve({ error: 'measureUserAgentSpecificMemory not available. Requires cross-origin isolation (COOP/COEP headers).' });
      }

      return performance.measureUserAgentSpecificMemory().then(function(result) {
        var totalBytes = result.bytes;
        var breakdown = result.breakdown.map(function(entry) {
          return {
            bytes: entry.bytes,
            types: entry.types,
            attribution: entry.attribution.map(function(attr) {
              return {
                url: attr.url,
                scope: attr.scope
              };
            })
          };
        });

        return {
          totalBytes: totalBytes,
          totalMB: parseFloat((totalBytes / 1024 / 1024).toFixed(2)),
          breakdown: breakdown,
          timestamp: Date.now()
        };
      }).catch(function(err) {
        return { error: err.message };
      });
    },

    // Audit event listeners for potential leaks
    auditEventListeners: function() {
      var allElements = document.querySelectorAll('*');
      var inlineHandlerCount = 0;
      var elementsWithInlineHandlers = [];
      var eventAttributes = ['onclick', 'onchange', 'onsubmit', 'onkeydown', 'onkeyup', 'onkeypress',
                            'onmousedown', 'onmouseup', 'onmousemove', 'onmouseover', 'onmouseout',
                            'onfocus', 'onblur', 'onload', 'onerror', 'onscroll', 'onresize',
                            'oninput', 'ontouchstart', 'ontouchmove', 'ontouchend'];

      for (var i = 0; i < allElements.length; i++) {
        var el = allElements[i];
        var handlers = [];

        for (var j = 0; j < eventAttributes.length; j++) {
          if (el.hasAttribute(eventAttributes[j])) {
            handlers.push(eventAttributes[j]);
            inlineHandlerCount++;
          }
        }

        if (handlers.length > 0) {
          elementsWithInlineHandlers.push({
            selector: generateSelector(el),
            handlers: handlers,
            count: handlers.length
          });
        }
      }

      // Sort by handler count
      elementsWithInlineHandlers.sort(function(a, b) { return b.count - a.count; });

      // Check for potential issues
      var issues = [];
      if (inlineHandlerCount > 50) {
        issues.push({
          type: 'excessive-inline-handlers',
          message: 'Found ' + inlineHandlerCount + ' inline event handlers. Consider event delegation.',
          severity: 'warning'
        });
      }

      var heavyElements = elementsWithInlineHandlers.filter(function(e) { return e.count >= 3; });
      if (heavyElements.length > 0) {
        issues.push({
          type: 'handler-concentration',
          message: heavyElements.length + ' element(s) have 3+ inline handlers.',
          severity: 'info',
          elements: heavyElements.slice(0, 5)
        });
      }

      return {
        totalInlineHandlers: inlineHandlerCount,
        elementsWithHandlers: elementsWithInlineHandlers.length,
        topElements: elementsWithInlineHandlers.slice(0, 20),
        issues: issues,
        recommendations: (function() {
          var recs = [];
          if (inlineHandlerCount > 20) {
            recs.push('Consider using addEventListener() instead of inline handlers for better maintainability.');
          }
          if (inlineHandlerCount > 50) {
            recs.push('Use event delegation on parent containers to reduce listener count.');
          }
          if (elementsWithInlineHandlers.length > 100) {
            recs.push('Large number of elements with handlers may indicate memory leak risk. Ensure cleanup on removal.');
          }
          return recs;
        })(),
        note: 'This audit only detects inline HTML handlers. Use Chrome DevTools getEventListeners() for comprehensive listener inspection.',
        timestamp: Date.now()
      };
    },

    // Estimate Total Blocking Time (TBT)
    estimateTBT: function() {
      var longTasks = [];
      var tbt = 0;

      // Try to get long tasks
      if (performance.getEntriesByType) {
        try {
          longTasks = performance.getEntriesByType('longtask') || [];
        } catch (e) {
          // longtask may not be available
        }
      }

      // Calculate TBT: sum of (duration - 50ms) for each long task
      longTasks.forEach(function(task) {
        if (task.duration > 50) {
          tbt += task.duration - 50;
        }
      });

      // Get FCP for context
      var fcp = 0;
      if (performance.getEntriesByType) {
        var paintEntries = performance.getEntriesByType('paint') || [];
        paintEntries.forEach(function(entry) {
          if (entry.name === 'first-contentful-paint') {
            fcp = entry.startTime;
          }
        });
      }

      return {
        totalBlockingTime: Math.round(tbt),
        longTaskCount: longTasks.length,
        longTasks: longTasks.slice(0, 20).map(function(task) {
          return {
            duration: Math.round(task.duration),
            blockingTime: Math.round(Math.max(0, task.duration - 50)),
            startTime: Math.round(task.startTime),
            name: task.name || 'unknown'
          };
        }),
        rating: tbt < 200 ? 'good' : tbt < 600 ? 'needs-improvement' : 'poor',
        context: {
          fcp: Math.round(fcp),
          note: 'TBT measures blocking time between FCP and TTI. Lower is better.'
        },
        thresholds: {
          good: '< 200ms',
          needsImprovement: '200-600ms',
          poor: '> 600ms'
        },
        timestamp: Date.now()
      };
    },

    // ========================================================================
    // COMPREHENSIVE QUALITY AUDIT
    // ========================================================================

    // Run all quality checks and generate comprehensive report
    auditPageQuality: function(options) {
      options = options || {};
      var self = this;

      return new Promise(function(resolve, reject) {
        try {
          // Gather all metrics
          var domComplexity = self.auditDOMComplexity();
          var memory = self.captureMemoryMetrics();
          var tbt = self.estimateTBT();
          var eventListeners = self.auditEventListeners();
          var performanceMetrics = self.capturePerformanceMetrics();
          var textFragility = self.checkTextFragility();
          var responsiveRisk = self.checkResponsiveRisk();

          // Calculate composite scores
          var scores = {
            dom: domComplexity.scores.overall,
            tbt: tbt.rating === 'good' ? 100 : tbt.rating === 'needs-improvement' ? 60 : 30,
            memory: memory.available ? (memory.pressure === 'low' ? 100 : memory.pressure === 'moderate' ? 70 : memory.pressure === 'high' ? 40 : 20) : null,
            eventListeners: eventListeners.totalInlineHandlers < 20 ? 100 : eventListeners.totalInlineHandlers < 50 ? 70 : 40,
            text: Math.max(0, 100 - (textFragility.summary.errors * 15) - (textFragility.summary.warnings * 5)),
            responsive: Math.max(0, 100 - (responsiveRisk.summary.errors * 15) - (responsiveRisk.summary.warnings * 5)),
            cls: performanceMetrics.cls ? (performanceMetrics.cls.rating === 'good' ? 100 : performanceMetrics.cls.rating === 'needs-improvement' ? 60 : 30) : null
          };

          // Calculate overall score (weighted average of available scores)
          var weightedScores = [];
          if (scores.dom !== null) weightedScores.push({ score: scores.dom, weight: 0.15 });
          if (scores.tbt !== null) weightedScores.push({ score: scores.tbt, weight: 0.2 });
          if (scores.memory !== null) weightedScores.push({ score: scores.memory, weight: 0.1 });
          if (scores.eventListeners !== null) weightedScores.push({ score: scores.eventListeners, weight: 0.1 });
          if (scores.text !== null) weightedScores.push({ score: scores.text, weight: 0.15 });
          if (scores.responsive !== null) weightedScores.push({ score: scores.responsive, weight: 0.15 });
          if (scores.cls !== null) weightedScores.push({ score: scores.cls, weight: 0.15 });

          var totalWeight = weightedScores.reduce(function(sum, s) { return sum + s.weight; }, 0);
          var overallScore = Math.round(weightedScores.reduce(function(sum, s) {
            return sum + (s.score * s.weight / totalWeight);
          }, 0));

          // Determine grade
          var grade = 'F';
          if (overallScore >= 90) grade = 'A';
          else if (overallScore >= 80) grade = 'B';
          else if (overallScore >= 70) grade = 'C';
          else if (overallScore >= 60) grade = 'D';

          // Collect all issues and recommendations
          var criticalIssues = [];
          var recommendations = [];
          var priority = 1;

          // DOM issues
          if (domComplexity.thresholds.nodes.exceeded) {
            criticalIssues.push({ category: 'dom', message: 'Excessive DOM nodes: ' + domComplexity.totalNodes });
            recommendations.push({ priority: priority++, category: 'dom', issue: 'DOM has ' + domComplexity.totalNodes + ' nodes', fix: 'Target <1500 nodes. Use virtualization for long lists.' });
          }

          // TBT issues
          if (tbt.rating === 'poor') {
            criticalIssues.push({ category: 'performance', message: 'High Total Blocking Time: ' + tbt.totalBlockingTime + 'ms' });
            recommendations.push({ priority: priority++, category: 'performance', issue: 'TBT is ' + tbt.totalBlockingTime + 'ms', fix: 'Break up long tasks. Use web workers for heavy computation.' });
          }

          // Memory issues
          if (memory.available && memory.pressure === 'critical') {
            criticalIssues.push({ category: 'memory', message: 'Critical memory pressure: ' + memory.jsHeap.percentUsed + '% used' });
            recommendations.push({ priority: priority++, category: 'memory', issue: 'Memory usage at ' + memory.jsHeap.percentUsed + '%', fix: 'Check for memory leaks. Remove unused data and listeners.' });
          }

          // CLS issues
          if (performanceMetrics.cls && performanceMetrics.cls.rating === 'poor') {
            criticalIssues.push({ category: 'layout', message: 'Poor CLS score: ' + performanceMetrics.cls.score });
            recommendations.push({ priority: priority++, category: 'layout', issue: 'CLS is ' + performanceMetrics.cls.score, fix: 'Reserve space for dynamic content. Set dimensions on images/embeds.' });
          }

          // Text fragility
          if (textFragility.summary.errors > 0) {
            recommendations.push({ priority: priority++, category: 'text', issue: textFragility.summary.errors + ' text truncation errors', fix: 'Remove text-overflow: ellipsis or expand containers.' });
          }

          // Responsive issues
          if (responsiveRisk.summary.viewportOverflows > 0) {
            criticalIssues.push({ category: 'responsive', message: responsiveRisk.summary.viewportOverflows + ' elements overflow viewport' });
            recommendations.push({ priority: priority++, category: 'responsive', issue: 'Elements cause horizontal scroll', fix: 'Add max-width: 100% to constrain elements.' });
          }

          // Add DOM recommendations
          domComplexity.recommendations.forEach(function(rec) {
            recommendations.push({ priority: priority++, category: 'dom', issue: rec, fix: rec });
          });

          var result = {
            scores: scores,
            overallScore: overallScore,
            grade: grade,
            criticalIssues: criticalIssues,
            recommendations: recommendations.slice(0, 10),
            details: {
              dom: domComplexity,
              memory: memory,
              tbt: tbt,
              eventListeners: eventListeners,
              performance: performanceMetrics,
              textFragility: { summary: textFragility.summary, issueCount: textFragility.issues.length },
              responsiveRisk: { summary: responsiveRisk.summary, issueCount: responsiveRisk.issues.length }
            },
            timestamp: Date.now()
          };

          resolve(result);
        } catch (e) {
          reject({ error: e.message });
        }
      });
    },

    // ========================================================================
    // CSS ARCHITECTURE & QUALITY EVALUATION
    // ========================================================================

    // Calculate specificity of a CSS selector (returns [inline, id, class, element])
    _calculateSpecificity: function(selector) {
      var spec = [0, 0, 0, 0]; // [inline, id, class, element]
      if (!selector || typeof selector !== 'string') return spec;

      // Remove :not(), :is(), :where() contents for accurate calculation
      // :where() has 0 specificity, :not() and :is() take inner specificity
      var cleaned = selector
        .replace(/:where\([^)]*\)/g, '')
        .replace(/:not\(([^)]*)\)/g, '$1')
        .replace(/:is\(([^)]*)\)/g, '$1');

      // Count IDs
      spec[1] = (cleaned.match(/#[a-zA-Z_-][a-zA-Z0-9_-]*/g) || []).length;

      // Count classes, attributes, pseudo-classes
      spec[2] = (cleaned.match(/\.[a-zA-Z_-][a-zA-Z0-9_-]*/g) || []).length;
      spec[2] += (cleaned.match(/\[[^\]]+\]/g) || []).length;
      spec[2] += (cleaned.match(/:[a-zA-Z-]+(?!\()/g) || []).length;

      // Count elements and pseudo-elements
      var withoutClasses = cleaned.replace(/[.#][a-zA-Z_-][a-zA-Z0-9_-]*/g, '')
                                  .replace(/\[[^\]]+\]/g, '')
                                  .replace(/:[a-zA-Z-]+/g, '');
      spec[3] = (withoutClasses.match(/[a-zA-Z][a-zA-Z0-9]*/g) || []).length;
      spec[3] += (cleaned.match(/::[a-zA-Z-]+/g) || []).length;

      return spec;
    },

    // Compare two specificity arrays
    _compareSpecificity: function(a, b) {
      for (var i = 0; i < 4; i++) {
        if (a[i] > b[i]) return 1;
        if (a[i] < b[i]) return -1;
      }
      return 0;
    },

    // Detect content area type (CMS content vs application frame vs layout)
    detectContentAreas: function() {
      var areas = [];

      // Common CMS content area patterns
      var cmsPatterns = [
        { selector: '[contenteditable]', type: 'cms-editable' },
        { selector: '.wp-content, .entry-content, .post-content', type: 'cms-wordpress' },
        { selector: '.prose, .markdown-body, .rich-text', type: 'cms-prose' },
        { selector: '[data-cms], [data-editable], [data-content]', type: 'cms-generic' },
        { selector: 'article, .article-body, .blog-post', type: 'cms-article' },
        { selector: '.ck-content, .trix-content, .ProseMirror', type: 'cms-editor' }
      ];

      // Application frame patterns
      var appPatterns = [
        { selector: 'nav, .navbar, .navigation, header nav', type: 'app-navigation' },
        { selector: '.sidebar, aside, [role="complementary"]', type: 'app-sidebar' },
        { selector: '.toolbar, .action-bar, [role="toolbar"]', type: 'app-toolbar' },
        { selector: '.modal, .dialog, [role="dialog"]', type: 'app-modal' },
        { selector: '.dropdown, .menu, [role="menu"]', type: 'app-menu' },
        { selector: 'form, .form-group, fieldset', type: 'app-form' },
        { selector: '.card, .panel, .widget', type: 'app-component' }
      ];

      // Layout frame patterns
      var layoutPatterns = [
        { selector: 'header, .header, [role="banner"]', type: 'layout-header' },
        { selector: 'footer, .footer, [role="contentinfo"]', type: 'layout-footer' },
        { selector: 'main, .main, [role="main"]', type: 'layout-main' },
        { selector: '.container, .wrapper, .page-wrapper', type: 'layout-container' },
        { selector: '.grid, .row, .columns', type: 'layout-grid' }
      ];

      function analyzeArea(pattern) {
        var elements = document.querySelectorAll(pattern.selector);
        if (elements.length === 0) return;

        for (var i = 0; i < elements.length; i++) {
          var el = elements[i];
          var rect = el.getBoundingClientRect();
          if (rect.width === 0 || rect.height === 0) continue;

          var computed = window.getComputedStyle(el);

          areas.push({
            selector: generateSelector(el),
            type: pattern.type,
            category: pattern.type.split('-')[0],
            dimensions: {
              width: Math.round(rect.width),
              height: Math.round(rect.height)
            },
            containment: {
              contain: computed.contain || 'none',
              contentVisibility: computed.contentVisibility || 'visible',
              containerType: computed.containerType || 'normal'
            },
            hasOverflow: el.scrollHeight > el.clientHeight || el.scrollWidth > el.clientWidth,
            childCount: el.children.length
          });
        }
      }

      cmsPatterns.forEach(analyzeArea);
      appPatterns.forEach(analyzeArea);
      layoutPatterns.forEach(analyzeArea);

      // Categorize results
      var byCategory = { cms: [], app: [], layout: [] };
      areas.forEach(function(a) {
        if (byCategory[a.category]) {
          byCategory[a.category].push(a);
        }
      });

      return {
        areas: areas,
        byCategory: byCategory,
        summary: {
          total: areas.length,
          cms: byCategory.cms.length,
          app: byCategory.app.length,
          layout: byCategory.layout.length
        },
        recommendations: (function() {
          var recs = [];

          // CMS areas should have flexible styling
          byCategory.cms.forEach(function(area) {
            if (area.containment.contain !== 'none' && area.containment.contain !== 'style') {
              recs.push({
                area: area.selector,
                type: 'cms-containment',
                message: 'CMS content area has layout containment - may break user content',
                fix: 'Use contain: style or none for CMS areas to allow flexible content'
              });
            }
          });

          // App areas should have containment for performance
          byCategory.app.forEach(function(area) {
            if (area.containment.contain === 'none' && area.childCount > 20) {
              recs.push({
                area: area.selector,
                type: 'app-containment',
                message: 'App component with many children lacks containment',
                fix: 'Add contain: layout or contain: content for performance'
              });
            }
          });

          return recs;
        })(),
        timestamp: Date.now()
      };
    },

    // Audit CSS architecture: specificity, selectors, patterns
    auditCSSArchitecture: function() {
      var self = this;
      var issues = [];
      var selectorStats = {
        total: 0,
        bySpecificity: { low: 0, medium: 0, high: 0, extreme: 0 },
        idSelectors: [],
        deepNesting: [],
        fragilePatterns: [],
        importantCount: 0
      };

      // Analyze all stylesheets
      var stylesheets = document.styleSheets;
      var allRules = [];

      for (var i = 0; i < stylesheets.length; i++) {
        try {
          var sheet = stylesheets[i];
          if (!sheet.cssRules) continue;

          var isExternal = sheet.href && !sheet.href.startsWith(window.location.origin);

          for (var j = 0; j < sheet.cssRules.length; j++) {
            var rule = sheet.cssRules[j];
            if (rule.type !== CSSRule.STYLE_RULE) continue;

            var selector = rule.selectorText;
            var spec = self._calculateSpecificity(selector);
            var specScore = spec[1] * 100 + spec[2] * 10 + spec[3];

            selectorStats.total++;

            // Categorize by specificity
            if (specScore <= 10) selectorStats.bySpecificity.low++;
            else if (specScore <= 30) selectorStats.bySpecificity.medium++;
            else if (specScore <= 100) selectorStats.bySpecificity.high++;
            else selectorStats.bySpecificity.extreme++;

            // Check for ID selectors
            if (spec[1] > 0) {
              selectorStats.idSelectors.push({
                selector: selector,
                specificity: spec,
                source: sheet.href || 'inline'
              });
            }

            // Check for deep nesting (more than 4 combinators)
            var combinators = (selector.match(/\s+|\s*>\s*|\s*\+\s*|\s*~\s*/g) || []).length;
            if (combinators > 4) {
              selectorStats.deepNesting.push({
                selector: selector,
                depth: combinators,
                source: sheet.href || 'inline'
              });
            }

            // Check for fragile patterns
            var fragilePatterns = [
              { pattern: /\*\s/, name: 'universal-descendant', message: 'Universal selector with descendant combinator is slow' },
              { pattern: />\s*\*/, name: 'universal-child', message: 'Universal child selector is fragile' },
              { pattern: /:nth-child\(\d+\)/, name: 'positional', message: 'Positional selectors are fragile to DOM changes' },
              { pattern: /\[class\*=/, name: 'partial-class', message: 'Partial class matching is fragile' },
              { pattern: /\[id\*=/, name: 'partial-id', message: 'Partial ID matching is fragile' },
              { pattern: /body\s+#|html\s+#/, name: 'root-id', message: 'Qualifying ID with root element is unnecessary' },
              { pattern: /^\s*[a-z]+\s*$/, name: 'bare-element', message: 'Bare element selectors affect all instances globally' }
            ];

            fragilePatterns.forEach(function(fp) {
              if (fp.pattern.test(selector)) {
                selectorStats.fragilePatterns.push({
                  selector: selector,
                  pattern: fp.name,
                  message: fp.message,
                  source: sheet.href || 'inline'
                });
              }
            });

            // Check for !important
            var cssText = rule.cssText || '';
            var importantMatches = cssText.match(/!important/g);
            if (importantMatches) {
              selectorStats.importantCount += importantMatches.length;
            }

            allRules.push({
              selector: selector,
              specificity: spec,
              specScore: specScore,
              source: sheet.href || 'inline',
              isExternal: isExternal
            });
          }
        } catch (e) {
          // Cross-origin stylesheets throw security errors
          continue;
        }
      }

      // Detect naming convention patterns
      var namingPatterns = {
        bem: 0,
        camelCase: 0,
        kebabCase: 0,
        utility: 0,
        other: 0
      };

      var classElements = document.querySelectorAll('[class]');
      var allClasses = new Set();

      for (var k = 0; k < classElements.length; k++) {
        var classes = classElements[k].className.split(/\s+/);
        classes.forEach(function(cls) {
          if (cls) allClasses.add(cls);
        });
      }

      allClasses.forEach(function(cls) {
        if (/^[a-z]+(__[a-z]+)?(--[a-z]+)?$/i.test(cls)) {
          namingPatterns.bem++;
        } else if (/^[a-z]+[A-Z]/.test(cls)) {
          namingPatterns.camelCase++;
        } else if (/^[a-z]+(-[a-z]+)+$/.test(cls)) {
          namingPatterns.kebabCase++;
        } else if (/^(m|p|w|h|flex|grid|text|bg|border)-/.test(cls) || /^(sm|md|lg|xl):/.test(cls)) {
          namingPatterns.utility++;
        } else {
          namingPatterns.other++;
        }
      });

      // Determine dominant pattern
      var maxPattern = Object.keys(namingPatterns).reduce(function(a, b) {
        return namingPatterns[a] > namingPatterns[b] ? a : b;
      });

      // Generate issues and recommendations
      if (selectorStats.idSelectors.length > 5) {
        issues.push({
          type: 'excessive-ids',
          severity: 'warning',
          message: selectorStats.idSelectors.length + ' selectors use IDs - high specificity, hard to override',
          fix: 'Replace ID selectors with classes for better reusability'
        });
      }

      if (selectorStats.deepNesting.length > 0) {
        issues.push({
          type: 'deep-nesting',
          severity: 'warning',
          message: selectorStats.deepNesting.length + ' selectors have deep nesting (>4 levels)',
          fix: 'Flatten selectors using BEM or other naming conventions'
        });
      }

      if (selectorStats.importantCount > 10) {
        issues.push({
          type: 'important-overuse',
          severity: 'error',
          message: selectorStats.importantCount + ' !important declarations found',
          fix: 'Refactor CSS to avoid !important; use @layer for cascade control'
        });
      }

      if (selectorStats.bySpecificity.extreme > selectorStats.total * 0.1) {
        issues.push({
          type: 'specificity-wars',
          severity: 'warning',
          message: 'Over 10% of selectors have extreme specificity',
          fix: 'Adopt a methodology (BEM, ITCSS) to manage specificity'
        });
      }

      // Calculate health score
      var healthScore = 100;
      healthScore -= Math.min(30, selectorStats.idSelectors.length * 2);
      healthScore -= Math.min(20, selectorStats.deepNesting.length * 3);
      healthScore -= Math.min(30, selectorStats.importantCount);
      healthScore -= Math.min(10, selectorStats.fragilePatterns.length * 2);
      healthScore = Math.max(0, healthScore);

      return {
        stats: {
          totalSelectors: selectorStats.total,
          bySpecificity: selectorStats.bySpecificity,
          idSelectorCount: selectorStats.idSelectors.length,
          deepNestingCount: selectorStats.deepNesting.length,
          fragilePatternCount: selectorStats.fragilePatterns.length,
          importantCount: selectorStats.importantCount,
          uniqueClasses: allClasses.size
        },
        idSelectors: selectorStats.idSelectors.slice(0, 20),
        deepNesting: selectorStats.deepNesting.slice(0, 20),
        fragilePatterns: selectorStats.fragilePatterns.slice(0, 20),
        namingConvention: {
          patterns: namingPatterns,
          dominant: maxPattern,
          consistency: namingPatterns[maxPattern] / allClasses.size
        },
        issues: issues,
        healthScore: healthScore,
        rating: healthScore >= 80 ? 'good' : healthScore >= 50 ? 'needs-improvement' : 'poor',
        timestamp: Date.now()
      };
    },

    // Audit CSS containment usage
    auditCSSContainment: function() {
      var allElements = document.querySelectorAll('*');
      var containmentUsage = {
        none: 0,
        layout: 0,
        paint: 0,
        size: 0,
        style: 0,
        content: 0,
        strict: 0
      };
      var contentVisibility = {
        visible: 0,
        auto: 0,
        hidden: 0
      };
      var containerQueries = {
        inlineSize: 0,
        size: 0,
        normal: 0
      };

      var elementsWithContainment = [];
      var candidatesForContainment = [];

      for (var i = 0; i < allElements.length; i++) {
        var el = allElements[i];
        var computed = window.getComputedStyle(el);

        // Check contain property
        var contain = computed.contain || 'none';
        if (contain === 'none') {
          containmentUsage.none++;
        } else {
          if (contain.indexOf('layout') !== -1) containmentUsage.layout++;
          if (contain.indexOf('paint') !== -1) containmentUsage.paint++;
          if (contain.indexOf('size') !== -1) containmentUsage.size++;
          if (contain.indexOf('style') !== -1) containmentUsage.style++;
          if (contain === 'content') containmentUsage.content++;
          if (contain === 'strict') containmentUsage.strict++;

          elementsWithContainment.push({
            selector: generateSelector(el),
            contain: contain,
            childCount: el.children.length
          });
        }

        // Check content-visibility
        var cv = computed.contentVisibility || 'visible';
        if (contentVisibility.hasOwnProperty(cv)) {
          contentVisibility[cv]++;
        }

        // Check container-type
        var ct = computed.containerType || 'normal';
        if (ct === 'inline-size') containerQueries.inlineSize++;
        else if (ct === 'size') containerQueries.size++;
        else containerQueries.normal++;

        // Find candidates for containment
        var rect = el.getBoundingClientRect();
        if (rect.width > 0 && rect.height > 0 && contain === 'none') {
          var isGoodCandidate = false;
          var reason = '';

          // Components with many children
          if (el.children.length > 20) {
            isGoodCandidate = true;
            reason = 'Many children (' + el.children.length + ')';
          }

          // Fixed dimensions
          if (computed.width.match(/px$/) && computed.height.match(/px$/)) {
            isGoodCandidate = true;
            reason = 'Fixed dimensions';
          }

          // Off-screen or below fold
          if (rect.top > window.innerHeight * 2) {
            isGoodCandidate = true;
            reason = 'Below fold (consider content-visibility: auto)';
          }

          // Sidebar/aside patterns
          if (el.tagName === 'ASIDE' || el.classList.contains('sidebar')) {
            isGoodCandidate = true;
            reason = 'Sidebar component';
          }

          if (isGoodCandidate) {
            candidatesForContainment.push({
              selector: generateSelector(el),
              reason: reason,
              suggestedContain: reason.indexOf('Below fold') !== -1 ? 'content-visibility: auto' : 'contain: content'
            });
          }
        }
      }

      var totalWithContainment = elementsWithContainment.length;
      var totalElements = allElements.length;
      var containmentRatio = totalWithContainment / totalElements;

      var issues = [];
      if (candidatesForContainment.length > 10 && containmentRatio < 0.01) {
        issues.push({
          type: 'missing-containment',
          severity: 'info',
          message: 'Many elements could benefit from CSS containment',
          fix: 'Add contain: content or contain: layout to isolated components'
        });
      }

      if (contentVisibility.auto === 0 && candidatesForContainment.filter(function(c) { return c.reason.indexOf('Below fold') !== -1; }).length > 5) {
        issues.push({
          type: 'missing-content-visibility',
          severity: 'warning',
          message: 'Below-fold content not using content-visibility: auto',
          fix: 'Use content-visibility: auto for off-screen content to improve initial render'
        });
      }

      var usingContainerQueries = containerQueries.inlineSize + containerQueries.size > 0;

      return {
        containment: {
          usage: containmentUsage,
          elements: elementsWithContainment.slice(0, 20),
          ratio: (containmentRatio * 100).toFixed(2) + '%'
        },
        contentVisibility: contentVisibility,
        containerQueries: {
          usage: containerQueries,
          inUse: usingContainerQueries
        },
        candidates: candidatesForContainment.slice(0, 30),
        issues: issues,
        recommendations: [
          containmentRatio < 0.01 ? 'Consider adding contain: content to card/panel components' : null,
          contentVisibility.auto === 0 ? 'Use content-visibility: auto for below-fold sections' : null,
          !usingContainerQueries ? 'Consider container queries for truly responsive components' : null
        ].filter(Boolean),
        timestamp: Date.now()
      };
    },

    // Audit responsive strategy: container queries vs media queries
    auditResponsiveStrategy: function() {
      var mediaQueryCount = 0;
      var containerQueryCount = 0;
      var mediaQueryBreakpoints = {};
      var containerQueryContainers = [];

      // Analyze stylesheets for @media and @container rules
      var stylesheets = document.styleSheets;

      for (var i = 0; i < stylesheets.length; i++) {
        try {
          var sheet = stylesheets[i];
          if (!sheet.cssRules) continue;

          function analyzeRules(rules) {
            for (var j = 0; j < rules.length; j++) {
              var rule = rules[j];

              if (rule.type === CSSRule.MEDIA_RULE) {
                mediaQueryCount++;

                // Extract breakpoint values
                var mediaText = rule.media.mediaText || '';
                var widthMatch = mediaText.match(/(\d+)px/g);
                if (widthMatch) {
                  widthMatch.forEach(function(w) {
                    mediaQueryBreakpoints[w] = (mediaQueryBreakpoints[w] || 0) + 1;
                  });
                }

                // Recurse into media rule
                if (rule.cssRules) {
                  analyzeRules(rule.cssRules);
                }
              }

              if (rule.type === CSSRule.CONTAINER_RULE) {
                containerQueryCount++;

                // Extract container name if present
                var containerText = rule.conditionText || '';
                containerQueryContainers.push(containerText);
              }

              // Check for @layer rules
              if (rule.type === CSSRule.LAYER_STATEMENT_RULE || rule.type === CSSRule.LAYER_BLOCK_RULE) {
                // Has CSS layers
              }
            }
          }

          analyzeRules(sheet.cssRules);
        } catch (e) {
          continue;
        }
      }

      // Detect elements with container-type set
      var containerElements = [];
      var allElements = document.querySelectorAll('*');

      for (var k = 0; k < allElements.length; k++) {
        var el = allElements[k];
        var computed = window.getComputedStyle(el);
        var containerType = computed.containerType;

        if (containerType && containerType !== 'normal') {
          containerElements.push({
            selector: generateSelector(el),
            containerType: containerType,
            containerName: computed.containerName || 'none'
          });
        }
      }

      // Determine strategy
      var strategy = 'media-queries-only';
      if (containerQueryCount > 0 && mediaQueryCount > 0) {
        strategy = 'hybrid';
      } else if (containerQueryCount > 0) {
        strategy = 'container-queries-primary';
      }

      // Sort breakpoints by usage
      var sortedBreakpoints = Object.keys(mediaQueryBreakpoints)
        .sort(function(a, b) { return mediaQueryBreakpoints[b] - mediaQueryBreakpoints[a]; })
        .slice(0, 10)
        .map(function(bp) { return { breakpoint: bp, count: mediaQueryBreakpoints[bp] }; });

      var issues = [];
      var recommendations = [];

      // Check for too many breakpoints
      if (Object.keys(mediaQueryBreakpoints).length > 8) {
        issues.push({
          type: 'too-many-breakpoints',
          severity: 'info',
          message: 'Using ' + Object.keys(mediaQueryBreakpoints).length + ' different breakpoints',
          fix: 'Consolidate to 3-5 standard breakpoints for consistency'
        });
      }

      // Recommend container queries for component-based architecture
      if (containerQueryCount === 0 && mediaQueryCount > 20) {
        recommendations.push({
          priority: 1,
          message: 'Consider container queries for component-level responsiveness',
          benefit: 'Components respond to their container, not viewport - more reusable'
        });
      }

      // Check for common breakpoint values
      var commonBreakpoints = ['320px', '375px', '480px', '640px', '768px', '1024px', '1280px', '1536px'];
      var usingCommonBreakpoints = sortedBreakpoints.filter(function(bp) {
        return commonBreakpoints.indexOf(bp.breakpoint) !== -1;
      }).length;

      if (sortedBreakpoints.length > 0 && usingCommonBreakpoints < sortedBreakpoints.length * 0.5) {
        issues.push({
          type: 'non-standard-breakpoints',
          severity: 'info',
          message: 'Using non-standard breakpoints',
          fix: 'Consider aligning with common breakpoints (768px, 1024px, etc.) for consistency'
        });
      }

      return {
        strategy: strategy,
        mediaQueries: {
          count: mediaQueryCount,
          breakpoints: sortedBreakpoints,
          uniqueBreakpoints: Object.keys(mediaQueryBreakpoints).length
        },
        containerQueries: {
          count: containerQueryCount,
          containers: containerElements.slice(0, 20)
        },
        issues: issues,
        recommendations: recommendations,
        suggestion: containerQueryCount === 0 ?
          'Container queries (93%+ browser support) enable truly reusable components' :
          'Good use of modern CSS responsive features',
        timestamp: Date.now()
      };
    },

    // Audit CSS consistency: colors, spacing, fonts
    auditCSSConsistency: function() {
      var colors = {};
      var fontSizes = {};
      var fontFamilies = {};
      var spacings = {};
      var borderRadii = {};

      var allElements = document.querySelectorAll('*');
      var sampleSize = Math.min(allElements.length, 500); // Limit for performance

      for (var i = 0; i < sampleSize; i++) {
        var el = allElements[i];
        var computed = window.getComputedStyle(el);

        // Collect colors
        var bgColor = computed.backgroundColor;
        var textColor = computed.color;
        var borderColor = computed.borderColor;

        if (bgColor && bgColor !== 'rgba(0, 0, 0, 0)') {
          colors[bgColor] = (colors[bgColor] || 0) + 1;
        }
        if (textColor) {
          colors[textColor] = (colors[textColor] || 0) + 1;
        }

        // Collect font sizes
        var fontSize = computed.fontSize;
        if (fontSize) {
          fontSizes[fontSize] = (fontSizes[fontSize] || 0) + 1;
        }

        // Collect font families
        var fontFamily = computed.fontFamily.split(',')[0].trim().replace(/['"]/g, '');
        if (fontFamily) {
          fontFamilies[fontFamily] = (fontFamilies[fontFamily] || 0) + 1;
        }

        // Collect spacings (margin and padding)
        var margins = [computed.marginTop, computed.marginRight, computed.marginBottom, computed.marginLeft];
        var paddings = [computed.paddingTop, computed.paddingRight, computed.paddingBottom, computed.paddingLeft];

        margins.concat(paddings).forEach(function(spacing) {
          if (spacing && spacing !== '0px') {
            spacings[spacing] = (spacings[spacing] || 0) + 1;
          }
        });

        // Collect border radii
        var borderRadius = computed.borderRadius;
        if (borderRadius && borderRadius !== '0px') {
          borderRadii[borderRadius] = (borderRadii[borderRadius] || 0) + 1;
        }
      }

      // Sort and analyze each category
      function analyzeValues(obj, name) {
        var sorted = Object.keys(obj)
          .sort(function(a, b) { return obj[b] - obj[a]; })
          .slice(0, 20)
          .map(function(val) { return { value: val, count: obj[val] }; });

        var uniqueCount = Object.keys(obj).length;
        var isConsistent = uniqueCount < 15; // Arbitrary threshold

        return {
          values: sorted,
          uniqueCount: uniqueCount,
          isConsistent: isConsistent,
          topValue: sorted[0] ? sorted[0].value : null
        };
      }

      var colorAnalysis = analyzeValues(colors, 'colors');
      var fontSizeAnalysis = analyzeValues(fontSizes, 'fontSizes');
      var fontFamilyAnalysis = analyzeValues(fontFamilies, 'fontFamilies');
      var spacingAnalysis = analyzeValues(spacings, 'spacings');
      var borderRadiusAnalysis = analyzeValues(borderRadii, 'borderRadii');

      var issues = [];

      if (colorAnalysis.uniqueCount > 30) {
        issues.push({
          type: 'color-inconsistency',
          severity: 'warning',
          message: colorAnalysis.uniqueCount + ' unique colors - consider a design system',
          fix: 'Define a color palette and use CSS custom properties'
        });
      }

      if (fontSizeAnalysis.uniqueCount > 15) {
        issues.push({
          type: 'font-size-inconsistency',
          severity: 'warning',
          message: fontSizeAnalysis.uniqueCount + ' unique font sizes',
          fix: 'Use a type scale (e.g., 12, 14, 16, 18, 24, 32px)'
        });
      }

      if (spacingAnalysis.uniqueCount > 20) {
        issues.push({
          type: 'spacing-inconsistency',
          severity: 'info',
          message: spacingAnalysis.uniqueCount + ' unique spacing values',
          fix: 'Use a spacing scale (e.g., 4, 8, 12, 16, 24, 32px)'
        });
      }

      if (fontFamilyAnalysis.uniqueCount > 5) {
        issues.push({
          type: 'font-family-inconsistency',
          severity: 'warning',
          message: fontFamilyAnalysis.uniqueCount + ' different font families',
          fix: 'Limit to 2-3 fonts for better performance and consistency'
        });
      }

      // Calculate consistency score
      var score = 100;
      score -= Math.min(20, colorAnalysis.uniqueCount - 10);
      score -= Math.min(20, fontSizeAnalysis.uniqueCount - 8);
      score -= Math.min(15, spacingAnalysis.uniqueCount - 10);
      score -= Math.min(15, fontFamilyAnalysis.uniqueCount - 2);
      score = Math.max(0, score);

      return {
        colors: colorAnalysis,
        fontSizes: fontSizeAnalysis,
        fontFamilies: fontFamilyAnalysis,
        spacing: spacingAnalysis,
        borderRadius: borderRadiusAnalysis,
        issues: issues,
        consistencyScore: Math.round(score),
        rating: score >= 70 ? 'consistent' : score >= 40 ? 'moderate' : 'inconsistent',
        recommendations: [
          colorAnalysis.uniqueCount > 20 ? 'Define CSS custom properties for colors (--color-primary, etc.)' : null,
          fontSizeAnalysis.uniqueCount > 10 ? 'Use a modular type scale' : null,
          spacingAnalysis.uniqueCount > 15 ? 'Adopt a spacing scale (multiples of 4px or 8px)' : null
        ].filter(Boolean),
        timestamp: Date.now()
      };
    },

    // Tailwind CSS-specific audit
    auditTailwind: function() {
      var self = this;
      var results = {
        detected: false,
        version: null,
        config: {
          darkMode: null,
          prefix: null,
          important: null
        },
        usage: {
          totalClasses: 0,
          utilityClasses: 0,
          responsiveClasses: 0,
          stateVariants: 0,
          customClasses: 0,
          arbitraryValues: 0
        },
        patterns: {
          breakpoints: {},
          colors: {},
          spacing: {},
          typography: {}
        },
        issues: [],
        recommendations: []
      };

      // Tailwind detection patterns
      var tailwindPatterns = {
        // Responsive prefixes
        responsive: /^(sm|md|lg|xl|2xl):/,
        // State variants
        states: /^(hover|focus|active|disabled|visited|first|last|odd|even|group-hover|focus-within|focus-visible|dark|motion-safe|motion-reduce|print|portrait|landscape|ltr|rtl|open):/,
        // Common utility patterns
        spacing: /^(m|p|mx|my|mt|mr|mb|ml|px|py|pt|pr|pb|pl|gap|space-x|space-y)-/,
        sizing: /^(w|h|min-w|max-w|min-h|max-h|size)-/,
        flexbox: /^(flex|flex-row|flex-col|flex-wrap|flex-nowrap|justify|items|content|self|order|grow|shrink|basis)-?/,
        grid: /^(grid|grid-cols|grid-rows|col|row|auto-cols|auto-rows|gap)-?/,
        colors: /^(bg|text|border|ring|shadow|accent|caret|fill|stroke|outline|decoration)-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)/,
        typography: /^(text|font|tracking|leading|decoration|underline|overline|line-through|no-underline)-/,
        borders: /^(border|rounded|ring|outline|divide)-/,
        effects: /^(shadow|opacity|mix-blend|bg-blend|filter|backdrop|blur|brightness|contrast|grayscale|hue-rotate|invert|saturate|sepia|drop-shadow)-?/,
        transforms: /^(scale|rotate|translate|skew|origin|transform)-?/,
        transitions: /^(transition|duration|ease|delay|animate)-?/,
        layout: /^(container|columns|break|box|float|clear|isolate|object|overflow|overscroll|position|inset|top|right|bottom|left|z|visible|invisible)-?/,
        // Arbitrary value syntax [...]
        arbitrary: /\[.+\]/
      };

      // Tailwind v3+ specific patterns
      var v3Patterns = {
        // JIT arbitrary values
        arbitraryValue: /^[a-z]+-\[.+\]$/,
        // Arbitrary variants
        arbitraryVariant: /^\[.+\]:/,
        // Container queries (v3.2+)
        containerQuery: /^@\[.+\]:|^@(sm|md|lg|xl):/,
        // Has/group-has (v3.4+)
        hasVariant: /^(has|group-has|peer-has)-\[.+\]/
      };

      // Collect all classes from DOM
      var allElements = document.querySelectorAll('[class]');
      var classUsage = {};
      var totalClassInstances = 0;

      for (var i = 0; i < allElements.length; i++) {
        var el = allElements[i];
        var classList = el.className;
        if (typeof classList !== 'string') continue;

        var classes = classList.split(/\s+/).filter(Boolean);
        classes.forEach(function(cls) {
          classUsage[cls] = (classUsage[cls] || 0) + 1;
          totalClassInstances++;
        });
      }

      var uniqueClasses = Object.keys(classUsage);
      results.usage.totalClasses = uniqueClasses.length;

      // Analyze each class
      var utilityCount = 0;
      var responsiveCount = 0;
      var stateCount = 0;
      var arbitraryCount = 0;
      var customCount = 0;

      var breakpointUsage = { sm: 0, md: 0, lg: 0, xl: 0, '2xl': 0 };
      var colorUsage = {};
      var spacingUsage = {};

      uniqueClasses.forEach(function(cls) {
        var isUtility = false;
        var count = classUsage[cls];

        // Check responsive prefix
        var responsiveMatch = cls.match(tailwindPatterns.responsive);
        if (responsiveMatch) {
          responsiveCount++;
          var bp = responsiveMatch[1];
          breakpointUsage[bp] = (breakpointUsage[bp] || 0) + count;
          isUtility = true;
        }

        // Check state variants
        if (tailwindPatterns.states.test(cls)) {
          stateCount++;
          isUtility = true;
        }

        // Check arbitrary values
        if (tailwindPatterns.arbitrary.test(cls) || v3Patterns.arbitraryValue.test(cls)) {
          arbitraryCount++;
          isUtility = true;
        }

        // Check common utility patterns
        Object.keys(tailwindPatterns).forEach(function(key) {
          if (key !== 'responsive' && key !== 'states' && key !== 'arbitrary') {
            if (tailwindPatterns[key].test(cls.replace(/^(sm|md|lg|xl|2xl):/, '').replace(/^(hover|focus|active|disabled|dark):/, ''))) {
              isUtility = true;

              // Track color usage
              if (key === 'colors') {
                var colorMatch = cls.match(/-(slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)(-\d+)?/);
                if (colorMatch) {
                  var colorName = colorMatch[1];
                  colorUsage[colorName] = (colorUsage[colorName] || 0) + count;
                }
              }

              // Track spacing usage
              if (key === 'spacing') {
                var spacingMatch = cls.match(/-(0|px|0\.5|1|1\.5|2|2\.5|3|3\.5|4|5|6|7|8|9|10|11|12|14|16|20|24|28|32|36|40|44|48|52|56|60|64|72|80|96|auto)$/);
                if (spacingMatch) {
                  var spacingValue = spacingMatch[1];
                  spacingUsage[spacingValue] = (spacingUsage[spacingValue] || 0) + count;
                }
              }
            }
          }
        });

        if (isUtility) {
          utilityCount++;
        } else {
          customCount++;
        }
      });

      results.usage.utilityClasses = utilityCount;
      results.usage.responsiveClasses = responsiveCount;
      results.usage.stateVariants = stateCount;
      results.usage.arbitraryValues = arbitraryCount;
      results.usage.customClasses = customCount;

      results.patterns.breakpoints = breakpointUsage;
      results.patterns.colors = colorUsage;
      results.patterns.spacing = spacingUsage;

      // Detect if Tailwind is in use
      var utilityRatio = utilityCount / uniqueClasses.length;
      results.detected = utilityRatio > 0.3 || responsiveCount > 10;

      // Check for Tailwind CSS file
      var stylesheets = document.styleSheets;
      for (var j = 0; j < stylesheets.length; j++) {
        try {
          var sheet = stylesheets[j];
          if (sheet.href && sheet.href.indexOf('tailwind') !== -1) {
            results.detected = true;
          }
        } catch (e) {
          continue;
        }
      }

      // Detect configuration from CSS custom properties
      var rootStyles = window.getComputedStyle(document.documentElement);
      var twColors = ['--tw-ring-color', '--tw-shadow', '--tw-bg-opacity'];
      twColors.forEach(function(prop) {
        if (rootStyles.getPropertyValue(prop)) {
          results.detected = true;
        }
      });

      // Detect dark mode configuration
      if (document.documentElement.classList.contains('dark') ||
          document.body.classList.contains('dark') ||
          uniqueClasses.some(function(c) { return /^dark:/.test(c); })) {
        results.config.darkMode = 'class';
      } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
        results.config.darkMode = 'media';
      }

      // Generate issues and recommendations
      if (!results.detected) {
        return {
          detected: false,
          message: 'Tailwind CSS not detected on this page',
          timestamp: Date.now()
        };
      }

      // Issue: Excessive arbitrary values
      if (arbitraryCount > uniqueClasses.length * 0.2) {
        results.issues.push({
          type: 'excessive-arbitrary-values',
          severity: 'warning',
          message: 'High usage of arbitrary values (' + arbitraryCount + ' / ' + uniqueClasses.length + ')',
          fix: 'Extend Tailwind config with custom values instead of using arbitrary syntax',
          details: {
            count: arbitraryCount,
            percentage: ((arbitraryCount / uniqueClasses.length) * 100).toFixed(1) + '%'
          }
        });
      }

      // Issue: Inconsistent breakpoint usage
      var bpValues = Object.values(breakpointUsage).filter(function(v) { return v > 0; });
      if (bpValues.length > 0) {
        var avgBp = bpValues.reduce(function(a, b) { return a + b; }, 0) / bpValues.length;
        var inconsistentBp = Object.keys(breakpointUsage).filter(function(bp) {
          return breakpointUsage[bp] > 0 && breakpointUsage[bp] < avgBp * 0.3;
        });
        if (inconsistentBp.length > 0) {
          results.issues.push({
            type: 'inconsistent-breakpoints',
            severity: 'info',
            message: 'Some breakpoints are rarely used: ' + inconsistentBp.join(', '),
            fix: 'Consider consolidating responsive styles to fewer breakpoints'
          });
        }
      }

      // Issue: Missing responsive styles
      if (responsiveCount === 0 && uniqueClasses.length > 50) {
        results.issues.push({
          type: 'no-responsive-styles',
          severity: 'warning',
          message: 'No responsive utility classes detected',
          fix: 'Add sm:, md:, lg: prefixes for responsive design'
        });
      }

      // Issue: Mixing custom CSS with utilities
      if (customCount > uniqueClasses.length * 0.4 && utilityCount > 20) {
        results.issues.push({
          type: 'mixed-methodology',
          severity: 'info',
          message: 'Significant mix of utility classes (' + utilityCount + ') and custom classes (' + customCount + ')',
          fix: 'Consider using @apply in component CSS or extracting Tailwind components'
        });
      }

      // Check for @apply overuse in stylesheets
      var applyCount = 0;
      for (var k = 0; k < stylesheets.length; k++) {
        try {
          var styleSheet = stylesheets[k];
          if (!styleSheet.cssRules) continue;
          for (var l = 0; l < styleSheet.cssRules.length; l++) {
            var cssText = styleSheet.cssRules[l].cssText || '';
            // Note: @apply is processed at build time, but we can detect patterns
            // that suggest heavy component extraction
          }
        } catch (e) {
          continue;
        }
      }

      // Issue: Long class strings (maintainability)
      var longClassElements = 0;
      for (var m = 0; m < allElements.length; m++) {
        var className = allElements[m].className;
        if (typeof className === 'string' && className.split(/\s+/).length > 15) {
          longClassElements++;
        }
      }
      if (longClassElements > 5) {
        results.issues.push({
          type: 'long-class-strings',
          severity: 'info',
          message: longClassElements + ' elements have more than 15 utility classes',
          fix: 'Extract to components using @apply or create component abstractions'
        });
      }

      // Issue: Deprecated/removed utilities (Tailwind v3 removed some v2 utilities)
      var deprecatedPatterns = [
        { pattern: /^(flex-grow|flex-shrink)$/, message: 'Use grow/shrink instead (Tailwind v3)' },
        { pattern: /^(overflow-ellipsis)$/, message: 'Use text-ellipsis instead (Tailwind v3)' },
        { pattern: /^(decoration-clone|decoration-slice)$/, message: 'Use box-decoration-clone/slice instead' }
      ];

      var deprecatedFound = [];
      uniqueClasses.forEach(function(cls) {
        var baseClass = cls.replace(/^(sm|md|lg|xl|2xl|hover|focus|active|disabled|dark):/, '');
        deprecatedPatterns.forEach(function(dp) {
          if (dp.pattern.test(baseClass)) {
            deprecatedFound.push({ class: cls, message: dp.message });
          }
        });
      });

      if (deprecatedFound.length > 0) {
        results.issues.push({
          type: 'deprecated-utilities',
          severity: 'warning',
          message: 'Found ' + deprecatedFound.length + ' deprecated utility classes',
          fix: 'Update to Tailwind v3 equivalents',
          details: deprecatedFound.slice(0, 10)
        });
      }

      // Recommendations
      if (!results.config.darkMode && uniqueClasses.some(function(c) { return /^dark:/.test(c); })) {
        results.recommendations.push({
          priority: 1,
          type: 'dark-mode-config',
          message: 'Dark mode classes detected but dark mode may not be configured',
          action: 'Ensure darkMode: "class" or "media" is set in tailwind.config.js'
        });
      }

      if (arbitraryCount > 10) {
        results.recommendations.push({
          priority: 2,
          type: 'extend-theme',
          message: 'Many arbitrary values could be added to theme configuration',
          action: 'Extract common arbitrary values to tailwind.config.js theme.extend'
        });
      }

      if (responsiveCount > 0 && !breakpointUsage['sm']) {
        results.recommendations.push({
          priority: 3,
          type: 'mobile-first',
          message: 'No sm: breakpoint usage - ensure mobile-first approach',
          action: 'Base styles should be mobile, use sm:/md:/lg: for larger screens'
        });
      }

      // Check for missing common patterns
      var hasFlexOrGrid = uniqueClasses.some(function(c) { return tailwindPatterns.flexbox.test(c) || tailwindPatterns.grid.test(c); });
      if (!hasFlexOrGrid && uniqueClasses.length > 30) {
        results.recommendations.push({
          priority: 3,
          type: 'layout-utilities',
          message: 'No flex/grid utilities detected',
          action: 'Consider using flex/grid utilities for layout instead of custom CSS'
        });
      }

      // JIT-specific checks
      var hasJitFeatures = arbitraryCount > 0 ||
        uniqueClasses.some(function(c) { return v3Patterns.arbitraryVariant.test(c) || v3Patterns.containerQuery.test(c); });

      if (hasJitFeatures) {
        results.recommendations.push({
          priority: 4,
          type: 'jit-mode',
          message: 'JIT mode features detected (arbitrary values/variants)',
          action: 'Ensure Tailwind v3+ or JIT mode is enabled for these features to work'
        });
      }

      // Calculate Tailwind-specific health score
      var healthScore = 100;
      results.issues.forEach(function(issue) {
        if (issue.severity === 'error') healthScore -= 15;
        else if (issue.severity === 'warning') healthScore -= 8;
        else healthScore -= 3;
      });
      healthScore = Math.max(0, healthScore);

      results.healthScore = healthScore;
      results.rating = healthScore >= 80 ? 'good' : healthScore >= 50 ? 'needs-improvement' : 'poor';
      results.timestamp = Date.now();

      return results;
    },

    // Comprehensive CSS audit
    auditCSS: function(options) {
      var self = this;
      options = options || {};
      var includeTailwind = options.includeTailwind !== false; // Default true

      var architecture = self.auditCSSArchitecture();
      var containment = self.auditCSSContainment();
      var responsive = self.auditResponsiveStrategy();
      var consistency = self.auditCSSConsistency();
      var contentAreas = self.detectContentAreas();

      // Optionally include Tailwind audit
      var tailwind = null;
      if (includeTailwind) {
        tailwind = self.auditTailwind();
      }

      // Combine all issues
      var allIssues = []
        .concat(architecture.issues)
        .concat(containment.issues)
        .concat(responsive.issues)
        .concat(consistency.issues);

      // Add Tailwind issues if detected
      if (tailwind && tailwind.detected && tailwind.issues) {
        allIssues = allIssues.concat(tailwind.issues);
      }

      // Calculate overall score
      var baseScore = Math.round(
        (architecture.healthScore * 0.35) +
        (consistency.consistencyScore * 0.25) +
        (containment.candidates.length < 10 ? 100 : Math.max(0, 100 - containment.candidates.length * 2)) * 0.2 +
        (responsive.containerQueries.count > 0 ? 100 : 70) * 0.2
      );

      // Adjust score based on Tailwind health if using Tailwind
      var overallScore = baseScore;
      if (tailwind && tailwind.detected && tailwind.healthScore !== undefined) {
        // Blend Tailwind score with base score (30% weight for Tailwind)
        overallScore = Math.round(baseScore * 0.7 + tailwind.healthScore * 0.3);
      }

      var grade = 'F';
      if (overallScore >= 90) grade = 'A';
      else if (overallScore >= 80) grade = 'B';
      else if (overallScore >= 70) grade = 'C';
      else if (overallScore >= 60) grade = 'D';

      var result = {
        architecture: architecture,
        containment: containment,
        responsive: responsive,
        consistency: consistency,
        contentAreas: contentAreas,
        summary: {
          totalSelectors: architecture.stats.totalSelectors,
          uniqueClasses: architecture.stats.uniqueClasses,
          namingConvention: architecture.namingConvention.dominant,
          responsiveStrategy: responsive.strategy,
          uniqueColors: consistency.colors.uniqueCount,
          uniqueFontSizes: consistency.fontSizes.uniqueCount,
          usingTailwind: tailwind ? tailwind.detected : false
        },
        issues: allIssues,
        overallScore: overallScore,
        grade: grade,
        timestamp: Date.now()
      };

      // Include Tailwind results if detected
      if (tailwind && tailwind.detected) {
        result.tailwind = tailwind;
        result.summary.tailwindUtilities = tailwind.usage.utilityClasses;
        result.summary.tailwindArbitrary = tailwind.usage.arbitraryValues;
      }

      return result;
    },

    // ========================================================================
    // SECURITY & VALIDATION AUDITING
    // ========================================================================

    // Audit security headers from document/meta tags (limited client-side visibility)
    auditSecurityHeaders: function() {
      var results = {
        headers: {},
        issues: [],
        recommendations: [],
        score: 100
      };

      // Check for CSP meta tag
      var cspMeta = document.querySelector('meta[http-equiv="Content-Security-Policy"]');
      if (cspMeta) {
        results.headers.csp = {
          present: true,
          value: cspMeta.getAttribute('content'),
          source: 'meta-tag'
        };
      } else {
        results.headers.csp = { present: false };
        results.issues.push({
          type: 'missing-csp',
          severity: 'warning',
          message: 'No Content-Security-Policy meta tag found',
          fix: 'Add CSP header or meta tag to prevent XSS attacks'
        });
        results.score -= 15;
      }

      // Check X-Frame-Options equivalent (frame-ancestors in CSP)
      var hasFrameProtection = false;
      if (results.headers.csp && results.headers.csp.value) {
        hasFrameProtection = /frame-ancestors/.test(results.headers.csp.value);
      }
      results.headers.frameProtection = { present: hasFrameProtection };
      if (!hasFrameProtection) {
        results.issues.push({
          type: 'missing-frame-protection',
          severity: 'info',
          message: 'No frame-ancestors directive in CSP',
          fix: 'Add frame-ancestors directive to prevent clickjacking'
        });
        results.score -= 5;
      }

      // Check if page is served over HTTPS
      results.headers.https = {
        present: window.location.protocol === 'https:',
        protocol: window.location.protocol
      };
      if (!results.headers.https.present && window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1') {
        results.issues.push({
          type: 'no-https',
          severity: 'error',
          message: 'Page served over HTTP instead of HTTPS',
          fix: 'Enable HTTPS and redirect HTTP to HTTPS'
        });
        results.score -= 25;
      }

      // Check for mixed content indicators
      var mixedContent = {
        scripts: [],
        stylesheets: [],
        images: [],
        iframes: []
      };

      if (window.location.protocol === 'https:') {
        // Check scripts
        document.querySelectorAll('script[src^="http:"]').forEach(function(el) {
          mixedContent.scripts.push(el.src);
        });
        // Check stylesheets
        document.querySelectorAll('link[rel="stylesheet"][href^="http:"]').forEach(function(el) {
          mixedContent.stylesheets.push(el.href);
        });
        // Check images
        document.querySelectorAll('img[src^="http:"]').forEach(function(el) {
          mixedContent.images.push(el.src);
        });
        // Check iframes
        document.querySelectorAll('iframe[src^="http:"]').forEach(function(el) {
          mixedContent.iframes.push(el.src);
        });
      }

      var totalMixed = mixedContent.scripts.length + mixedContent.stylesheets.length +
                       mixedContent.images.length + mixedContent.iframes.length;
      results.headers.mixedContent = {
        present: totalMixed > 0,
        count: totalMixed,
        details: mixedContent
      };

      if (mixedContent.scripts.length > 0 || mixedContent.stylesheets.length > 0) {
        results.issues.push({
          type: 'mixed-content-active',
          severity: 'error',
          message: 'Active mixed content detected (scripts/stylesheets over HTTP)',
          fix: 'Load all scripts and stylesheets over HTTPS',
          details: {
            scripts: mixedContent.scripts.slice(0, 5),
            stylesheets: mixedContent.stylesheets.slice(0, 5)
          }
        });
        results.score -= 20;
      }

      if (mixedContent.images.length > 0 || mixedContent.iframes.length > 0) {
        results.issues.push({
          type: 'mixed-content-passive',
          severity: 'warning',
          message: 'Passive mixed content detected (images/iframes over HTTP)',
          fix: 'Load all resources over HTTPS'
        });
        results.score -= 5;
      }

      // Check referrer policy
      var referrerMeta = document.querySelector('meta[name="referrer"]');
      results.headers.referrerPolicy = {
        present: !!referrerMeta,
        value: referrerMeta ? referrerMeta.getAttribute('content') : null
      };

      // Generate recommendations
      if (!results.headers.csp.present) {
        results.recommendations.push({
          priority: 1,
          message: 'Implement Content Security Policy',
          action: "Add <meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'self'\">"
        });
      }

      results.score = Math.max(0, results.score);
      results.rating = results.score >= 80 ? 'good' : results.score >= 50 ? 'needs-improvement' : 'poor';
      results.timestamp = Date.now();

      return results;
    },

    // Listen for CSP violations
    observeCSPViolations: function(callback) {
      var violations = [];

      var handler = function(e) {
        var violation = {
          blockedURI: e.blockedURI,
          violatedDirective: e.violatedDirective,
          originalPolicy: e.originalPolicy,
          sourceFile: e.sourceFile,
          lineNumber: e.lineNumber,
          columnNumber: e.columnNumber,
          timestamp: Date.now()
        };
        violations.push(violation);
        if (callback) {
          callback(violation);
        }
      };

      document.addEventListener('securitypolicyviolation', handler);

      return {
        stop: function() {
          document.removeEventListener('securitypolicyviolation', handler);
          return { violations: violations, count: violations.length };
        },
        getViolations: function() {
          return violations.slice();
        }
      };
    },

    // Audit DOM for security issues (innerHTML, eval, dangerous patterns)
    auditDOMSecurity: function() {
      var results = {
        issues: [],
        dangerousElements: {
          inlineScripts: [],
          inlineEventHandlers: [],
          javascriptURLs: [],
          dangerousAttributes: []
        },
        summary: {
          inlineScripts: 0,
          inlineEventHandlers: 0,
          javascriptURLs: 0,
          dangerousAttributes: 0,
          total: 0
        }
      };

      // Check for inline scripts (potential XSS vectors)
      document.querySelectorAll('script:not([src])').forEach(function(el) {
        var content = el.textContent || '';
        // Skip JSON-LD and other data scripts
        if (el.type && (el.type.indexOf('json') !== -1 || el.type.indexOf('template') !== -1)) {
          return;
        }
        results.dangerousElements.inlineScripts.push({
          selector: generateSelector(el),
          length: content.length,
          preview: content.substring(0, 100) + (content.length > 100 ? '...' : '')
        });
        results.summary.inlineScripts++;
      });

      // Check for inline event handlers
      var eventAttributes = ['onclick', 'onload', 'onerror', 'onmouseover', 'onmouseout',
        'onfocus', 'onblur', 'onchange', 'onsubmit', 'onkeydown', 'onkeyup', 'onkeypress'];

      eventAttributes.forEach(function(attr) {
        document.querySelectorAll('[' + attr + ']').forEach(function(el) {
          results.dangerousElements.inlineEventHandlers.push({
            selector: generateSelector(el),
            attribute: attr,
            value: el.getAttribute(attr).substring(0, 50)
          });
          results.summary.inlineEventHandlers++;
        });
      });

      // Check for javascript: URLs
      document.querySelectorAll('a[href^="javascript:"], [src^="javascript:"], [action^="javascript:"]').forEach(function(el) {
        var attr = el.hasAttribute('href') ? 'href' : (el.hasAttribute('src') ? 'src' : 'action');
        results.dangerousElements.javascriptURLs.push({
          selector: generateSelector(el),
          attribute: attr,
          value: el.getAttribute(attr).substring(0, 50)
        });
        results.summary.javascriptURLs++;
      });

      // Check for dangerous attributes
      var dangerousAttrs = ['srcdoc', 'data-html', 'data-content'];
      dangerousAttrs.forEach(function(attr) {
        document.querySelectorAll('[' + attr + ']').forEach(function(el) {
          results.dangerousElements.dangerousAttributes.push({
            selector: generateSelector(el),
            attribute: attr,
            length: (el.getAttribute(attr) || '').length
          });
          results.summary.dangerousAttributes++;
        });
      });

      results.summary.total = results.summary.inlineScripts + results.summary.inlineEventHandlers +
                              results.summary.javascriptURLs + results.summary.dangerousAttributes;

      // Generate issues
      if (results.summary.inlineScripts > 0) {
        results.issues.push({
          type: 'inline-scripts',
          severity: 'warning',
          message: results.summary.inlineScripts + ' inline script(s) found',
          fix: 'Move scripts to external files and use strict CSP',
          wcag: 'N/A'
        });
      }

      if (results.summary.inlineEventHandlers > 0) {
        results.issues.push({
          type: 'inline-event-handlers',
          severity: 'warning',
          message: results.summary.inlineEventHandlers + ' inline event handler(s) found',
          fix: 'Use addEventListener() instead of inline handlers',
          wcag: 'N/A'
        });
      }

      if (results.summary.javascriptURLs > 0) {
        results.issues.push({
          type: 'javascript-urls',
          severity: 'error',
          message: results.summary.javascriptURLs + ' javascript: URL(s) found',
          fix: 'Replace javascript: URLs with proper event handlers',
          wcag: 'N/A'
        });
      }

      // Calculate score
      var score = 100;
      score -= Math.min(20, results.summary.inlineScripts * 2);
      score -= Math.min(15, results.summary.inlineEventHandlers);
      score -= Math.min(25, results.summary.javascriptURLs * 5);
      score -= Math.min(10, results.summary.dangerousAttributes * 2);
      results.score = Math.max(0, score);
      results.rating = results.score >= 80 ? 'good' : results.score >= 50 ? 'needs-improvement' : 'poor';
      results.timestamp = Date.now();

      return results;
    },

    // Detect framework and version
    detectFramework: function() {
      var frameworks = [];

      // React detection
      if (window.React || document.querySelector('[data-reactroot], [data-reactid]')) {
        var reactInfo = { name: 'React', detected: true };
        if (window.React && window.React.version) {
          reactInfo.version = window.React.version;
        }
        // Check for React DevTools hook
        if (window.__REACT_DEVTOOLS_GLOBAL_HOOK__) {
          reactInfo.devToolsInstalled = true;
          // Try to get version from fiber
          if (window.__REACT_DEVTOOLS_GLOBAL_HOOK__.renderers) {
            var renderers = window.__REACT_DEVTOOLS_GLOBAL_HOOK__.renderers;
            renderers.forEach(function(renderer) {
              if (renderer && renderer.version) {
                reactInfo.version = renderer.version;
              }
            });
          }
        }
        frameworks.push(reactInfo);
      }

      // Vue detection
      if (window.Vue || document.querySelector('[data-v-]') || window.__VUE__) {
        var vueInfo = { name: 'Vue', detected: true };
        if (window.Vue && window.Vue.version) {
          vueInfo.version = window.Vue.version;
        }
        // Vue 3 detection
        if (window.__VUE__) {
          vueInfo.version = '3.x';
        }
        // Check for Vue DevTools
        if (window.__VUE_DEVTOOLS_GLOBAL_HOOK__) {
          vueInfo.devToolsInstalled = true;
        }
        frameworks.push(vueInfo);
      }

      // Angular detection
      if (window.ng || document.querySelector('[ng-version], [_nghost], [_ngcontent]')) {
        var angularInfo = { name: 'Angular', detected: true };
        var ngVersion = document.querySelector('[ng-version]');
        if (ngVersion) {
          angularInfo.version = ngVersion.getAttribute('ng-version');
        }
        // AngularJS (1.x) detection
        if (window.angular && window.angular.version) {
          angularInfo.name = 'AngularJS';
          angularInfo.version = window.angular.version.full;
          angularInfo.legacy = true;
        }
        frameworks.push(angularInfo);
      }

      // Svelte detection
      if (document.querySelector('[class*="svelte-"]')) {
        frameworks.push({ name: 'Svelte', detected: true });
      }

      // Next.js detection
      if (window.__NEXT_DATA__ || document.querySelector('#__next')) {
        var nextInfo = { name: 'Next.js', detected: true };
        if (window.__NEXT_DATA__ && window.__NEXT_DATA__.buildId) {
          nextInfo.buildId = window.__NEXT_DATA__.buildId;
        }
        frameworks.push(nextInfo);
      }

      // Nuxt detection
      if (window.__NUXT__ || document.querySelector('#__nuxt')) {
        frameworks.push({ name: 'Nuxt', detected: true });
      }

      // jQuery detection
      if (window.jQuery || window.$) {
        var jqInfo = { name: 'jQuery', detected: true };
        if (window.jQuery && window.jQuery.fn && window.jQuery.fn.jquery) {
          jqInfo.version = window.jQuery.fn.jquery;
        }
        frameworks.push(jqInfo);
      }

      // Preact detection
      if (window.preact) {
        frameworks.push({ name: 'Preact', detected: true });
      }

      // Alpine.js detection
      if (window.Alpine) {
        var alpineInfo = { name: 'Alpine.js', detected: true };
        if (window.Alpine.version) {
          alpineInfo.version = window.Alpine.version;
        }
        frameworks.push(alpineInfo);
      }

      // HTMX detection
      if (window.htmx) {
        var htmxInfo = { name: 'htmx', detected: true };
        if (window.htmx.version) {
          htmxInfo.version = window.htmx.version;
        }
        frameworks.push(htmxInfo);
      }

      return {
        frameworks: frameworks,
        count: frameworks.length,
        primary: frameworks.length > 0 ? frameworks[0].name : null,
        timestamp: Date.now()
      };
    },

    // Audit framework-specific security and quality issues
    auditFrameworkQuality: function() {
      var self = this;
      var detection = self.detectFramework();
      var results = {
        framework: detection,
        issues: [],
        recommendations: [],
        patterns: {}
      };

      // React-specific checks
      if (detection.frameworks.some(function(f) { return f.name === 'React' || f.name === 'Next.js'; })) {
        // Check for dangerouslySetInnerHTML usage
        var dangerousHTML = document.querySelectorAll('[dangerouslysetinnerhtml]');
        // This won't find them directly as React handles this internally
        // Instead check for common patterns in script content
        results.patterns.react = {
          devMode: window.__REACT_DEVTOOLS_GLOBAL_HOOK__ !== undefined,
          strictMode: false // Would need runtime inspection
        };

        // Check for development build in production
        if (window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1') {
          var scripts = document.querySelectorAll('script[src*="react"]');
          scripts.forEach(function(s) {
            if (s.src.indexOf('.development.') !== -1 || s.src.indexOf('-dev.') !== -1) {
              results.issues.push({
                type: 'react-dev-build',
                severity: 'error',
                message: 'React development build detected in production',
                fix: 'Use production build for better performance and security',
                details: { src: s.src }
              });
            }
          });
        }
      }

      // Vue-specific checks
      if (detection.frameworks.some(function(f) { return f.name === 'Vue' || f.name === 'Nuxt'; })) {
        results.patterns.vue = {
          devToolsEnabled: window.__VUE_DEVTOOLS_GLOBAL_HOOK__ !== undefined
        };

        // Check for v-html usage (potential XSS)
        var vHtmlElements = document.querySelectorAll('[v-html]');
        if (vHtmlElements.length > 0) {
          results.issues.push({
            type: 'vue-v-html',
            severity: 'warning',
            message: vHtmlElements.length + ' element(s) using v-html directive',
            fix: 'Sanitize content before using v-html or use v-text instead',
            details: Array.from(vHtmlElements).slice(0, 5).map(function(el) {
              return generateSelector(el);
            })
          });
        }
      }

      // Angular-specific checks
      if (detection.frameworks.some(function(f) { return f.name === 'Angular' || f.name === 'AngularJS'; })) {
        results.patterns.angular = {};

        // Check for AngularJS (1.x) which has known security issues
        var angularJS = detection.frameworks.find(function(f) { return f.name === 'AngularJS'; });
        if (angularJS) {
          results.issues.push({
            type: 'angularjs-legacy',
            severity: 'warning',
            message: 'AngularJS (v' + (angularJS.version || '1.x') + ') detected - end of life',
            fix: 'Migrate to modern Angular (v2+) for security updates'
          });

          // Check for ng-bind-html (potential XSS)
          var ngBindHtml = document.querySelectorAll('[ng-bind-html], [ng-bind-html-unsafe]');
          if (ngBindHtml.length > 0) {
            results.issues.push({
              type: 'angularjs-bind-html',
              severity: 'error',
              message: ngBindHtml.length + ' element(s) using ng-bind-html',
              fix: 'Use $sce.trustAsHtml() only with sanitized content'
            });
          }
        }
      }

      // jQuery-specific checks
      var jquery = detection.frameworks.find(function(f) { return f.name === 'jQuery'; });
      if (jquery) {
        // Check for old jQuery versions with known vulnerabilities
        if (jquery.version) {
          var majorVersion = parseInt(jquery.version.split('.')[0], 10);
          var minorVersion = parseInt(jquery.version.split('.')[1], 10);
          if (majorVersion < 3 || (majorVersion === 3 && minorVersion < 5)) {
            results.issues.push({
              type: 'jquery-vulnerable',
              severity: 'warning',
              message: 'jQuery ' + jquery.version + ' has known XSS vulnerabilities',
              fix: 'Upgrade to jQuery 3.5.0 or later'
            });
          }
        }

        // Check for .html() calls with user input (can't detect dynamically, just flag presence)
        results.patterns.jquery = {
          version: jquery.version,
          modern: jquery.version && parseInt(jquery.version.split('.')[0], 10) >= 3
        };
      }

      // Calculate score
      var score = 100;
      results.issues.forEach(function(issue) {
        if (issue.severity === 'error') score -= 15;
        else if (issue.severity === 'warning') score -= 8;
        else score -= 3;
      });

      results.score = Math.max(0, score);
      results.rating = results.score >= 80 ? 'good' : results.score >= 50 ? 'needs-improvement' : 'poor';
      results.timestamp = Date.now();

      return results;
    },

    // Audit form validation and security
    auditFormSecurity: function() {
      var results = {
        forms: [],
        issues: [],
        summary: {
          total: 0,
          withValidation: 0,
          withAutocomplete: 0,
          withCSRF: 0,
          passwordFields: 0,
          sensitiveFields: 0
        }
      };

      var forms = document.querySelectorAll('form');
      results.summary.total = forms.length;

      forms.forEach(function(form, index) {
        var formInfo = {
          selector: generateSelector(form),
          action: form.action || 'none',
          method: form.method || 'get',
          hasValidation: form.hasAttribute('novalidate') === false,
          fields: []
        };

        // Check for CSRF token
        var csrfFields = form.querySelectorAll('input[name*="csrf"], input[name*="token"], input[name*="_token"]');
        formInfo.hasCSRF = csrfFields.length > 0;
        if (formInfo.hasCSRF) results.summary.withCSRF++;

        // Check autocomplete
        formInfo.autocomplete = form.getAttribute('autocomplete') || 'on';
        if (formInfo.autocomplete !== 'off') results.summary.withAutocomplete++;

        // Analyze fields
        var inputs = form.querySelectorAll('input, textarea, select');
        inputs.forEach(function(input) {
          var fieldInfo = {
            type: input.type || input.tagName.toLowerCase(),
            name: input.name || input.id,
            hasValidation: input.hasAttribute('required') || input.hasAttribute('pattern') ||
                          input.hasAttribute('minlength') || input.hasAttribute('maxlength')
          };

          // Check password fields
          if (input.type === 'password') {
            results.summary.passwordFields++;
            fieldInfo.isPassword = true;

            // Check for autocomplete on password
            if (input.getAttribute('autocomplete') !== 'new-password' &&
                input.getAttribute('autocomplete') !== 'current-password') {
              results.issues.push({
                type: 'password-autocomplete',
                severity: 'info',
                message: 'Password field without specific autocomplete value',
                fix: 'Use autocomplete="new-password" or "current-password"',
                element: generateSelector(input)
              });
            }
          }

          // Check sensitive fields
          var sensitivePatterns = ['ssn', 'social', 'credit', 'card', 'cvv', 'pin', 'account'];
          var fieldName = (input.name || input.id || '').toLowerCase();
          if (sensitivePatterns.some(function(p) { return fieldName.indexOf(p) !== -1; })) {
            results.summary.sensitiveFields++;
            fieldInfo.isSensitive = true;

            if (input.getAttribute('autocomplete') !== 'off') {
              results.issues.push({
                type: 'sensitive-autocomplete',
                severity: 'warning',
                message: 'Sensitive field "' + fieldName + '" should disable autocomplete',
                fix: 'Add autocomplete="off" to sensitive fields',
                element: generateSelector(input)
              });
            }
          }

          if (fieldInfo.hasValidation) results.summary.withValidation++;
          formInfo.fields.push(fieldInfo);
        });

        // Check for action URL security
        if (form.action && form.action.indexOf('http:') === 0 && window.location.protocol === 'https:') {
          results.issues.push({
            type: 'insecure-form-action',
            severity: 'error',
            message: 'Form submits to insecure HTTP URL',
            fix: 'Change form action to HTTPS',
            element: generateSelector(form)
          });
        }

        // Check for missing CSRF on POST forms
        if (form.method.toLowerCase() === 'post' && !formInfo.hasCSRF) {
          results.issues.push({
            type: 'missing-csrf',
            severity: 'warning',
            message: 'POST form without CSRF token',
            fix: 'Add CSRF token to prevent cross-site request forgery',
            element: generateSelector(form)
          });
        }

        results.forms.push(formInfo);
      });

      // Calculate score
      var score = 100;
      results.issues.forEach(function(issue) {
        if (issue.severity === 'error') score -= 15;
        else if (issue.severity === 'warning') score -= 8;
        else score -= 3;
      });

      results.score = Math.max(0, score);
      results.rating = results.score >= 80 ? 'good' : results.score >= 50 ? 'needs-improvement' : 'poor';
      results.timestamp = Date.now();

      return results;
    },

    // Check for potential prototype pollution vulnerabilities
    auditPrototypePollution: function() {
      var results = {
        vulnerable: false,
        tests: [],
        issues: []
      };

      // Test Object.prototype for pollution
      var testKey = '__devtool_proto_test_' + Date.now();
      var originalValue = Object.prototype[testKey];

      // Check if prototype is already polluted with suspicious keys
      var suspiciousKeys = ['__proto__', 'constructor', 'prototype'];
      var foundPollution = [];

      // Check for common pollution patterns
      var testObj = {};
      suspiciousKeys.forEach(function(key) {
        try {
          if (testObj[key] !== undefined && key !== 'constructor') {
            // constructor is expected, others are suspicious
            if (key === '__proto__' && testObj[key] !== Object.prototype) {
              foundPollution.push(key);
            }
          }
        } catch (e) {
          // Some environments restrict access
        }
      });

      // Check for frozen prototype (good security practice)
      var prototypeIsFrozen = Object.isFrozen(Object.prototype);
      results.tests.push({
        name: 'Object.prototype frozen',
        passed: prototypeIsFrozen,
        message: prototypeIsFrozen ? 'Prototype is frozen (good)' : 'Prototype is not frozen'
      });

      if (!prototypeIsFrozen) {
        results.issues.push({
          type: 'prototype-not-frozen',
          severity: 'info',
          message: 'Object.prototype is not frozen',
          fix: 'Consider freezing prototypes: Object.freeze(Object.prototype)'
        });
      }

      // Check for Object.create(null) usage patterns (can't detect directly)
      results.recommendations = [
        'Use Object.create(null) for dictionary objects',
        'Validate and sanitize all user input before using as object keys',
        'Consider using Map instead of plain objects for dynamic keys',
        'Freeze Object.prototype in security-critical applications'
      ];

      results.timestamp = Date.now();
      return results;
    },

    // Audit external resources (scripts, iframes, etc.)
    auditExternalResources: function() {
      var results = {
        scripts: [],
        iframes: [],
        stylesheets: [],
        issues: [],
        summary: {
          totalScripts: 0,
          externalScripts: 0,
          withIntegrity: 0,
          crossOriginScripts: 0,
          iframes: 0,
          sandboxedIframes: 0
        }
      };

      // Audit scripts
      document.querySelectorAll('script[src]').forEach(function(script) {
        var src = script.src;
        var isExternal = !src.startsWith(window.location.origin) && !src.startsWith('/');
        var scriptInfo = {
          src: src,
          isExternal: isExternal,
          hasIntegrity: script.hasAttribute('integrity'),
          crossOrigin: script.getAttribute('crossorigin'),
          async: script.async,
          defer: script.defer
        };

        results.summary.totalScripts++;
        if (isExternal) {
          results.summary.externalScripts++;
          results.summary.crossOriginScripts++;

          if (!script.hasAttribute('integrity')) {
            results.issues.push({
              type: 'missing-sri',
              severity: 'warning',
              message: 'External script without Subresource Integrity (SRI)',
              fix: 'Add integrity attribute with hash of script content',
              resource: src
            });
          } else {
            results.summary.withIntegrity++;
          }
        }

        results.scripts.push(scriptInfo);
      });

      // Audit iframes
      document.querySelectorAll('iframe').forEach(function(iframe) {
        var src = iframe.src;
        var iframeInfo = {
          src: src || '(no src)',
          sandbox: iframe.getAttribute('sandbox'),
          allow: iframe.getAttribute('allow'),
          loading: iframe.loading
        };

        results.summary.iframes++;
        if (iframeInfo.sandbox) {
          results.summary.sandboxedIframes++;
        } else if (src && !src.startsWith(window.location.origin)) {
          results.issues.push({
            type: 'unsandboxed-iframe',
            severity: 'warning',
            message: 'External iframe without sandbox attribute',
            fix: 'Add sandbox attribute to restrict iframe capabilities',
            resource: src
          });
        }

        results.iframes.push(iframeInfo);
      });

      // Audit external stylesheets
      document.querySelectorAll('link[rel="stylesheet"]').forEach(function(link) {
        var href = link.href;
        var isExternal = href && !href.startsWith(window.location.origin) && !href.startsWith('/');

        if (isExternal) {
          var linkInfo = {
            href: href,
            hasIntegrity: link.hasAttribute('integrity'),
            crossOrigin: link.getAttribute('crossorigin')
          };

          if (!link.hasAttribute('integrity')) {
            results.issues.push({
              type: 'missing-sri-css',
              severity: 'info',
              message: 'External stylesheet without SRI',
              fix: 'Add integrity attribute to external stylesheets',
              resource: href
            });
          }

          results.stylesheets.push(linkInfo);
        }
      });

      // Calculate score
      var score = 100;
      results.issues.forEach(function(issue) {
        if (issue.severity === 'error') score -= 15;
        else if (issue.severity === 'warning') score -= 8;
        else score -= 3;
      });

      results.score = Math.max(0, score);
      results.rating = results.score >= 80 ? 'good' : results.score >= 50 ? 'needs-improvement' : 'poor';
      results.timestamp = Date.now();

      return results;
    },

    // Comprehensive security audit
    auditSecurity: function() {
      var self = this;

      var headers = self.auditSecurityHeaders();
      var domSecurity = self.auditDOMSecurity();
      var framework = self.auditFrameworkQuality();
      var forms = self.auditFormSecurity();
      var resources = self.auditExternalResources();
      var prototype = self.auditPrototypePollution();

      // Combine all issues
      var allIssues = []
        .concat(headers.issues)
        .concat(domSecurity.issues)
        .concat(framework.issues)
        .concat(forms.issues)
        .concat(resources.issues)
        .concat(prototype.issues);

      // Calculate overall score (weighted average)
      var overallScore = Math.round(
        (headers.score * 0.25) +
        (domSecurity.score * 0.25) +
        (framework.score * 0.15) +
        (forms.score * 0.20) +
        (resources.score * 0.15)
      );

      var grade = 'F';
      if (overallScore >= 90) grade = 'A';
      else if (overallScore >= 80) grade = 'B';
      else if (overallScore >= 70) grade = 'C';
      else if (overallScore >= 60) grade = 'D';

      // Generate critical issues list
      var criticalIssues = allIssues.filter(function(i) {
        return i.severity === 'error';
      });

      return {
        headers: headers,
        domSecurity: domSecurity,
        framework: framework,
        forms: forms,
        resources: resources,
        prototype: prototype,
        summary: {
          frameworkDetected: framework.framework.primary,
          totalIssues: allIssues.length,
          criticalIssues: criticalIssues.length,
          httpsEnabled: headers.headers.https.present,
          cspEnabled: headers.headers.csp.present
        },
        issues: allIssues,
        criticalIssues: criticalIssues,
        overallScore: overallScore,
        grade: grade,
        timestamp: Date.now()
      };
    },

    // Load DOMPurify from CDN for sanitization
    loadDOMPurify: function() {
      return new Promise(function(resolve, reject) {
        if (window.DOMPurify) {
          resolve(window.DOMPurify);
          return;
        }

        var script = document.createElement('script');
        script.src = 'https://cdnjs.cloudflare.com/ajax/libs/dompurify/3.2.4/purify.min.js';
        script.integrity = 'sha512-tyLAHSqkPuLNxH6FBdJjGMXmEDkbjFtaYwIXPPNLHk85KRPrM/3ErkphWGx5wqGKaUjKulwnEw7HZuXLiEUYAA==';
        script.crossOrigin = 'anonymous';

        script.onload = function() {
          if (window.DOMPurify) {
            resolve(window.DOMPurify);
          } else {
            reject(new Error('DOMPurify failed to load'));
          }
        };

        script.onerror = function() {
          reject(new Error('Failed to load DOMPurify from CDN'));
        };

        document.head.appendChild(script);
      });
    },

    // Sanitize HTML using DOMPurify (loads on demand)
    sanitizeHTML: function(dirty, options) {
      var self = this;
      return self.loadDOMPurify().then(function(DOMPurify) {
        var config = options || {};
        return {
          clean: DOMPurify.sanitize(dirty, config),
          removed: DOMPurify.removed,
          timestamp: Date.now()
        };
      });
    },

    // Check if a string contains potential XSS
    checkXSSRisk: function(input) {
      var risks = [];
      var patterns = [
        { name: 'script-tag', pattern: /<script[\s>]/i, severity: 'high' },
        { name: 'event-handler', pattern: /\bon\w+\s*=/i, severity: 'high' },
        { name: 'javascript-url', pattern: /javascript\s*:/i, severity: 'high' },
        { name: 'data-url', pattern: /data\s*:[^,]*;base64/i, severity: 'medium' },
        { name: 'vbscript-url', pattern: /vbscript\s*:/i, severity: 'high' },
        { name: 'expression', pattern: /expression\s*\(/i, severity: 'medium' },
        { name: 'eval', pattern: /\beval\s*\(/i, severity: 'high' },
        { name: 'document-write', pattern: /document\s*\.\s*write/i, severity: 'medium' },
        { name: 'innerHTML', pattern: /\.innerHTML\s*=/i, severity: 'medium' },
        { name: 'fromCharCode', pattern: /fromCharCode/i, severity: 'low' },
        { name: 'svg-onload', pattern: /<svg[^>]*onload/i, severity: 'high' },
        { name: 'img-onerror', pattern: /<img[^>]*onerror/i, severity: 'high' },
        { name: 'iframe-src', pattern: /<iframe[^>]*src/i, severity: 'medium' }
      ];

      patterns.forEach(function(p) {
        if (p.pattern.test(input)) {
          risks.push({
            type: p.name,
            severity: p.severity,
            pattern: p.pattern.toString()
          });
        }
      });

      return {
        input: input.substring(0, 100) + (input.length > 100 ? '...' : ''),
        hasRisk: risks.length > 0,
        risks: risks,
        highRisk: risks.some(function(r) { return r.severity === 'high'; }),
        timestamp: Date.now()
      };
    }
  };

  // Initialize connection
  connect();

  console.log('[DevTool] API available at window.__devtool');
  console.log('[DevTool] Usage:');
  console.log('  __devtool.log("message", "info", {key: "value"})');
  console.log('  __devtool.screenshot("my-screenshot")');
})();
</script>
`
}

// InjectInstrumentation adds monitoring JavaScript to HTML responses.
// The wsPort parameter is deprecated and unused (kept for backward compatibility).
// The script now uses relative URLs via window.location.host.
func InjectInstrumentation(body []byte, wsPort int) []byte {
	script := instrumentationScript()

	// Try to inject before </head>
	if idx := bytes.Index(body, []byte("</head>")); idx != -1 {
		result := make([]byte, 0, len(body)+len(script))
		result = append(result, body[:idx]...)
		result = append(result, []byte(script)...)
		result = append(result, body[idx:]...)
		return result
	}

	// Try to inject after <head>
	if idx := bytes.Index(body, []byte("<head>")); idx != -1 {
		insertAt := idx + 6
		result := make([]byte, 0, len(body)+len(script))
		result = append(result, body[:insertAt]...)
		result = append(result, []byte(script)...)
		result = append(result, body[insertAt:]...)
		return result
	}

	// Try to inject after <body>
	if idx := bytes.Index(body, []byte("<body")); idx != -1 {
		// Find the end of the body tag
		endIdx := bytes.Index(body[idx:], []byte(">"))
		if endIdx != -1 {
			insertAt := idx + endIdx + 1
			result := make([]byte, 0, len(body)+len(script))
			result = append(result, body[:insertAt]...)
			result = append(result, []byte(script)...)
			result = append(result, body[insertAt:]...)
			return result
		}
	}

	// Try to inject after <html>
	if idx := bytes.Index(body, []byte("<html")); idx != -1 {
		endIdx := bytes.Index(body[idx:], []byte(">"))
		if endIdx != -1 {
			insertAt := idx + endIdx + 1
			result := make([]byte, 0, len(body)+len(script))
			result = append(result, body[:insertAt]...)
			result = append(result, []byte(script)...)
			result = append(result, body[insertAt:]...)
			return result
		}
	}

	// Last resort: prepend to body
	result := make([]byte, 0, len(body)+len(script))
	result = append(result, []byte(script)...)
	result = append(result, body...)
	return result
}

// ShouldInject determines if JavaScript should be injected based on content type.
func ShouldInject(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.Contains(contentType, "text/html")
}
