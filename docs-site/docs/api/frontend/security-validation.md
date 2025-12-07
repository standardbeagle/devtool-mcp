---
sidebar_position: 14
---

# Security & Validation Auditing

Functions for auditing security headers, detecting frameworks, checking DOM security, validating forms, and identifying potential vulnerabilities.

## auditSecurityHeaders

Check security-related headers and meta tags visible from the client side.

```javascript
window.__devtool.auditSecurityHeaders()
```

**Returns:**
```javascript
{
  headers: {
    csp: {
      present: true,
      value: "default-src 'self'; script-src 'self' 'unsafe-inline'",
      source: "meta-tag"
    },
    frameProtection: { present: true },
    https: { present: true, protocol: "https:" },
    mixedContent: {
      present: false,
      count: 0,
      details: { scripts: [], stylesheets: [], images: [], iframes: [] }
    },
    referrerPolicy: { present: true, value: "strict-origin-when-cross-origin" }
  },
  issues: [
    {
      type: "missing-frame-protection",
      severity: "info",
      message: "No frame-ancestors directive in CSP",
      fix: "Add frame-ancestors directive to prevent clickjacking"
    }
  ],
  recommendations: [...],
  score: 85,
  rating: "good",
  timestamp: 1699999999999
}
```

**Issues Detected:**
| Type | Severity | Description |
|------|----------|-------------|
| `missing-csp` | warning | No Content-Security-Policy meta tag |
| `missing-frame-protection` | info | No frame-ancestors in CSP |
| `no-https` | error | Page served over HTTP (non-localhost) |
| `mixed-content-active` | error | Scripts/stylesheets loaded over HTTP |
| `mixed-content-passive` | warning | Images/iframes loaded over HTTP |

---

## observeCSPViolations

Monitor Content Security Policy violations in real-time.

```javascript
window.__devtool.observeCSPViolations(callback?)
```

**Parameters:**
- `callback` (function): Called on each violation with violation object

**Returns:**
```javascript
{
  stop: function() { /* Returns { violations, count } */ },
  getViolations: function() { /* Returns current violations array */ }
}
```

**Violation Object:**
```javascript
{
  blockedURI: "https://evil.com/script.js",
  violatedDirective: "script-src",
  originalPolicy: "default-src 'self'",
  sourceFile: "https://example.com/page.html",
  lineNumber: 42,
  columnNumber: 15,
  timestamp: 1699999999999
}
```

**Example:**
```javascript
const observer = window.__devtool.observeCSPViolations(v => {
  console.log('CSP violation:', v.violatedDirective, v.blockedURI)
})

// Later...
const results = observer.stop()
console.log(`Total violations: ${results.count}`)
```

---

## auditDOMSecurity

Audit the DOM for security issues like inline scripts, event handlers, and dangerous patterns.

```javascript
window.__devtool.auditDOMSecurity()
```

**Returns:**
```javascript
{
  issues: [
    {
      type: "inline-scripts",
      severity: "warning",
      message: "5 inline script(s) found",
      fix: "Move scripts to external files and use strict CSP"
    },
    {
      type: "javascript-urls",
      severity: "error",
      message: "2 javascript: URL(s) found",
      fix: "Replace javascript: URLs with proper event handlers"
    }
  ],
  dangerousElements: {
    inlineScripts: [
      { selector: "script:nth-of-type(3)", length: 1250, preview: "var config = {...}" }
    ],
    inlineEventHandlers: [
      { selector: "button.submit", attribute: "onclick", value: "submitForm()" }
    ],
    javascriptURLs: [
      { selector: "a.legacy-link", attribute: "href", value: "javascript:void(0)" }
    ],
    dangerousAttributes: [
      { selector: "div.content", attribute: "srcdoc", length: 500 }
    ]
  },
  summary: {
    inlineScripts: 5,
    inlineEventHandlers: 12,
    javascriptURLs: 2,
    dangerousAttributes: 1,
    total: 20
  },
  score: 65,
  rating: "needs-improvement",
  timestamp: 1699999999999
}
```

**Issues Detected:**
| Type | Severity | Description |
|------|----------|-------------|
| `inline-scripts` | warning | Inline `<script>` tags (XSS vectors) |
| `inline-event-handlers` | warning | onclick, onload, etc. attributes |
| `javascript-urls` | error | `javascript:` URLs in href/src/action |
| `dangerous-attributes` | info | srcdoc, data-html attributes |

---

## detectFramework

Detect JavaScript frameworks and libraries used on the page.

```javascript
window.__devtool.detectFramework()
```

**Returns:**
```javascript
{
  frameworks: [
    { name: "React", detected: true, version: "18.2.0", devToolsInstalled: true },
    { name: "Next.js", detected: true, buildId: "abc123" },
    { name: "jQuery", detected: true, version: "3.6.0" }
  ],
  count: 3,
  primary: "React",
  timestamp: 1699999999999
}
```

