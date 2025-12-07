---
sidebar_position: 11
---

# Layout Robustness & Fragility Detection

Functions for detecting layout fragility, text overflow issues, responsive risks, and performance problems.

## checkTextFragility

Detect text truncation, overflow, and font size issues that may cause content loss.

```javascript
window.__devtool.checkTextFragility(selector?)
```

**Parameters:**
- `selector` (string, optional): CSS selector to scope the check. Defaults to entire page.

**Returns:**
```javascript
{
  issues: [
    {
      selector: ".card-title",
      type: "truncation-ellipsis",
      severity: "error",
      message: "Text is truncated with ellipsis - content loss (WCAG 1.4.10)",
      details: {
        scrollWidth: 250,
        clientWidth: 150,
        excess: 100,
        textContent: "This is the full text that gets...",
        overflowStyle: "hidden",
        textOverflow: "ellipsis"
      },
      wcag: "1.4.10",
      fix: "Allow text to wrap or expand container; avoid text-overflow: ellipsis"
    }
  ],
  summary: {
    truncations: 2,
    overflows: 1,
    nowrapRisks: 0,
    viewportFonts: 1,
    fixedHeights: 0,
    lineHeightIssues: 0,
    errors: 2,
    warnings: 2,
    total: 4
  },
  timestamp: 1699999999999
}
```

**Issue Types:**
| Type | Severity | Description |
|------|----------|-------------|
| `truncation-ellipsis` | error | Text truncated with ellipsis (WCAG 1.4.10 failure) |
| `overflow-clipped` | warning | Text clipped by `overflow: hidden` |
| `nowrap-risk` | warning | `white-space: nowrap` may cause overflow |
| `viewport-font` | warning | Font uses viewport units (`vw`, `vh`) - may become unreadable |
| `fixed-height-text` | warning | Fixed height container may clip variable text |
| `tight-line-height` | info | Line-height < 1.1 may cause text overlap |

**Example:**
```javascript
const fragility = window.__devtool.checkTextFragility()
if (fragility.summary.errors > 0) {
  console.log('Critical text issues found!')
  fragility.issues
    .filter(i => i.severity === 'error')
    .forEach(i => console.log(i.selector, i.message))
}
```

---

## checkResponsiveRisk

Detect elements that may break at different viewport sizes.

```javascript
window.__devtool.checkResponsiveRisk(selector?)
```

**Parameters:**
- `selector` (string, optional): CSS selector to scope the check.

**Returns:**
```javascript
{
  issues: [
    {
      selector: ".product-card",
      type: "fixed-width-in-fluid",
      severity: "warning",
      message: "Fixed pixel width in flexible container",
      details: {
        elementWidth: "400px",
        parentDisplay: "flex",
        breaksAtViewport: "432px"
      },
      breakpoints: {
        "320px": true,
        "375px": true,
        "768px": false
      },
      fix: "Use max-width: 100% or responsive units (%, vw, clamp())"
    }
  ],
  breakpoints: {
    "320px": { issues: [".product-card"], willOverflow: true },
    "375px": { issues: [".product-card"], willOverflow: true },
    "768px": { issues: [], willOverflow: false },
    "1024px": { issues: [], willOverflow: false }
  },
  summary: {
    fixedWidthIssues: 1,
    unboundedImages: 2,
    flexWrapRisks: 0,
    gridIssues: 0,
    viewportOverflows: 0,
    horizontalScrolls: 0,
    errors: 0,
    warnings: 3,
    total: 3
  },
  currentViewport: 1440,
  timestamp: 1699999999999
}
```

**Issue Types:**
| Type | Severity | Description |
|------|----------|-------------|
| `fixed-width-in-fluid` | warning | Fixed pixel width in flex/grid/fluid container |
| `unbounded-image` | warning | Image without `max-width` constraint |
| `flex-nowrap-overflow` | warning | Flex container with `nowrap` may overflow |
| `grid-fixed-columns` | warning | Grid with fixed pixel columns |
| `viewport-overflow` | error | Element extends beyond viewport |
| `horizontal-scroll` | error/info | Causes horizontal scroll |

**Example:**
```javascript
const risks = window.__devtool.checkResponsiveRisk()

// Check mobile compatibility
if (risks.breakpoints['320px'].willOverflow) {
  console.log('Issues on mobile:', risks.breakpoints['320px'].issues)
}
```

---

## capturePerformanceMetrics

Capture comprehensive performance metrics including CLS, long tasks, and resource timing.

```javascript
window.__devtool.capturePerformanceMetrics()
```

