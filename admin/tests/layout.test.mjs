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

test('user directory exposes account identities and safe management actions', async () => {
  const source = await view('UsersView.vue')

  assert.match(source, /搜索登录名、昵称或 OpenID/)
  assert.match(source, /const status = ref\('active'\)/)
  assert.match(source, /<option value="all">全部展示<\/option>/)
  assert.match(source, /<option value="banned">已封禁<\/option>/)
  assert.match(source, /<option value="deleted">已删除<\/option>/)
  assert.match(source, /<th>登录名<\/th><th>昵称<\/th><th>OpenID<\/th>/)
  assert.match(source, /\{\{ user\.username \|\| '-' \}\}/)
  assert.match(source, /@click="openCreate">添加用户<\/button>/)
  assert.match(source, /AdminApi\.createUser\(createForm\)/)
  assert.match(source, /随机初始密码，仅在本次创建成功后显示一次/)
  assert.match(source, /AdminApi\.deleteUser\(user\.id\)/)
  assert.match(source, /确认删除 \$\{label\}？该操作会清理学习数据并无法恢复。/)
  assert.match(source, /user\.role !== 'admin' && user\.account_status !== 'deleted'/)
  assert.match(source, /<table class="data-table user-table">/)
  assert.match(source, /\.user-table th:last-child,\.user-table td\.actions\{width:122px\}/)
  assert.match(source, /\.actions\{display:grid;grid-template-columns:32px 42px;align-items:center;column-gap:10px;white-space:nowrap\}/)
})