**Detected Frameworks:**
| Framework | Detection Method |
|-----------|------------------|
| React | `window.React`, `[data-reactroot]`, DevTools hook |
| Vue | `window.Vue`, `[data-v-]`, `window.__VUE__` |
| Angular | `window.ng`, `[ng-version]`, `[_nghost]` |
| AngularJS | `window.angular` (legacy) |
| Svelte | `[class*="svelte-"]` |
| Next.js | `window.__NEXT_DATA__`, `#__next` |
| Nuxt | `window.__NUXT__`, `#__nuxt` |
| jQuery | `window.jQuery`, `window.$` |
| Preact | `window.preact` |
| Alpine.js | `window.Alpine` |
| htmx | `window.htmx` |

---

## auditFrameworkQuality

Audit framework-specific security and quality issues.

```javascript
window.__devtool.auditFrameworkQuality()
```

**Returns:**
```javascript
{
  framework: { /* detectFramework() results */ },
  issues: [
    {
      type: "react-dev-build",
      severity: "error",
      message: "React development build detected in production",
      fix: "Use production build for better performance and security",
      details: { src: "https://example.com/react.development.js" }
    },
    {
      type: "vue-v-html",
      severity: "warning",
      message: "3 element(s) using v-html directive",
      fix: "Sanitize content before using v-html or use v-text instead",
      details: [".content", ".preview", ".description"]
    },
    {
      type: "jquery-vulnerable",
      severity: "warning",
      message: "jQuery 3.4.1 has known XSS vulnerabilities",
      fix: "Upgrade to jQuery 3.5.0 or later"
    }
  ],
  patterns: {
    react: { devMode: true, strictMode: false },
    vue: { devToolsEnabled: true },
    jquery: { version: "3.4.1", modern: true }
  },
  recommendations: [...],
  score: 72,
  rating: "needs-improvement",
  timestamp: 1699999999999
}
```

**Framework-Specific Issues:**
| Type | Framework | Severity | Description |
|------|-----------|----------|-------------|
| `react-dev-build` | React | error | Development build in production |
| `vue-v-html` | Vue | warning | Using v-html (potential XSS) |
| `angularjs-legacy` | AngularJS | warning | AngularJS 1.x is end of life |
| `angularjs-bind-html` | AngularJS | error | ng-bind-html usage |
| `jquery-vulnerable` | jQuery | warning | jQuery < 3.5.0 vulnerabilities |

---

## auditFormSecurity

Audit forms for security and validation issues.

```javascript
window.__devtool.auditFormSecurity()
```

**Returns:**
```javascript
{
  forms: [
    {
      selector: "form#login",
      action: "https://example.com/auth",
      method: "post",
      hasValidation: true,
      hasCSRF: true,
      autocomplete: "on",
      fields: [
        { type: "email", name: "email", hasValidation: true },
        { type: "password", name: "password", hasValidation: true, isPassword: true }
      ]
    }
  ],
  issues: [
    {
      type: "missing-csrf",
      severity: "warning",
      message: "POST form without CSRF token",
      fix: "Add CSRF token to prevent cross-site request forgery",
      element: "form#contact"
    },
    {
      type: "sensitive-autocomplete",
      severity: "warning",
      message: "Sensitive field \"credit_card\" should disable autocomplete",
      fix: "Add autocomplete=\"off\" to sensitive fields",
      element: "input#credit_card"
    }
  ],
  summary: {
    total: 3,
    withValidation: 2,
    withAutocomplete: 3,
    withCSRF: 2,
    passwordFields: 1,
    sensitiveFields: 1
  },
  score: 78,
  rating: "needs-improvement",
  timestamp: 1699999999999
}
```

**Issues Detected:**
| Type | Severity | Description |
|------|----------|-------------|
| `missing-csrf` | warning | POST form without CSRF token |
| `insecure-form-action` | error | Form submits to HTTP URL |
| `password-autocomplete` | info | Password without specific autocomplete |
| `sensitive-autocomplete` | warning | Sensitive field without autocomplete=off |

---

## auditExternalResources

Audit external scripts, iframes, and stylesheets for security issues.

```javascript
window.__devtool.auditExternalResources()
```

