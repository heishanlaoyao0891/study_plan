import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const styles = () => readFile(new URL('../src/styles.css', import.meta.url), 'utf8')

test('admin sidebar width is independent from routed content reflow', async () => {
  const css = await styles()

  assert.match(css, /--admin-sidebar-width:\s*240px/)
  assert.match(css, /\.sidebar\s*\{[^}]*position:\s*fixed/s)
  assert.match(css, /\.sidebar\s*\{[^}]*width:\s*var\(--admin-sidebar-width\)/s)
  assert.match(css, /\.sidebar\s*\{[^}]*flex-basis:\s*var\(--admin-sidebar-width\)/s)
  assert.match(css, /\.content\s*\{[^}]*margin-left:\s*var\(--admin-sidebar-width\)/s)
  assert.match(css, /\.content\s*\{[^}]*width:\s*calc\(100% - var\(--admin-sidebar-width\)\)/s)
})
