// Tree walking primitives for DevTool
// Navigate DOM tree relationships

(function() {
  'use strict';

  var utils = window.__devtool_utils;

  function walkChildren(selector, depth, filter) {
    var el = utils.resolveElement(selector);
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
            selector: utils.generateSelector(child),
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
  }

  function walkParents(selector) {
    var el = utils.resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    var parents = [];
    var current = el.parentElement;

    while (current) {
      parents.push({
        element: current,
        selector: utils.generateSelector(current),
        tag: current.tagName.toLowerCase()
      });
      current = current.parentElement;
    }

    return { parents: parents, count: parents.length };
  }

  function findAncestor(selector, condition) {
    var el = utils.resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    if (typeof condition !== 'function') {
      return { error: 'Condition must be a function' };
    }

    var current = el.parentElement;
    while (current) {
      if (condition(current)) {
        return {
          element: current,
          selector: utils.generateSelector(current)
        };
      }
      current = current.parentElement;
    }

    return { found: false };
  }

  // Export tree functions
  window.__devtool_tree = {
    walkChildren: walkChildren,
    walkParents: walkParents,
    findAncestor: findAncestor
  };
})();