**Returns:**
```javascript
{
  scripts: [
    {
      src: "https://cdn.example.com/lib.js",
      isExternal: true,
      hasIntegrity: true,
      crossOrigin: "anonymous",
      async: true,
      defer: false
    }
  ],
  iframes: [
    {
      src: "https://embed.example.com/widget",
      sandbox: "allow-scripts allow-same-origin",
      allow: "fullscreen",
      loading: "lazy"
    }
  ],
  stylesheets: [
    { href: "https://cdn.example.com/style.css", hasIntegrity: false }
  ],
  issues: [
    {
      type: "missing-sri",
      severity: "warning",
      message: "External script without Subresource Integrity (SRI)",
      fix: "Add integrity attribute with hash of script content",
      resource: "https://cdn.example.com/analytics.js"
    },
    {
      type: "unsandboxed-iframe",
      severity: "warning",
      message: "External iframe without sandbox attribute",
      fix: "Add sandbox attribute to restrict iframe capabilities",
      resource: "https://ads.example.com/banner"
    }
  ],
  summary: {
    totalScripts: 8,
    externalScripts: 3,
    withIntegrity: 2,
    crossOriginScripts: 3,
    iframes: 2,
    sandboxedIframes: 1
  },
  score: 76,
  rating: "needs-improvement",
  timestamp: 1699999999999
}
```

**Issues Detected:**
| Type | Severity | Description |
|------|----------|-------------|
| `missing-sri` | warning | External script without integrity hash |
| `missing-sri-css` | info | External stylesheet without integrity |
| `unsandboxed-iframe` | warning | External iframe without sandbox |

---

## auditPrototypePollution

Check for prototype pollution vulnerabilities and best practices.

```javascript
window.__devtool.auditPrototypePollution()
```

**Returns:**
```javascript
{
  vulnerable: false,
  tests: [
    {
      name: "Object.prototype frozen",
      passed: false,
      message: "Prototype is not frozen"
    }
  ],
  issues: [
    {
      type: "prototype-not-frozen",
      severity: "info",
      message: "Object.prototype is not frozen",
      fix: "Consider freezing prototypes: Object.freeze(Object.prototype)"
    }
  ],
  recommendations: [
    "Use Object.create(null) for dictionary objects",
    "Validate and sanitize all user input before using as object keys",
    "Consider using Map instead of plain objects for dynamic keys",
    "Freeze Object.prototype in security-critical applications"
  ],
  timestamp: 1699999999999
}
```

---

## auditSecurity

Comprehensive security audit combining all security checks.

```javascript
window.__devtool.auditSecurity()
```

**Returns:**
```javascript
{
  headers: { /* auditSecurityHeaders results */ },
  domSecurity: { /* auditDOMSecurity results */ },
  framework: { /* auditFrameworkQuality results */ },
  forms: { /* auditFormSecurity results */ },
  resources: { /* auditExternalResources results */ },
  prototype: { /* auditPrototypePollution results */ },
  summary: {
    frameworkDetected: "React",
    totalIssues: 12,
    criticalIssues: 2,
    httpsEnabled: true,
    cspEnabled: true
  },
  issues: [ /* all issues from all audits */ ],
  criticalIssues: [ /* only error severity issues */ ],
  overallScore: 75,
  grade: "C",  // A (90+), B (80-89), C (70-79), D (60-69), F (<60)
  timestamp: 1699999999999
}
```

**Score Weights:**
| Component | Weight |
|-----------|--------|
| Security Headers | 25% |
| DOM Security | 25% |
| Form Security | 20% |
| Framework Quality | 15% |
| External Resources | 15% |

---

## loadDOMPurify

Load DOMPurify library from CDN for HTML sanitization.

```javascript
window.__devtool.loadDOMPurify()
```

**Returns:** Promise that resolves with the DOMPurify object.

**Example:**
```javascript
window.__devtool.loadDOMPurify().then(DOMPurify => {
  const clean = DOMPurify.sanitize('<img src=x onerror=alert(1)>')
  console.log(clean)  // '<img src="x">'
})
```

---

## sanitizeHTML

Sanitize HTML using DOMPurify (loads on demand).

```javascript
window.__devtool.sanitizeHTML(dirty, options?)
```

**Parameters:**
- `dirty` (string): HTML string to sanitize
- `options` (object): DOMPurify configuration options

**Returns:** Promise
```javascript
{
  clean: "<p>Safe content</p>",
  removed: [{ element: "script", ... }],
  timestamp: 1699999999999
}
```

**Example:**
```javascript
window.__devtool.sanitizeHTML('<div onclick="alert(1)">Hello</div>')
  .then(result => {
    console.log(result.clean)  // '<div>Hello</div>'
  })

// With options
window.__devtool.sanitizeHTML(html, { ALLOWED_TAGS: ['b', 'i', 'em'] })
```

---

## checkXSSRisk

Check if a string contains potential XSS patterns.

```javascript
window.__devtool.checkXSSRisk(input)
```

**Parameters:**
- `input` (string): String to check for XSS patterns

**Returns:**
```javascript
{
  input: "<script>alert(1)</script>...",
  hasRisk: true,
  risks: [
    { type: "script-tag", severity: "high", pattern: "/<script[\\s>]/i" }
  ],
  highRisk: true,
  timestamp: 1699999999999
}
```