**Returns:**
```javascript
{
  cls: {
    score: 0.15,
    rating: "needs-improvement",  // "good" | "needs-improvement" | "poor"
    shifts: [
      {
        value: 0.08,
        startTime: 1523,
        sources: [".ad-banner", ".hero-image"]
      }
    ]
  },
  longTasks: [
    { duration: 85, startTime: 234, name: "self" }
  ],
  resources: {
    byType: {
      script: { count: 12, totalSize: 450000, totalDuration: 1200 },
      img: { count: 25, totalSize: 1200000, totalDuration: 2500 },
      css: { count: 5, totalSize: 85000, totalDuration: 300 }
    },
    largest: [
      { url: "/images/hero.jpg", type: "img", size: 450000, duration: 800 }
    ],
    slowest: [
      { url: "/api/data", type: "fetch", size: 5000, duration: 1200 }
    ],
    renderBlocking: [
      { url: "/styles.css", type: "css", size: 45000, duration: 200 }
    ]
  },
  paint: {
    firstPaint: 450,
    firstContentfulPaint: 620
  },
  totals: {
    pageWeight: 1855000,
    resourceCount: 46,
    loadTime: 2800,
    domContentLoaded: 1200
  },
  timestamp: 1699999999999
}
```

**CLS Rating Thresholds:**
- `good`: < 0.1
- `needs-improvement`: 0.1 - 0.25
- `poor`: > 0.25

**Example:**
```javascript
const perf = window.__devtool.capturePerformanceMetrics()

console.log(`Page weight: ${(perf.totals.pageWeight / 1024 / 1024).toFixed(2)} MB`)
console.log(`Load time: ${perf.totals.loadTime}ms`)

if (perf.cls && perf.cls.rating !== 'good') {
  console.log('CLS issues:', perf.cls.shifts)
}
```

---

## runAxeAudit

Run a full accessibility audit using axe-core (dynamically loaded from CDN).

```javascript
window.__devtool.runAxeAudit(options?)
```

**Parameters:**
- `options.selector` (string): Scope audit to specific element
- `options.runOnly` (array): WCAG levels to test (default: `['wcag2a', 'wcag2aa', 'best-practice']`)

**Returns:** Promise
```javascript
{
  violations: [
    {
      id: "color-contrast",
      impact: "serious",
      description: "Elements must have sufficient color contrast",
      help: "Elements must have sufficient color contrast",
      helpUrl: "https://dequeuniversity.com/rules/axe/4.10/color-contrast",
      tags: ["wcag2aa", "wcag143"],
      nodes: [
        {
          selector: ".light-text",
          html: "<span class='light-text'>Hard to read</span>",
          failureSummary: "Fix: ensure contrast ratio is at least 4.5:1",
          impact: "serious"
        }
      ]
    }
  ],
  passes: 45,
  incomplete: 3,
  inapplicable: 12,
  summary: {
    critical: 0,
    serious: 2,
    moderate: 5,
    minor: 3,
    totalViolations: 10,
    totalNodes: 15
  },
  score: 78,
  testEngine: {
    name: "axe-core",
    version: "4.10.0"
  },
  timestamp: 1699999999999
}
```

**Example:**
```javascript
// Full page audit
window.__devtool.runAxeAudit().then(results => {
  console.log(`Accessibility score: ${results.score}/100`)
  console.log(`Violations: ${results.summary.totalViolations}`)

  results.violations.forEach(v => {
    console.log(`[${v.impact}] ${v.id}: ${v.help}`)
  })
})

// Audit specific component
window.__devtool.runAxeAudit({ selector: '.modal' }).then(results => {
  console.log('Modal accessibility:', results.score)
})
```

---

## auditLayoutRobustness

Comprehensive layout audit combining all analysis functions with scoring.

```javascript
window.__devtool.auditLayoutRobustness(options?)
```

**Parameters:**
- `options.selector` (string): Scope audit to specific element
- `options.includeAxe` (boolean): Include full axe-core audit (loads ~300KB)

**Returns:** Promise
```javascript
{
  textFragility: { /* checkTextFragility results */ },
  responsiveRisk: { /* checkResponsiveRisk results */ },
  layoutIssues: { /* diagnoseLayout results */ },
  accessibility: { /* auditAccessibility results */ },
  performance: { /* capturePerformanceMetrics results */ },

  scores: {
    text: 85,
    responsive: 70,
    layout: 95,
    accessibility: 78,
    performance: 82,
    overall: 80
  },
  grade: "B",  // A (90+), B (80-89), C (70-79), D (60-69), F (<60)

  criticalIssues: [
    {
      category: "text",
      type: "truncation-ellipsis",
      selector: ".card-title",
      message: "Text is truncated with ellipsis - content loss (WCAG 1.4.10)",
      fix: "Allow text to wrap or expand container"
    }
  ],

  recommendations: [
    {
      priority: 1,
      category: "responsive",
      issue: "2 element(s) cause horizontal scroll",
      impact: "Poor mobile experience, content may be inaccessible",
      fix: "Constrain element widths with max-width: 100%"
    }
  ],

  timestamp: 1699999999999
}
```

**Example:**
```javascript
// Quick audit (no axe-core)
window.__devtool.auditLayoutRobustness().then(audit => {
  console.log(`Grade: ${audit.grade} (${audit.scores.overall}/100)`)

  if (audit.criticalIssues.length > 0) {
    console.log('Critical issues to fix:')
    audit.criticalIssues.forEach(i => console.log(`  - ${i.message}`))
  }

  console.log('Top recommendations:')
  audit.recommendations.slice(0, 3).forEach(r => {
    console.log(`  ${r.priority}. [${r.category}] ${r.issue}`)
  })
})

// Full audit with axe-core
window.__devtool.auditLayoutRobustness({ includeAxe: true }).then(audit => {
  console.log('Full accessibility score:', audit.axeAudit?.score)
})
```

