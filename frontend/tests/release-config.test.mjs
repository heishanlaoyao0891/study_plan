import assert from 'node:assert/strict'
import test from 'node:test'

import { EXPECTED_APP_ID, validateReleaseConfig } from '../scripts/verify-mp-weixin-release.mjs'

function manifest(overrides = {}) {
  return JSON.stringify({
    versionName: '1.0.1',
    versionCode: '101',
    'mp-weixin': { appid: EXPECTED_APP_ID },
    ...overrides,
  })
}

test('accepts the canonical HTTPS mini-program release configuration', () => {
  const result = validateReleaseConfig(
    'VITE_API_BASE=https://slls.asia\nVITE_ENABLE_DEV_LOGIN=false\n',
    manifest(),
  )

  assert.equal(result.versionCode, 101)
})

test('rejects IP, insecure, development-overridable, and stale releases', () => {
  assert.throws(
    () => validateReleaseConfig(
      'VITE_API_BASE=http://124.223.6.26\nVITE_ENABLE_DEV_LOGIN=true\n',
      manifest({ versionName: '1.0.0', versionCode: '100' }),
    ),
    /VITE_API_BASE must equal https:\/\/slls\.asia[\s\S]*HTTPS[\s\S]*IP literal[\s\S]*VITE_ENABLE_DEV_LOGIN[\s\S]*versionName[\s\S]*versionCode/,
  )
})
