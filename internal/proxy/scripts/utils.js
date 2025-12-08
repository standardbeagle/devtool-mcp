// Utility functions for DevTool instrumentation
// Shared helpers used by multiple modules

(function() {
  'use strict';

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

  // Export utilities
  window.__devtool_utils = {
    resolveElement: resolveElement,
    generateSelector: generateSelector,
    safeGetComputed: safeGetComputed,
    parseValue: parseValue,
    getRect: getRect,
    isElementInViewport: isElementInViewport,
    getStackingContext: getStackingContext
  };
})();
