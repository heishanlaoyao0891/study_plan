import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const styles = () => readFile(new URL('../src/styles.css', import.meta.url), 'utf8')
const view = path => readFile(new URL(`../src/views/${path}`, import.meta.url), 'utf8')

test('admin sidebar width is independent from routed content reflow', async () => {
  const css = await styles()

  assert.match(css, /--admin-sidebar-width:\s*240px/)
  assert.match(css, /\.sidebar\s*\{[^}]*position:\s*fixed/s)
  assert.match(css, /\.sidebar\s*\{[^}]*width:\s*var\(--admin-sidebar-width\)/s)
  assert.match(css, /\.sidebar\s*\{[^}]*flex-basis:\s*var\(--admin-sidebar-width\)/s)
  assert.match(css, /\.content\s*\{[^}]*margin-left:\s*var\(--admin-sidebar-width\)/s)
  assert.match(css, /\.content\s*\{[^}]*width:\s*calc\(100% - var\(--admin-sidebar-width\)\)/s)
})

test('AI invocation refresh also refreshes aggregate token metrics', async () => {
  const source = await view('AIConfigView.vue')

  assert.match(source, /async function loadInvocations\([^)]*\)[\s\S]*AdminApi\.aiMetrics\(\)/)
  assert.match(source, /metrics\.value = planningMetrics/)
})

test('AI invocation history exposes pagination and uses full workspace width', async () => {
  const source = await view('AIConfigView.vue')

  assert.match(source, /const invocationPage = ref\(1\)/)
  assert.match(source, /const invocationPageSize = ref\(20\)/)
  assert.match(source, /const invocationTotal = ref\(0\)/)
  assert.match(source, /const invocationTotalPages = computed\(/)
  assert.match(source, /AdminApi\.aiInvocations\(\{ page, size: invocationPageSize\.value/)
  assert.match(source, /queryInvocations/)
  assert.match(source, /changeInvocationPageSize/)
  assert.match(source, /共 \{\{ invocationTotal \}\} 条 · 第 \{\{ invocationPage \}\} \/ \{\{ invocationTotalPages \}\} 页/)
  assert.match(source, /:disabled="invocationPage <= 1/)
  assert.match(source, /:disabled="invocationPage >= invocationTotalPages/)
  assert.match(source, /\.invocation-panel \{[^}]*width:100%;[^}]*max-width:none/s)
  assert.match(source, /\.invocation-table \{ table-layout:fixed;/)
  assert.doesNotMatch(source, /min-width:1100px/)
})
