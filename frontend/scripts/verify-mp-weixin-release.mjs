import { isIP } from 'node:net'
import { readFile, readdir, stat } from 'node:fs/promises'
import { resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

import { parse } from 'jsonc-parser'

export const CANONICAL_ORIGIN = 'https://slls.asia'
export const EXPECTED_APP_ID = 'wx985c473e161501fc'
export const MIN_VERSION_NAME = '1.0.1'
export const MIN_VERSION_CODE = 101

const scriptDir = fileURLToPath(new URL('.', import.meta.url))
const frontendDir = resolve(scriptDir, '..')

function parseEnv(source) {
  return Object.fromEntries(
    source
      .split(/\r?\n/)
      .map(line => line.trim())
      .filter(line => line && !line.startsWith('#'))
      .map(line => {
        const separator = line.indexOf('=')
        return separator < 0 ? [line, ''] : [line.slice(0, separator), line.slice(separator + 1)]
      }),
  )
}

function compareVersions(left, right) {
  const normalize = value => String(value).split('.').map(part => Number(part))
  const leftParts = normalize(left)
  const rightParts = normalize(right)
  const length = Math.max(leftParts.length, rightParts.length)
  for (let index = 0; index < length; index += 1) {
    const difference = (leftParts[index] || 0) - (rightParts[index] || 0)
    if (difference !== 0) return difference
  }
  return 0
}

export function validateReleaseConfig(envSource, manifestSource) {
  const errors = []
  const env = parseEnv(envSource)
  const manifestErrors = []
  const manifest = parse(manifestSource, manifestErrors)
  if (manifestErrors.length > 0 || !manifest) errors.push('frontend/src/manifest.json is not valid JSONC')

  let apiOrigin
  try {
    apiOrigin = new URL(env.VITE_API_BASE || '')
  } catch {
    errors.push('VITE_API_BASE must be an absolute URL')
  }
  if (apiOrigin) {
    if (apiOrigin.origin !== CANONICAL_ORIGIN) errors.push(`VITE_API_BASE must equal ${CANONICAL_ORIGIN}`)
    if (apiOrigin.protocol !== 'https:') errors.push('VITE_API_BASE must use HTTPS')
    if (isIP(apiOrigin.hostname)) errors.push('VITE_API_BASE must not use an IP literal')
    if (apiOrigin.pathname !== '/' || apiOrigin.search || apiOrigin.hash) errors.push('VITE_API_BASE must not include a path, query, or fragment')
    if (apiOrigin.username || apiOrigin.password) errors.push('VITE_API_BASE must not include credentials')
  }
  if (env.VITE_ENABLE_DEV_LOGIN !== 'false') errors.push('VITE_ENABLE_DEV_LOGIN must be false')

  const appId = manifest?.['mp-weixin']?.appid
  if (appId !== EXPECTED_APP_ID) errors.push(`mp-weixin.appid must equal ${EXPECTED_APP_ID}`)
  if (compareVersions(manifest?.versionName || '0', MIN_VERSION_NAME) < 0) {
    errors.push(`versionName must be at least ${MIN_VERSION_NAME}`)
  }
  if (!Number.isInteger(Number(manifest?.versionCode)) || Number(manifest?.versionCode) < MIN_VERSION_CODE) {
    errors.push(`versionCode must be an integer of at least ${MIN_VERSION_CODE}`)
  }

  if (errors.length > 0) throw new Error(errors.join('\n'))
  return { appId, versionName: manifest.versionName, versionCode: Number(manifest.versionCode) }
}

async function listFiles(directory) {
  const entries = await readdir(directory)
  const files = []
  for (const entry of entries) {
    const path = resolve(directory, entry)
    const details = await stat(path)
    if (details.isDirectory()) files.push(...await listFiles(path))
    else files.push(path)
  }
  return files
}

export async function validateArtifact(artifactDir) {
  const projectConfig = JSON.parse(await readFile(resolve(artifactDir, 'project.config.json'), 'utf8'))
  if (projectConfig.appid !== EXPECTED_APP_ID) throw new Error(`generated project AppID must equal ${EXPECTED_APP_ID}`)

  const sourceFiles = (await listFiles(artifactDir)).filter(path => /\.(js|json|wxml)$/.test(path))
  const contents = await Promise.all(sourceFiles.map(path => readFile(path, 'utf8')))
  if (!contents.some(content => content.includes(CANONICAL_ORIGIN))) {
    throw new Error(`generated artifact does not contain ${CANONICAL_ORIGIN}`)
  }
  return { fileCount: sourceFiles.length }
}

async function main() {
  const mode = process.argv[2] || '--config'
  const envSource = await readFile(resolve(frontendDir, '.env.production'), 'utf8')
  const manifestSource = await readFile(resolve(frontendDir, 'src/manifest.json'), 'utf8')
  const config = validateReleaseConfig(envSource, manifestSource)
  if (mode === '--artifact') {
    const artifact = await validateArtifact(resolve(frontendDir, 'dist/build/mp-weixin'))
    console.log(`WeChat release artifact verified: ${config.versionName} (${config.versionCode}), ${artifact.fileCount} files checked`)
    return
  }
  if (mode !== '--config') throw new Error(`unknown mode: ${mode}`)
  console.log(`WeChat release config verified: ${config.versionName} (${config.versionCode}), ${CANONICAL_ORIGIN}`)
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main().catch(error => {
    console.error(error.message)
    process.exitCode = 1
  })
}
