// Quality audit primitives for DevTool
// DOM complexity, CSS, security, and page quality audits

(function() {
  'use strict';

  var utils = window.__devtool_utils;

  function auditDOMComplexity() {
    var elements = document.querySelectorAll('*');
    var depth = 0;
    var maxDepth = 0;

    function calculateDepth(el) {
      var d = 0;
      var current = el;
      while (current.parentElement) {
        d++;
        current = current.parentElement;
      }
      return d;
    }

    for (var i = 0; i < elements.length; i++) {
      var d = calculateDepth(elements[i]);
      if (d > maxDepth) maxDepth = d;
    }

    var duplicateIds = [];
    var ids = {};
    var elementsWithId = document.querySelectorAll('[id]');
    for (var j = 0; j < elementsWithId.length; j++) {
      var id = elementsWithId[j].id;
      if (ids[id]) {
        duplicateIds.push(id);
      }
      ids[id] = true;
    }

    return {
      totalElements: elements.length,
      maxDepth: maxDepth,
      duplicateIds: duplicateIds,
      elementsWithId: elementsWithId.length,
      forms: document.forms.length,
      images: document.images.length,
      links: document.links.length,
      scripts: document.scripts.length,
      stylesheets: document.styleSheets.length,
      iframes: document.querySelectorAll('iframe').length
    };
  }

  function auditCSS() {
    var issues = [];
    var inlineStyles = document.querySelectorAll('[style]');

    if (inlineStyles.length > 10) {
      issues.push({
        type: 'excessive-inline-styles',
        severity: 'warning',
        count: inlineStyles.length,
        message: 'Many elements with inline styles (' + inlineStyles.length + ')'
      });
    }

    // Check for !important usage
    var importantCount = 0;
    for (var i = 0; i < document.styleSheets.length; i++) {
      try {
        var rules = document.styleSheets[i].cssRules || [];
        for (var j = 0; j < rules.length; j++) {
          if (rules[j].cssText && rules[j].cssText.indexOf('!important') !== -1) {
            importantCount++;
          }
        }
      } catch (e) {
        // Cross-origin stylesheets can't be accessed
      }
    }

    if (importantCount > 5) {
      issues.push({
        type: 'excessive-important',
        severity: 'warning',
        count: importantCount,
        message: 'Many !important declarations (' + importantCount + ')'
      });
    }

    return {
      issues: issues,
      inlineStyleCount: inlineStyles.length,
      importantCount: importantCount,
      stylesheetCount: document.styleSheets.length
    };
  }

  function auditSecurity() {
    var issues = [];

    // Check for HTTP resources on HTTPS page
    if (window.location.protocol === 'https:') {
      var mixedContent = [];

      var scripts = document.querySelectorAll('script[src^="http:"]');
      for (var i = 0; i < scripts.length; i++) {
        mixedContent.push({
          type: 'script',
          url: scripts[i].src
        });
      }

      var links = document.querySelectorAll('link[href^="http:"]');
      for (var j = 0; j < links.length; j++) {
        mixedContent.push({
          type: 'stylesheet',
          url: links[j].href
        });
      }

      var images = document.querySelectorAll('img[src^="http:"]');
      for (var k = 0; k < images.length; k++) {
        mixedContent.push({
          type: 'image',
          url: images[k].src
        });
      }

      if (mixedContent.length > 0) {
        issues.push({
          type: 'mixed-content',
          severity: 'error',
          resources: mixedContent,
          message: 'Mixed content detected (' + mixedContent.length + ' HTTP resources)'
        });
      }
    }

    // Check for forms without HTTPS action
    var forms = document.querySelectorAll('form[action^="http:"]');
    if (forms.length > 0) {
      issues.push({
        type: 'insecure-form',
        severity: 'error',
        count: forms.length,
        message: 'Forms with insecure (HTTP) action URLs'
      });
    }

    // Check for target="_blank" without rel="noopener"
    var unsafeLinks = document.querySelectorAll('a[target="_blank"]:not([rel*="noopener"])');
    if (unsafeLinks.length > 0) {
      issues.push({
        type: 'missing-noopener',
        severity: 'warning',
        count: unsafeLinks.length,
        message: 'Links with target="_blank" missing rel="noopener"'
      });
    }

    // Check for autocomplete on password fields
    var passwordFields = document.querySelectorAll('input[type="password"][autocomplete="on"]');
    if (passwordFields.length > 0) {
      issues.push({
        type: 'password-autocomplete',
        severity: 'warning',
        count: passwordFields.length,
        message: 'Password fields with autocomplete enabled'
      });
    }

    return {
      issues: issues,
      count: issues.length,
      errors: issues.filter(function(i) { return i.severity === 'error'; }).length,
      warnings: issues.filter(function(i) { return i.severity === 'warning'; }).length
    };
  }

  function auditPageQuality() {
    var issues = [];

    // Check for missing meta tags
    if (!document.querySelector('meta[name="viewport"]')) {
      issues.push({
        type: 'missing-viewport',
        severity: 'warning',
        message: 'Missing viewport meta tag'
      });
    }

    if (!document.querySelector('meta[name="description"]')) {
      issues.push({
        type: 'missing-description',
        severity: 'info',
        message: 'Missing meta description'
      });
    }

    // Check document structure
    if (!document.querySelector('h1')) {
      issues.push({
        type: 'missing-h1',
        severity: 'warning',
        message: 'Page missing H1 heading'
      });
    }

    var h1s = document.querySelectorAll('h1');
    if (h1s.length > 1) {
      issues.push({
        type: 'multiple-h1',
        severity: 'info',
        count: h1s.length,
        message: 'Multiple H1 headings found'
      });
    }

    // Check language attribute
    if (!document.documentElement.lang) {
      issues.push({
        type: 'missing-lang',
        severity: 'warning',
        message: 'HTML element missing lang attribute'
      });
    }

    // Check title
    if (!document.title || document.title.trim() === '') {
      issues.push({
        type: 'missing-title',
        severity: 'error',
        message: 'Page missing or empty title'
      });
    }

    return {
      issues: issues,
      count: issues.length,
      title: document.title,
      lang: document.documentElement.lang,
      viewport: document.querySelector('meta[name="viewport"]')?.content
    };
  }

  // Export audit functions
  window.__devtool_audit = {
    auditDOMComplexity: auditDOMComplexity,
    auditCSS: auditCSS,
    auditSecurity: auditSecurity,
    auditPageQuality: auditPageQuality
  };
})();