---

## observeLayoutShifts

Start real-time observation of layout shifts (CLS).

```javascript
window.__devtool.observeLayoutShifts(callback?)
```

**Parameters:**
- `callback` (function): Called on each layout shift with `(entry, cumulativeCLS)`

**Returns:**
```javascript
{
  stop: function() { /* Returns { finalCLS, entries } */ },
  getCurrent: function() { /* Returns { cls, entryCount } */ }
}
```

**Example:**
```javascript
// Start observing
const observer = window.__devtool.observeLayoutShifts((entry, cumulative) => {
  console.log(`Shift: ${entry.value}, Total CLS: ${cumulative}`)
  console.log('Elements that shifted:', entry.sources)
})

// Check current value
console.log(observer.getCurrent())

// Stop and get final results
const results = observer.stop()
console.log(`Final CLS: ${results.finalCLS}`)
```

---

## observeLongTasks

Start real-time observation of long tasks (>50ms main thread blocks).

```javascript
window.__devtool.observeLongTasks(callback?)
```

**Parameters:**
- `callback` (function): Called on each long task with `(entry)`

**Returns:**
```javascript
{
  stop: function() { /* Returns { tasks, count } */ },
  getCurrent: function() { /* Returns { taskCount, totalBlocking } */ }
}
```

**Example:**
```javascript
// Start observing
const observer = window.__devtool.observeLongTasks(task => {
  console.log(`Long task: ${task.duration}ms at ${task.startTime}ms`)
})

// Trigger some heavy operation
someHeavyFunction()

// Check results
const results = observer.stop()
console.log(`Total blocking time: ${results.tasks.reduce((sum, t) => sum + t.duration, 0)}ms`)
```

---

## Common Patterns

### Pre-Deploy Quality Check

```javascript
async function preDeployCheck() {
  const audit = await window.__devtool.auditLayoutRobustness({ includeAxe: true })

  const issues = []

  // Must pass
  if (audit.grade === 'F' || audit.grade === 'D') {
    issues.push(`Overall grade too low: ${audit.grade}`)
  }

  // No critical text issues
  if (audit.textFragility.summary.errors > 0) {
    issues.push(`${audit.textFragility.summary.errors} text truncation errors`)
  }

  // No viewport overflow
  if (audit.responsiveRisk.summary.viewportOverflows > 0) {
    issues.push(`${audit.responsiveRisk.summary.viewportOverflows} elements overflow viewport`)
  }

  // Accessibility baseline
  if (audit.axeAudit && audit.axeAudit.summary.critical > 0) {
    issues.push(`${audit.axeAudit.summary.critical} critical accessibility issues`)
  }

  return {
    pass: issues.length === 0,
    grade: audit.grade,
    score: audit.scores.overall,
    issues: issues
  }
}

preDeployCheck().then(result => {
  console.log(result.pass ? 'PASS' : 'FAIL', result)
})
```

### Mobile Compatibility Check

```javascript
function checkMobileReady() {
  const risks = window.__devtool.checkResponsiveRisk()

  const mobileIssues = risks.issues.filter(i =>
    i.breakpoints && (i.breakpoints['320px'] || i.breakpoints['375px'])
  )

  console.log('Mobile issues:', mobileIssues.length)
  mobileIssues.forEach(i => {
    window.__devtool.highlight(i.selector, { color: 'rgba(255,0,0,0.3)' })
    console.log(i.selector, i.message)
  })

  return {
    mobileReady: mobileIssues.length === 0,
    issues: mobileIssues
  }
}
```

### Content Loss Detection

```javascript
function findContentLoss() {
  const fragility = window.__devtool.checkTextFragility()

  // Elements losing content due to truncation
  const contentLoss = fragility.issues.filter(i =>
    i.type === 'truncation-ellipsis' || i.type === 'overflow-clipped'
  )

  contentLoss.forEach(i => {
    console.log(`Content lost at: ${i.selector}`)
    console.log(`  Original: "${i.details.textContent}"`)
    console.log(`  Visible width: ${i.details.clientWidth}px`)
    console.log(`  Full width: ${i.details.scrollWidth}px`)
    console.log(`  Fix: ${i.fix}`)
  })

  return contentLoss
}
```

---

## Performance Notes

- `checkTextFragility` and `checkResponsiveRisk` scan all elements - may be slow on large pages
- `runAxeAudit` loads ~300KB axe-core on first call (cached after)
- `auditLayoutRobustness` combines multiple scans - use sparingly
- Use `selector` parameter to scope checks to specific areas
- Real-time observers (`observeLayoutShifts`, `observeLongTasks`) are lightweight

## See Also

- [Layout Diagnostics](/api/frontend/layout-diagnostics) - Basic overflow and stacking detection
- [Accessibility](/api/frontend/accessibility) - Built-in a11y checks
- [Performance Monitoring](/use-cases/performance-monitoring) - Performance use cases
