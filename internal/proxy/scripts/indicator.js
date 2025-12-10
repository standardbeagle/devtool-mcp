// Floating Indicator Bug for DevTool
// A draggable floating indicator with expanding panel for input, screenshots, and element selection

(function() {
  'use strict';

  var core = window.__devtool_core;
  var utils = window.__devtool_utils;

  // Indicator state
  var indicatorState = {
    container: null,
    bug: null,
    panel: null,
    isExpanded: false,
    isDragging: false,
    dragOffset: { x: 0, y: 0 },
    position: { x: 20, y: 20 },
    isMinimized: false,
    isVisible: true, // Show by default
    screenshotMode: null, // null, 'area', 'element'
    selectedElements: [],
    areaSelection: null
  };

  // CSS Styles for the indicator
  var STYLES = {
    bug: [
      'position: fixed',
      'width: 48px',
      'height: 48px',
      'border-radius: 50%',
      'background: linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      'box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4)',
      'cursor: pointer',
      'z-index: 2147483646',
      'display: flex',
      'align-items: center',
      'justify-content: center',
      'transition: transform 0.2s ease, box-shadow 0.2s ease',
      'user-select: none',
      'font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
    ].join(';'),

    bugIcon: [
      'width: 24px',
      'height: 24px',
      'fill: white'
    ].join(';'),

    statusDot: [
      'position: absolute',
      'top: 2px',
      'right: 2px',
      'width: 12px',
      'height: 12px',
      'border-radius: 50%',
      'border: 2px solid white',
      'transition: background-color 0.3s ease'
    ].join(';'),

    panel: [
      'position: fixed',
      'width: 360px',
      'max-height: 500px',
      'background: white',
      'border-radius: 12px',
      'box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15)',
      'z-index: 2147483645',
      'overflow: hidden',
      'font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
      'font-size: 14px',
      'color: #333',
      'transform-origin: bottom left',
      'transition: opacity 0.2s ease, transform 0.2s ease'
    ].join(';'),

    panelHeader: [
      'display: flex',
      'align-items: center',
      'justify-content: space-between',
      'padding: 12px 16px',
      'background: linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      'color: white',
      'font-weight: 600',
      'font-size: 14px'
    ].join(';'),

    panelBody: [
      'padding: 16px',
      'max-height: 400px',
      'overflow-y: auto'
    ].join(';'),

    inputGroup: [
      'margin-bottom: 16px'
    ].join(';'),

    inputLabel: [
      'display: block',
      'margin-bottom: 6px',
      'font-weight: 500',
      'color: #555',
      'font-size: 12px',
      'text-transform: uppercase',
      'letter-spacing: 0.5px'
    ].join(';'),

    textInput: [
      'width: 100%',
      'padding: 10px 12px',
      'border: 1px solid #e0e0e0',
      'border-radius: 8px',
      'font-size: 14px',
      'outline: none',
      'transition: border-color 0.2s ease, box-shadow 0.2s ease',
      'box-sizing: border-box'
    ].join(';'),

    textArea: [
      'width: 100%',
      'padding: 10px 12px',
      'border: 1px solid #e0e0e0',
      'border-radius: 8px',
      'font-size: 14px',
      'outline: none',
      'resize: vertical',
      'min-height: 80px',
      'font-family: inherit',
      'transition: border-color 0.2s ease, box-shadow 0.2s ease',
      'box-sizing: border-box'
    ].join(';'),

    buttonRow: [
      'display: flex',
      'gap: 8px',
      'flex-wrap: wrap'
    ].join(';'),

    button: [
      'flex: 1',
      'min-width: 100px',
      'padding: 10px 16px',
      'border: none',
      'border-radius: 8px',
      'font-size: 13px',
      'font-weight: 500',
      'cursor: pointer',
      'transition: background 0.2s ease, transform 0.1s ease',
      'display: flex',
      'align-items: center',
      'justify-content: center',
      'gap: 6px'
    ].join(';'),

    primaryButton: [
      'background: linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      'color: white'
    ].join(';'),

    secondaryButton: [
      'background: #f0f0f0',
      'color: #333'
    ].join(';'),

    sketchButton: [
      'background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
      'color: white'
    ].join(';'),

    attachmentList: [
      'margin-top: 12px',
      'padding: 8px',
      'background: #f8f9fa',
      'border-radius: 8px',
      'font-size: 12px'
    ].join(';'),

    attachmentItem: [
      'display: flex',
      'align-items: center',
      'justify-content: space-between',
      'padding: 6px 8px',
      'background: white',
      'border-radius: 4px',
      'margin-bottom: 4px'
    ].join(';'),

    closeButton: [
      'background: none',
      'border: none',
      'color: white',
      'cursor: pointer',
      'padding: 4px',
      'display: flex',
      'align-items: center',
      'justify-content: center',
      'opacity: 0.8',
      'transition: opacity 0.2s ease'
    ].join(';'),

    selectionOverlay: [
      'position: fixed',
      'top: 0',
      'left: 0',
      'right: 0',
      'bottom: 0',
      'background: rgba(0, 0, 0, 0.3)',
      'z-index: 2147483647',
      'cursor: crosshair'
    ].join(';'),

    selectionBox: [
      'position: absolute',
      'border: 2px dashed #667eea',
      'background: rgba(102, 126, 234, 0.2)',
      'pointer-events: none'
    ].join(';'),

    elementHighlight: [
      'position: absolute',
      'border: 2px solid #667eea',
      'background: rgba(102, 126, 234, 0.1)',
      'pointer-events: none',
      'z-index: 2147483647'
    ].join(';')
  };

  // SVG Icons
  var ICONS = {
    devtool: '<svg viewBox="0 0 24 24" style="' + STYLES.bugIcon + '"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>',
    close: '<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z"/></svg>',
    send: '<svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor"><path d="M15.854.146a.5.5 0 0 1 .11.54l-5.819 14.547a.75.75 0 0 1-1.329.124l-3.178-4.995L.643 7.184a.75.75 0 0 1 .124-1.33L15.314.037a.5.5 0 0 1 .54.11ZM6.636 10.07l2.761 4.338L14.13 2.576 6.636 10.07Zm6.787-8.201L1.591 6.602l4.339 2.76 7.494-7.493Z"/></svg>',
    screenshot: '<svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor"><path d="M10.5 8.5a2.5 2.5 0 1 1-5 0 2.5 2.5 0 0 1 5 0z"/><path d="M2 4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2h-1.172a2 2 0 0 1-1.414-.586l-.828-.828A2 2 0 0 0 9.172 2H6.828a2 2 0 0 0-1.414.586l-.828.828A2 2 0 0 1 3.172 4H2zm.5 2a.5.5 0 1 1 0-1 .5.5 0 0 1 0 1zm9 2.5a3.5 3.5 0 1 1-7 0 3.5 3.5 0 0 1 7 0z"/></svg>',
    element: '<svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor"><path d="M14 1a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1h12zM2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H2z"/><path d="M6.854 4.646a.5.5 0 0 1 0 .708L4.207 8l2.647 2.646a.5.5 0 0 1-.708.708l-3-3a.5.5 0 0 1 0-.708l3-3a.5.5 0 0 1 .708 0zm2.292 0a.5.5 0 0 0 0 .708L11.793 8l-2.647 2.646a.5.5 0 0 0 .708.708l3-3a.5.5 0 0 0 0-.708l-3-3a.5.5 0 0 0-.708 0z"/></svg>',
    sketch: '<svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor"><path d="M12.146.146a.5.5 0 0 1 .708 0l3 3a.5.5 0 0 1 0 .708l-10 10a.5.5 0 0 1-.168.11l-5 2a.5.5 0 0 1-.65-.65l2-5a.5.5 0 0 1 .11-.168l10-10zM11.207 2.5 13.5 4.793 14.793 3.5 12.5 1.207 11.207 2.5zm1.586 3L10.5 3.207 4 9.707V10h.5a.5.5 0 0 1 .5.5v.5h.5a.5.5 0 0 1 .5.5v.5h.293l6.5-6.5zm-9.761 5.175-.106.106-1.528 3.821 3.821-1.528.106-.106A.5.5 0 0 1 5 12.5V12h-.5a.5.5 0 0 1-.5-.5V11h-.5a.5.5 0 0 1-.468-.325z"/></svg>',
    remove: '<svg width="12" height="12" viewBox="0 0 16 16" fill="currentColor"><path d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6z"/><path fill-rule="evenodd" d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1v1zM4.118 4 4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4H4.118zM2.5 3V2h11v1h-11z"/></svg>'
  };

  // Initialize the indicator
  function init() {
    if (indicatorState.container) return;

    // Load saved preferences (position and visibility)
    loadPreferences();

    // Create container
    indicatorState.container = document.createElement('div');
    indicatorState.container.id = '__devtool-indicator';

    // Apply visibility state
    if (!indicatorState.isVisible) {
      indicatorState.container.style.display = 'none';
    }

    // Create the bug
    createBug();

    // Create the panel (hidden initially)
    createPanel();

    // Add to document
    document.documentElement.appendChild(indicatorState.container);

    // Setup status updates
    setupStatusUpdates();

    console.log('[DevTool] Floating indicator initialized (visible: ' + indicatorState.isVisible + ')');
  }

  function createBug() {
    var bug = document.createElement('div');
    bug.id = '__devtool-bug';
    bug.style.cssText = STYLES.bug;
    bug.style.left = indicatorState.position.x + 'px';
    bug.style.bottom = indicatorState.position.y + 'px';

    // Bug icon
    bug.innerHTML = ICONS.devtool;

    // Status dot
    var statusDot = document.createElement('div');
    statusDot.id = '__devtool-status';
    statusDot.style.cssText = STYLES.statusDot;
    statusDot.style.backgroundColor = core.isConnected() ? '#22c55e' : '#ef4444';
    bug.appendChild(statusDot);

    // Event listeners
    bug.addEventListener('mousedown', handleBugMouseDown);
    bug.addEventListener('click', handleBugClick);
    bug.addEventListener('mouseenter', function() {
      if (!indicatorState.isDragging) {
        bug.style.transform = 'scale(1.1)';
        bug.style.boxShadow = '0 6px 20px rgba(102, 126, 234, 0.5)';
      }
    });
    bug.addEventListener('mouseleave', function() {
      if (!indicatorState.isDragging) {
        bug.style.transform = 'scale(1)';
        bug.style.boxShadow = '0 4px 15px rgba(102, 126, 234, 0.4)';
      }
    });

    indicatorState.bug = bug;
    indicatorState.container.appendChild(bug);
  }

  function createPanel() {
    var panel = document.createElement('div');
    panel.id = '__devtool-panel';
    panel.style.cssText = STYLES.panel;
    panel.style.display = 'none';
    panel.style.opacity = '0';
    panel.style.transform = 'scale(0.95)';

    // Header
    var header = document.createElement('div');
    header.style.cssText = STYLES.panelHeader;
    header.innerHTML = '<span>DevTool Panel</span>';

    var closeBtn = document.createElement('button');
    closeBtn.style.cssText = STYLES.closeButton;
    closeBtn.innerHTML = ICONS.close;
    closeBtn.onclick = function(e) {
      e.stopPropagation();
      togglePanel(false);
    };
    header.appendChild(closeBtn);

    panel.appendChild(header);

    // Body
    var body = document.createElement('div');
    body.style.cssText = STYLES.panelBody;
    body.id = '__devtool-panel-body';

    // Message input group
    var msgGroup = createInputGroup('Message', 'textarea', '__devtool-message', 'Type your message or note...');
    body.appendChild(msgGroup);

    // Button row
    var buttonRow = document.createElement('div');
    buttonRow.style.cssText = STYLES.buttonRow;

    var sendBtn = createButton('Send', ICONS.send, 'primary', handleSendMessage);
    var screenshotBtn = createButton('Screenshot', ICONS.screenshot, 'secondary', handleScreenshotArea);
    var elementBtn = createButton('Select Element', ICONS.element, 'secondary', handleSelectElement);
    var sketchBtn = createButton('Sketch', ICONS.sketch, 'sketch', handleSketchMode);

    buttonRow.appendChild(sendBtn);
    buttonRow.appendChild(screenshotBtn);
    body.appendChild(buttonRow);

    var buttonRow2 = document.createElement('div');
    buttonRow2.style.cssText = STYLES.buttonRow;
    buttonRow2.style.marginTop = '8px';
    buttonRow2.appendChild(elementBtn);
    buttonRow2.appendChild(sketchBtn);
    body.appendChild(buttonRow2);

    // Attachments list
    var attachments = document.createElement('div');
    attachments.id = '__devtool-attachments';
    attachments.style.cssText = STYLES.attachmentList;
    attachments.style.display = 'none';
    attachments.innerHTML = '<div style="font-weight: 500; margin-bottom: 8px; color: #666;">Attachments</div><div id="__devtool-attachment-list"></div>';
    body.appendChild(attachments);

    panel.appendChild(body);
    indicatorState.panel = panel;
    indicatorState.container.appendChild(panel);
  }

  function createInputGroup(label, type, id, placeholder) {
    var group = document.createElement('div');
    group.style.cssText = STYLES.inputGroup;

    var labelEl = document.createElement('label');
    labelEl.style.cssText = STYLES.inputLabel;
    labelEl.textContent = label;
    labelEl.setAttribute('for', id);
    group.appendChild(labelEl);

    var input;
    if (type === 'textarea') {
      input = document.createElement('textarea');
      input.style.cssText = STYLES.textArea;
    } else {
      input = document.createElement('input');
      input.type = type;
      input.style.cssText = STYLES.textInput;
    }
    input.id = id;
    input.placeholder = placeholder;
    input.addEventListener('focus', function() {
      input.style.borderColor = '#667eea';
      input.style.boxShadow = '0 0 0 3px rgba(102, 126, 234, 0.1)';
    });
    input.addEventListener('blur', function() {
      input.style.borderColor = '#e0e0e0';
      input.style.boxShadow = 'none';
    });
    group.appendChild(input);

    return group;
  }

  function createButton(text, icon, type, onClick) {
    var btn = document.createElement('button');
    btn.style.cssText = STYLES.button + ';' + STYLES[type + 'Button'];
    btn.innerHTML = icon + '<span>' + text + '</span>';
    btn.onclick = onClick;
    btn.addEventListener('mouseenter', function() {
      btn.style.transform = 'translateY(-1px)';
    });
    btn.addEventListener('mouseleave', function() {
      btn.style.transform = 'translateY(0)';
    });
    return btn;
  }

  // Event handlers
  function handleBugMouseDown(e) {
    if (e.button !== 0) return;

    indicatorState.isDragging = false;
    indicatorState.dragOffset = {
      x: e.clientX - indicatorState.bug.getBoundingClientRect().left,
      y: e.clientY - indicatorState.bug.getBoundingClientRect().top
    };

    var startX = e.clientX;
    var startY = e.clientY;

    function onMouseMove(e) {
      var dx = Math.abs(e.clientX - startX);
      var dy = Math.abs(e.clientY - startY);

      if (dx > 5 || dy > 5) {
        indicatorState.isDragging = true;
      }

      if (indicatorState.isDragging) {
        var x = e.clientX - indicatorState.dragOffset.x;
        var y = window.innerHeight - e.clientY - (indicatorState.bug.offsetHeight - indicatorState.dragOffset.y);

        // Constrain to viewport
        x = Math.max(0, Math.min(x, window.innerWidth - indicatorState.bug.offsetWidth));
        y = Math.max(0, Math.min(y, window.innerHeight - indicatorState.bug.offsetHeight));

        indicatorState.position = { x: x, y: y };
        indicatorState.bug.style.left = x + 'px';
        indicatorState.bug.style.bottom = y + 'px';
        updatePanelPosition();
      }
    }

    function onMouseUp() {
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);

      if (indicatorState.isDragging) {
        savePreferences();
        setTimeout(function() {
          indicatorState.isDragging = false;
        }, 0);
      }
    }

    document.addEventListener('mousemove', onMouseMove);
    document.addEventListener('mouseup', onMouseUp);
  }

  function handleBugClick(e) {
    if (indicatorState.isDragging) {
      e.preventDefault();
      e.stopPropagation();
      return;
    }
    togglePanel();
  }

  function togglePanel(forceState) {
    var shouldShow = forceState !== undefined ? forceState : !indicatorState.isExpanded;
    indicatorState.isExpanded = shouldShow;

    if (shouldShow) {
      updatePanelPosition();
      indicatorState.panel.style.display = 'block';
      setTimeout(function() {
        indicatorState.panel.style.opacity = '1';
        indicatorState.panel.style.transform = 'scale(1)';
      }, 10);
    } else {
      indicatorState.panel.style.opacity = '0';
      indicatorState.panel.style.transform = 'scale(0.95)';
      setTimeout(function() {
        indicatorState.panel.style.display = 'none';
      }, 200);
    }
  }

  function updatePanelPosition() {
    if (!indicatorState.panel) return;

    var bugRect = indicatorState.bug.getBoundingClientRect();
    var panelWidth = 360;
    var panelHeight = indicatorState.panel.offsetHeight || 400;

    var x = bugRect.left;
    var y = bugRect.top - panelHeight - 10;

    // Adjust if panel would go off screen
    if (x + panelWidth > window.innerWidth) {
      x = window.innerWidth - panelWidth - 10;
    }
    if (y < 10) {
      y = bugRect.bottom + 10;
    }

    indicatorState.panel.style.left = x + 'px';
    indicatorState.panel.style.top = y + 'px';
  }

  function handleSendMessage() {
    var messageEl = document.getElementById('__devtool-message');
    var message = messageEl ? messageEl.value.trim() : '';

    if (!message && indicatorState.selectedElements.length === 0 && !indicatorState.areaSelection) {
      return;
    }

    var payload = {
      message: message,
      attachments: []
    };

    // Add element attachments
    indicatorState.selectedElements.forEach(function(el) {
      payload.attachments.push({
        type: 'element',
        selector: el.selector,
        tag: el.tag,
        id: el.id,
        classes: el.classes,
        text: el.text
      });
    });

    // Add area screenshot attachment
    if (indicatorState.areaSelection) {
      payload.attachments.push({
        type: 'screenshot_area',
        area: indicatorState.areaSelection
      });
    }

    // Send to server
    core.send('panel_message', {
      timestamp: Date.now(),
      payload: payload
    });

    // Clear inputs
    if (messageEl) messageEl.value = '';
    clearAttachments();

    console.log('[DevTool] Message sent:', payload);
  }

  function handleScreenshotArea() {
    togglePanel(false);
    startAreaSelection();
  }

  function handleSelectElement() {
    togglePanel(false);
    startElementSelection();
  }

  function handleSketchMode() {
    togglePanel(false);

    // Check if sketch module is available
    if (window.__devtool_sketch && window.__devtool_sketch.toggle) {
      window.__devtool_sketch.toggle();
    } else {
      console.warn('[DevTool] Sketch module not loaded');
    }
  }

  // Area selection for screenshots
  function startAreaSelection() {
    indicatorState.screenshotMode = 'area';

    var overlay = document.createElement('div');
    overlay.id = '__devtool-selection-overlay';
    overlay.style.cssText = STYLES.selectionOverlay;

    var selectionBox = document.createElement('div');
    selectionBox.id = '__devtool-selection-box';
    selectionBox.style.cssText = STYLES.selectionBox;
    selectionBox.style.display = 'none';
    overlay.appendChild(selectionBox);

    var startPos = null;

    overlay.addEventListener('mousedown', function(e) {
      startPos = { x: e.clientX, y: e.clientY };
      selectionBox.style.display = 'block';
      selectionBox.style.left = startPos.x + 'px';
      selectionBox.style.top = startPos.y + 'px';
      selectionBox.style.width = '0px';
      selectionBox.style.height = '0px';
    });

    overlay.addEventListener('mousemove', function(e) {
      if (!startPos) return;

      var x = Math.min(startPos.x, e.clientX);
      var y = Math.min(startPos.y, e.clientY);
      var width = Math.abs(e.clientX - startPos.x);
      var height = Math.abs(e.clientY - startPos.y);

      selectionBox.style.left = x + 'px';
      selectionBox.style.top = y + 'px';
      selectionBox.style.width = width + 'px';
      selectionBox.style.height = height + 'px';
    });

    overlay.addEventListener('mouseup', function(e) {
      if (!startPos) return;

      var x = Math.min(startPos.x, e.clientX);
      var y = Math.min(startPos.y, e.clientY);
      var width = Math.abs(e.clientX - startPos.x);
      var height = Math.abs(e.clientY - startPos.y);

      if (width > 10 && height > 10) {
        captureArea(x, y, width, height);
      }

      document.body.removeChild(overlay);
      indicatorState.screenshotMode = null;
      togglePanel(true);
    });

    // Allow escape to cancel
    function handleKeyDown(e) {
      if (e.key === 'Escape') {
        document.body.removeChild(overlay);
        indicatorState.screenshotMode = null;
        togglePanel(true);
        document.removeEventListener('keydown', handleKeyDown);
      }
    }
    document.addEventListener('keydown', handleKeyDown);

    document.body.appendChild(overlay);
  }

  function captureArea(x, y, width, height) {
    indicatorState.areaSelection = {
      x: x + window.scrollX,
      y: y + window.scrollY,
      width: width,
      height: height
    };

    // Capture screenshot of area
    if (typeof html2canvas !== 'undefined') {
      html2canvas(document.body, {
        x: x + window.scrollX,
        y: y + window.scrollY,
        width: width,
        height: height,
        allowTaint: true,
        useCORS: true,
        logging: false
      }).then(function(canvas) {
        var dataUrl = canvas.toDataURL('image/png');
        indicatorState.areaSelection.data = dataUrl;
        addAttachment('screenshot', 'Area: ' + width + 'x' + height + 'px');
      }).catch(function(err) {
        console.error('[DevTool] Screenshot failed:', err);
        addAttachment('screenshot', 'Area: ' + width + 'x' + height + 'px (capture pending)');
      });
    } else {
      addAttachment('screenshot', 'Area: ' + width + 'x' + height + 'px (capture pending)');
    }
  }

  // Element selection
  function startElementSelection() {
    indicatorState.screenshotMode = 'element';

    var overlay = document.createElement('div');
    overlay.id = '__devtool-element-overlay';
    overlay.style.cssText = 'position: fixed; top: 0; left: 0; right: 0; bottom: 0; z-index: 2147483647; cursor: crosshair;';

    var highlight = document.createElement('div');
    highlight.id = '__devtool-element-highlight';
    highlight.style.cssText = STYLES.elementHighlight;
    highlight.style.display = 'none';
    overlay.appendChild(highlight);

    var tooltip = document.createElement('div');
    tooltip.id = '__devtool-element-tooltip';
    tooltip.style.cssText = [
      'position: absolute',
      'background: #333',
      'color: white',
      'padding: 4px 8px',
      'border-radius: 4px',
      'font-size: 12px',
      'font-family: monospace',
      'white-space: nowrap',
      'pointer-events: none',
      'z-index: 2147483647'
    ].join(';');
    tooltip.style.display = 'none';
    overlay.appendChild(tooltip);

    var hoveredElement = null;

    overlay.addEventListener('mousemove', function(e) {
      overlay.style.pointerEvents = 'none';
      var el = document.elementFromPoint(e.clientX, e.clientY);
      overlay.style.pointerEvents = 'auto';

      if (!el || el === indicatorState.container || indicatorState.container.contains(el)) {
        highlight.style.display = 'none';
        tooltip.style.display = 'none';
        hoveredElement = null;
        return;
      }

      hoveredElement = el;
      var rect = el.getBoundingClientRect();

      highlight.style.display = 'block';
      highlight.style.left = rect.left + 'px';
      highlight.style.top = rect.top + 'px';
      highlight.style.width = rect.width + 'px';
      highlight.style.height = rect.height + 'px';

      var selector = utils.generateSelector(el);
      tooltip.textContent = selector;
      tooltip.style.display = 'block';
      tooltip.style.left = Math.min(rect.left, window.innerWidth - tooltip.offsetWidth - 10) + 'px';
      tooltip.style.top = Math.max(rect.top - 30, 10) + 'px';
    });

    overlay.addEventListener('click', function(e) {
      e.preventDefault();
      e.stopPropagation();

      if (hoveredElement) {
        var selector = utils.generateSelector(hoveredElement);
        var info = {
          selector: selector,
          tag: hoveredElement.tagName.toLowerCase(),
          id: hoveredElement.id || null,
          classes: Array.from(hoveredElement.classList),
          text: (hoveredElement.textContent || '').trim().substring(0, 100)
        };

        indicatorState.selectedElements.push(info);
        addAttachment('element', info.selector);
      }

      document.body.removeChild(overlay);
      indicatorState.screenshotMode = null;
      togglePanel(true);
    });

    // Allow escape to cancel
    function handleKeyDown(e) {
      if (e.key === 'Escape') {
        document.body.removeChild(overlay);
        indicatorState.screenshotMode = null;
        togglePanel(true);
        document.removeEventListener('keydown', handleKeyDown);
      }
    }
    document.addEventListener('keydown', handleKeyDown);

    document.body.appendChild(overlay);
  }

  // Attachment management
  function addAttachment(type, label) {
    var attachmentsContainer = document.getElementById('__devtool-attachments');
    var list = document.getElementById('__devtool-attachment-list');

    if (!attachmentsContainer || !list) return;

    attachmentsContainer.style.display = 'block';

    var item = document.createElement('div');
    item.style.cssText = STYLES.attachmentItem;

    var icon = type === 'screenshot' ? ICONS.screenshot : ICONS.element;
    var labelSpan = document.createElement('span');
    labelSpan.innerHTML = icon + ' <span style="margin-left: 6px;">' + label + '</span>';
    labelSpan.style.display = 'flex';
    labelSpan.style.alignItems = 'center';
    item.appendChild(labelSpan);

    var removeBtn = document.createElement('button');
    removeBtn.style.cssText = 'background: none; border: none; cursor: pointer; color: #999; padding: 2px;';
    removeBtn.innerHTML = ICONS.remove;
    removeBtn.onclick = function() {
      list.removeChild(item);

      // Remove from state
      if (type === 'screenshot') {
        indicatorState.areaSelection = null;
      } else {
        indicatorState.selectedElements = indicatorState.selectedElements.filter(function(el) {
          return el.selector !== label;
        });
      }

      if (list.children.length === 0) {
        attachmentsContainer.style.display = 'none';
      }
    };
    item.appendChild(removeBtn);

    list.appendChild(item);
  }

  function clearAttachments() {
    indicatorState.selectedElements = [];
    indicatorState.areaSelection = null;

    var attachmentsContainer = document.getElementById('__devtool-attachments');
    var list = document.getElementById('__devtool-attachment-list');

    if (attachmentsContainer) attachmentsContainer.style.display = 'none';
    if (list) list.innerHTML = '';
  }

  // Status updates
  function setupStatusUpdates() {
    setInterval(function() {
      var statusDot = document.getElementById('__devtool-status');
      if (statusDot) {
        statusDot.style.backgroundColor = core.isConnected() ? '#22c55e' : '#ef4444';
      }
    }, 1000);
  }

  // Preferences persistence (position and visibility)
  function savePreferences() {
    try {
      var prefs = {
        position: indicatorState.position,
        isVisible: indicatorState.isVisible
      };
      localStorage.setItem('__devtool_indicator_prefs', JSON.stringify(prefs));
    } catch (e) {
      // localStorage not available
    }
  }

  function loadPreferences() {
    try {
      var saved = localStorage.getItem('__devtool_indicator_prefs');
      if (saved) {
        var prefs = JSON.parse(saved);
        if (prefs.position) {
          indicatorState.position = prefs.position;
        }
        if (typeof prefs.isVisible === 'boolean') {
          indicatorState.isVisible = prefs.isVisible;
        }
      }
    } catch (e) {
      // localStorage not available or invalid data
    }
  }

  // Public methods
  function show() {
    if (indicatorState.container) {
      indicatorState.container.style.display = 'block';
      indicatorState.isVisible = true;
      savePreferences();
    }
  }

  function hide() {
    if (indicatorState.container) {
      indicatorState.container.style.display = 'none';
      indicatorState.isVisible = false;
      savePreferences();
    }
  }

  function toggle() {
    if (indicatorState.container) {
      var isHidden = indicatorState.container.style.display === 'none';
      indicatorState.container.style.display = isHidden ? 'block' : 'none';
      indicatorState.isVisible = isHidden;
      savePreferences();
    }
  }

  function destroy() {
    if (indicatorState.container && indicatorState.container.parentNode) {
      indicatorState.container.parentNode.removeChild(indicatorState.container);
    }
    indicatorState.container = null;
    indicatorState.bug = null;
    indicatorState.panel = null;
    indicatorState.isExpanded = false;
  }

  // Initialize on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Export indicator functions
  window.__devtool_indicator = {
    init: init,
    show: show,
    hide: hide,
    toggle: toggle,
    destroy: destroy,
    togglePanel: togglePanel,
    state: indicatorState
  };
})();
