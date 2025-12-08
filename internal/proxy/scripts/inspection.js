// Element inspection primitives for DevTool
// Provides detailed information about DOM elements

(function() {
  'use strict';

  var utils = window.__devtool_utils;

  function getElementInfo(selector) {
    var el = utils.resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var attrs = {};
      for (var i = 0; i < el.attributes.length; i++) {
        var attr = el.attributes[i];
        attrs[attr.name] = attr.value;
      }

      return {
        element: el,
        selector: utils.generateSelector(el),
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
    var el = utils.resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var rect = utils.getRect(el);
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
    var el = utils.resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    return utils.safeGetComputed(el, properties);
  }

  function getBox(selector) {
    var el = utils.resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var computed = window.getComputedStyle(el);

      return {
        margin: {
          top: utils.parseValue(computed.marginTop),
          right: utils.parseValue(computed.marginRight),
          bottom: utils.parseValue(computed.marginBottom),
          left: utils.parseValue(computed.marginLeft)
        },
        border: {
          top: utils.parseValue(computed.borderTopWidth),
          right: utils.parseValue(computed.borderRightWidth),
          bottom: utils.parseValue(computed.borderBottomWidth),
          left: utils.parseValue(computed.borderLeftWidth)
        },
        padding: {
          top: utils.parseValue(computed.paddingTop),
          right: utils.parseValue(computed.paddingRight),
          bottom: utils.parseValue(computed.paddingBottom),
          left: utils.parseValue(computed.paddingLeft)
        },
        content: {
          width: el.clientWidth - utils.parseValue(computed.paddingLeft) - utils.parseValue(computed.paddingRight),
          height: el.clientHeight - utils.parseValue(computed.paddingTop) - utils.parseValue(computed.paddingBottom)
        }
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  function getLayout(selector) {
    var el = utils.resolveElement(selector);
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
    var el = utils.resolveElement(selector);
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
    var el = utils.resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var computed = window.getComputedStyle(el);
      var context = utils.getStackingContext(el);

      return {
        zIndex: computed.zIndex,
        position: computed.position,
        context: context ? utils.generateSelector(context) : null,
        opacity: parseFloat(computed.opacity),
        transform: computed.transform,
        filter: computed.filter
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  function getTransform(selector) {
    var el = utils.resolveElement(selector);
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
    var el = utils.resolveElement(selector);
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

  // Export inspection functions
  window.__devtool_inspection = {
    getElementInfo: getElementInfo,
    getPosition: getPosition,
    getComputed: getComputed,
    getBox: getBox,
    getLayout: getLayout,
    getContainer: getContainer,
    getStacking: getStacking,
    getTransform: getTransform,
    getOverflow: getOverflow
  };
})();
