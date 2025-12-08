// Accessibility primitives for DevTool
// A11y information, contrast checking, tab order

(function() {
  'use strict';

  var utils = window.__devtool_utils;

  function getA11yInfo(selector) {
    var el = utils.resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var role = el.getAttribute('role') || getImplicitRole(el);

      return {
        role: role,
        ariaLabel: el.getAttribute('aria-label'),
        ariaLabelledBy: el.getAttribute('aria-labelledby'),
        ariaDescribedBy: el.getAttribute('aria-describedby'),
        ariaHidden: el.getAttribute('aria-hidden'),
        ariaExpanded: el.getAttribute('aria-expanded'),
        ariaDisabled: el.getAttribute('aria-disabled'),
        tabIndex: el.tabIndex,
        focusable: isFocusable(el),
        accessibleName: getAccessibleName(el)
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  function getImplicitRole(el) {
    var tag = el.tagName.toLowerCase();
    var roleMap = {
      'a': el.href ? 'link' : null,
      'article': 'article',
      'aside': 'complementary',
      'button': 'button',
      'footer': 'contentinfo',
      'form': 'form',
      'header': 'banner',
      'img': 'img',
      'input': getInputRole(el),
      'li': 'listitem',
      'main': 'main',
      'nav': 'navigation',
      'ol': 'list',
      'section': 'region',
      'select': 'combobox',
      'table': 'table',
      'textarea': 'textbox',
      'ul': 'list'
    };
    return roleMap[tag] || null;
  }

  function getInputRole(el) {
    var type = (el.type || 'text').toLowerCase();
    var inputRoles = {
      'button': 'button',
      'checkbox': 'checkbox',
      'email': 'textbox',
      'number': 'spinbutton',
      'radio': 'radio',
      'range': 'slider',
      'search': 'searchbox',
      'submit': 'button',
      'tel': 'textbox',
      'text': 'textbox',
      'url': 'textbox'
    };
    return inputRoles[type] || 'textbox';
  }

  function isFocusable(el) {
    if (el.disabled) return false;
    if (el.tabIndex < 0) return false;

    var tag = el.tagName.toLowerCase();
    var focusableTags = ['a', 'button', 'input', 'select', 'textarea'];

    if (focusableTags.indexOf(tag) !== -1) return true;
    if (el.tabIndex >= 0) return true;
    if (el.contentEditable === 'true') return true;

    return false;
  }

  function getAccessibleName(el) {
    // Try aria-label first
    var ariaLabel = el.getAttribute('aria-label');
    if (ariaLabel) return ariaLabel;

    // Try aria-labelledby
    var labelledBy = el.getAttribute('aria-labelledby');
    if (labelledBy) {
      var labelEl = document.getElementById(labelledBy);
      if (labelEl) return labelEl.textContent.trim();
    }

    // Try associated label
    if (el.id) {
      var label = document.querySelector('label[for="' + el.id + '"]');
      if (label) return label.textContent.trim();
    }

    // Try alt attribute (for images)
    var alt = el.getAttribute('alt');
    if (alt) return alt;

    // Try title attribute
    var title = el.getAttribute('title');
    if (title) return title;

    // Try text content (for buttons, links)
    if (['button', 'a'].indexOf(el.tagName.toLowerCase()) !== -1) {
      return el.textContent.trim();
    }

    return null;
  }

  function getContrast(foreground, background) {
    function getLuminance(color) {
      // Parse rgb/rgba color string
      var match = color.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
      if (!match) return 0;

      var rgb = [parseInt(match[1]), parseInt(match[2]), parseInt(match[3])];

      for (var i = 0; i < 3; i++) {
        var c = rgb[i] / 255;
        rgb[i] = c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
      }

      return 0.2126 * rgb[0] + 0.7152 * rgb[1] + 0.0722 * rgb[2];
    }

    var lum1 = getLuminance(foreground);
    var lum2 = getLuminance(background);

    var lighter = Math.max(lum1, lum2);
    var darker = Math.min(lum1, lum2);

    var ratio = (lighter + 0.05) / (darker + 0.05);

    return {
      ratio: Math.round(ratio * 100) / 100,
      passesAA: ratio >= 4.5,
      passesAALarge: ratio >= 3,
      passesAAA: ratio >= 7,
      passesAAALarge: ratio >= 4.5
    };
  }

  function getTabOrder() {
    var focusable = document.querySelectorAll(
      'a[href], button, input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );

    var elements = [];
    for (var i = 0; i < focusable.length; i++) {
      var el = focusable[i];
      if (!el.disabled && el.offsetParent !== null) {
        elements.push({
          element: el,
          selector: utils.generateSelector(el),
          tabIndex: el.tabIndex,
          accessibleName: getAccessibleName(el)
        });
      }
    }

    // Sort by tabindex (0 comes last among positive values)
    elements.sort(function(a, b) {
      if (a.tabIndex === b.tabIndex) return 0;
      if (a.tabIndex === 0) return 1;
      if (b.tabIndex === 0) return -1;
      return a.tabIndex - b.tabIndex;
    });

    return { elements: elements, count: elements.length };
  }

  function getScreenReaderText(selector) {
    var el = utils.resolveElement(selector);
    if (!el) return { error: 'Element not found' };

    try {
      var text = [];

      // Accessible name
      var name = getAccessibleName(el);
      if (name) text.push('Name: ' + name);

      // Role
      var role = el.getAttribute('role') || getImplicitRole(el);
      if (role) text.push('Role: ' + role);

      // State
      if (el.getAttribute('aria-expanded')) {
        text.push(el.getAttribute('aria-expanded') === 'true' ? 'expanded' : 'collapsed');
      }
      if (el.getAttribute('aria-checked')) {
        text.push(el.getAttribute('aria-checked') === 'true' ? 'checked' : 'not checked');
      }
      if (el.getAttribute('aria-selected')) {
        text.push(el.getAttribute('aria-selected') === 'true' ? 'selected' : 'not selected');
      }
      if (el.disabled) {
        text.push('disabled');
      }

      // Description
      var describedBy = el.getAttribute('aria-describedby');
      if (describedBy) {
        var descEl = document.getElementById(describedBy);
        if (descEl) text.push('Description: ' + descEl.textContent.trim());
      }

      return {
        text: text.join(', '),
        parts: text
      };
    } catch (e) {
      return { error: e.message };
    }
  }

  function auditAccessibility() {
    var issues = [];

    // Check images without alt
    var images = document.querySelectorAll('img');
    for (var i = 0; i < images.length; i++) {
      var img = images[i];
      if (!img.alt && !img.getAttribute('role')) {
        issues.push({
          type: 'missing-alt',
          severity: 'error',
          element: img,
          selector: utils.generateSelector(img),
          message: 'Image missing alt attribute'
        });
      }
    }

    // Check form inputs without labels
    var inputs = document.querySelectorAll('input, select, textarea');
    for (var j = 0; j < inputs.length; j++) {
      var input = inputs[j];
      if (input.type === 'hidden') continue;

      var hasLabel = input.getAttribute('aria-label') ||
                     input.getAttribute('aria-labelledby') ||
                     (input.id && document.querySelector('label[for="' + input.id + '"]'));

      if (!hasLabel) {
        issues.push({
          type: 'missing-label',
          severity: 'error',
          element: input,
          selector: utils.generateSelector(input),
          message: 'Form input missing label'
        });
      }
    }

    // Check buttons without accessible names
    var buttons = document.querySelectorAll('button, [role="button"]');
    for (var k = 0; k < buttons.length; k++) {
      var btn = buttons[k];
      var name = getAccessibleName(btn);
      if (!name) {
        issues.push({
          type: 'missing-button-label',
          severity: 'error',
          element: btn,
          selector: utils.generateSelector(btn),
          message: 'Button missing accessible name'
        });
      }
    }

    // Check links without href or with empty text
    var links = document.querySelectorAll('a');
    for (var l = 0; l < links.length; l++) {
      var link = links[l];
      if (!link.href) {
        issues.push({
          type: 'link-no-href',
          severity: 'warning',
          element: link,
          selector: utils.generateSelector(link),
          message: 'Link missing href attribute'
        });
      }
      if (link.textContent.trim() === '' && !link.getAttribute('aria-label')) {
        issues.push({
          type: 'empty-link',
          severity: 'error',
          element: link,
          selector: utils.generateSelector(link),
          message: 'Link has no text content or aria-label'
        });
      }
    }

    return {
      issues: issues,
      count: issues.length,
      errors: issues.filter(function(i) { return i.severity === 'error'; }).length,
      warnings: issues.filter(function(i) { return i.severity === 'warning'; }).length
    };
  }

  // Export accessibility functions
  window.__devtool_accessibility = {
    getA11yInfo: getA11yInfo,
    getContrast: getContrast,
    getTabOrder: getTabOrder,
    getScreenReaderText: getScreenReaderText,
    auditAccessibility: auditAccessibility
  };
})();
