// Interactive primitives for DevTool
// Element selection, waiting, and user prompts

(function() {
  'use strict';

  var utils = window.__devtool_utils;

  function selectElement() {
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

        var selector = utils.generateSelector(target);
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
          var selector = utils.generateSelector(target);
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
  }

  function waitForElement(selector, timeout) {
    timeout = timeout || 5000;
    var startTime = Date.now();

    return new Promise(function(resolve, reject) {
      var el = utils.resolveElement(selector);
      if (el) {
        resolve(el);
        return;
      }

      var observer = new MutationObserver(function() {
        var el = utils.resolveElement(selector);
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
  }

  function ask(question, options) {
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
  }

  function measureBetween(selector1, selector2) {
    var el1 = utils.resolveElement(selector1);
    var el2 = utils.resolveElement(selector2);

    if (!el1 || !el2) return { error: 'Element not found' };

    try {
      var rect1 = utils.getRect(el1);
      var rect2 = utils.getRect(el2);

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
  }

  // Export interactive functions
  window.__devtool_interactive = {
    selectElement: selectElement,
    waitForElement: waitForElement,
    ask: ask,
    measureBetween: measureBetween
  };
})();