**Patterns Detected:**
| Type | Severity | Description |
|------|----------|-------------|
| `script-tag` | high | `<script>` tags |
| `event-handler` | high | `on*=` event handlers |
| `javascript-url` | high | `javascript:` URLs |
| `data-url` | medium | `data:` URLs with base64 |
| `vbscript-url` | high | `vbscript:` URLs |
| `expression` | medium | CSS expressions |
| `eval` | high | `eval()` calls |
| `document-write` | medium | `document.write` |
| `innerHTML` | medium | `.innerHTML =` |
| `fromCharCode` | low | `fromCharCode` obfuscation |
| `svg-onload` | high | `<svg onload=...>` |
| `img-onerror` | high | `<img onerror=...>` |
| `iframe-src` | medium | `<iframe src=...>` |

---

## Common Patterns

### Pre-Deploy Security Check

```javascript
async function securityCheck() {
  const audit = window.__devtool.auditSecurity()

  const blockers = []

  // Critical issues block deployment
  if (audit.criticalIssues.length > 0) {
    blockers.push(`${audit.criticalIssues.length} critical security issues`)
  }

  // No HTTPS in production
  if (!audit.headers.headers.https.present) {
    blockers.push('HTTPS not enabled')
  }

  // Dev build in production
  if (audit.framework.issues.some(i => i.type === 'react-dev-build')) {
    blockers.push('React development build detected')
  }

  return {
    pass: blockers.length === 0,
    grade: audit.grade,
    score: audit.overallScore,
    blockers: blockers
  }
}
```

### Monitor CSP During Development

```javascript
// Start monitoring
const cspObserver = window.__devtool.observeCSPViolations(violation => {
  console.error('CSP Violation:', {
    blocked: violation.blockedURI,
    directive: violation.violatedDirective,
    location: `${violation.sourceFile}:${violation.lineNumber}`
  })
})

// ... run your application ...

// Stop and get summary
const results = cspObserver.stop()
if (results.count > 0) {
  console.warn(`CSP needs adjustment: ${results.count} violations`)
  console.table(results.violations)
}
```

### User Input Validation

```javascript
function validateUserInput(input) {
  // Check for XSS patterns
  const xssCheck = window.__devtool.checkXSSRisk(input)

  if (xssCheck.highRisk) {
    return { valid: false, reason: 'Potentially malicious content detected' }
  }

  // Sanitize if medium risk
  if (xssCheck.hasRisk) {
    return window.__devtool.sanitizeHTML(input).then(result => ({
      valid: true,
      sanitized: result.clean,
      removed: result.removed.length
    }))
  }

  return { valid: true, value: input }
}
```

### Framework Security Check

```javascript
function checkFrameworkSecurity() {
  const audit = window.__devtool.auditFrameworkQuality()

  console.log(`Primary framework: ${audit.framework.primary}`)
  console.log(`Security score: ${audit.score}/100 (${audit.rating})`)

  // Check for specific issues
  const devBuild = audit.issues.find(i => i.type.includes('dev-build'))
  if (devBuild) {
    console.error('DEV BUILD IN PRODUCTION:', devBuild.details)
  }

  const vulnerableLib = audit.issues.find(i => i.type.includes('vulnerable'))
  if (vulnerableLib) {
    console.warn('VULNERABLE LIBRARY:', vulnerableLib.message)
  }

  // Vue-specific: check v-html usage
  const vHtml = audit.issues.find(i => i.type === 'vue-v-html')
  if (vHtml) {
    console.warn('v-html usage detected:', vHtml.details)
  }

  return audit.issues
}
```

---

## Performance Notes

- `auditSecurityHeaders` is fast (queries meta tags only)
- `auditDOMSecurity` scans all elements - may be slow on large DOMs
- `auditExternalResources` queries all script/iframe/link elements
- `auditSecurity` runs all audits - use individual functions for targeted checks
- `loadDOMPurify` loads ~25KB from CDN on first call (cached after)
- `sanitizeHTML` is async (waits for DOMPurify to load)

## Security Considerations

- These audits detect client-side patterns only
- Server-side headers (HSTS, X-Frame-Options) aren't visible to client JS
- Some issues can only be detected through HTTP response headers
- Use browser DevTools Network tab for complete header inspection
- Consider server-side security scanning for comprehensive coverage

## See Also

- [OWASP XSS Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html)
- [DOMPurify Documentation](https://github.com/cure53/DOMPurify)
- [Content Security Policy - MDN](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
- [CSS Evaluation](/api/frontend/css-evaluation) - CSS architecture auditing
- [Quality Auditing](/api/frontend/quality-auditing) - Performance metrics
