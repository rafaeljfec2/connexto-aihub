#!/usr/bin/env node
/**
 * Syncs the MIT-licensed gokuscraper/ai-icons SVG set into
 * apps/frontend/vendor/ai-icons for offline Vite embedding.
 *
 * Usage:
 *   node scripts/sync-ai-icons.mjs            # sync pinned SHA from VERSION (or DEFAULT_SHA)
 *   node scripts/sync-ai-icons.mjs <sha|ref>  # sync a specific commit / branch / tag
 */
import { createWriteStream, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { cp, mkdtemp, rename } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { pipeline } from 'node:stream/promises'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'
import { Readable } from 'node:stream'

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const VENDOR_DIR = path.join(ROOT, 'apps/frontend/vendor/ai-icons')
const VERSION_PATH = path.join(VENDOR_DIR, 'VERSION')
const REPO = 'gokuscraper/ai-icons'
const DEFAULT_SHA = 'e321829544d26b959ffd9cf9cca63b688e6d755f'

function readPinnedSha() {
  if (!existsSync(VERSION_PATH)) {
    return DEFAULT_SHA
  }
  const text = readFileSync(VERSION_PATH, 'utf8')
  const match = text.match(/^sha:\s*([0-9a-f]{7,40})\s*$/im)
  return match?.[1] ?? DEFAULT_SHA
}

function resolveRef() {
  const arg = process.argv[2]?.trim()
  return arg && arg.length > 0 ? arg : readPinnedSha()
}

async function downloadTarball(ref, destTarGz) {
  const url = `https://codeload.github.com/${REPO}/tar.gz/${ref}`
  const response = await fetch(url)
  if (!response.ok || !response.body) {
    throw new Error(`Failed to download ${url}: HTTP ${response.status}`)
  }
  await pipeline(Readable.fromWeb(response.body), createWriteStream(destTarGz))
}

function extractTarGz(tarGz, destDir) {
  mkdirSync(destDir, { recursive: true })
  execFileSync('tar', ['-xzf', tarGz, '-C', destDir], { stdio: 'inherit' })
}

function findExtractedRoot(staging) {
  const entries = execFileSync('ls', [staging], { encoding: 'utf8' })
    .trim()
    .split('\n')
    .filter(Boolean)
  if (entries.length !== 1) {
    throw new Error(`Expected a single top-level directory in tarball, got: ${entries.join(', ')}`)
  }
  return path.join(staging, entries[0])
}

async function main() {
  const ref = resolveRef()
  console.log(`sync-ai-icons: fetching ${REPO}@${ref}`)

  const work = await mkdtemp(path.join(tmpdir(), 'ai-icons-sync-'))
  const tarGz = path.join(work, 'ai-icons.tar.gz')
  const extractDir = path.join(work, 'extract')

  try {
    await downloadTarball(ref, tarGz)
    extractTarGz(tarGz, extractDir)
    const sourceRoot = findExtractedRoot(extractDir)

    const required = ['svgs', 'icon-toc.json']
    for (const name of required) {
      const p = path.join(sourceRoot, name)
      if (!existsSync(p)) {
        throw new Error(`Upstream archive missing required path: ${name}`)
      }
    }

    const stagingVendor = path.join(work, 'vendor-staging')
    mkdirSync(stagingVendor, { recursive: true })
    await cp(path.join(sourceRoot, 'svgs'), path.join(stagingVendor, 'svgs'), { recursive: true })
    await cp(path.join(sourceRoot, 'icon-toc.json'), path.join(stagingVendor, 'icon-toc.json'))

    const licenseSrc = path.join(sourceRoot, 'LICENSE')
    if (existsSync(licenseSrc)) {
      await cp(licenseSrc, path.join(stagingVendor, 'LICENSE'))
    } else {
      writeFileSync(
        path.join(stagingVendor, 'LICENSE'),
        [
          'MIT License',
          '',
          'Copyright (c) contributors of https://github.com/gokuscraper/ai-icons',
          '',
          'Permission is hereby granted, free of charge, to any person obtaining a copy',
          'of this software and associated documentation files (the "Software"), to deal',
          'in the Software without restriction, including without limitation the rights',
          'to use, copy, modify, merge, publish, distribute, sublicense, and/or sell',
          'copies of the Software, and to permit persons to whom the Software is',
          'furnished to do so, subject to the following conditions:',
          '',
          'The above copyright notice and this permission notice shall be included in all',
          'copies or substantial portions of the Software.',
          '',
          'THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR',
          'IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,',
          'FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE',
          'AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER',
          'LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,',
          'OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE',
          'SOFTWARE.',
          '',
        ].join('\n'),
      )
    }

    // Resolve the commit SHA when a branch/tag was requested.
    let sha = ref
    if (!/^[0-9a-f]{40}$/i.test(ref)) {
      const meta = await fetch(`https://api.github.com/repos/${REPO}/commits/${encodeURIComponent(ref)}`)
      if (meta.ok) {
        const body = await meta.json()
        if (typeof body.sha === 'string') {
          sha = body.sha
        }
      }
    }

    writeFileSync(
      path.join(stagingVendor, 'VERSION'),
      [
        `repo: https://github.com/${REPO}`,
        `sha: ${sha}`,
        `synced_at: ${new Date().toISOString()}`,
        `source_ref: ${ref}`,
        '',
      ].join('\n'),
    )

    mkdirSync(path.dirname(VENDOR_DIR), { recursive: true })
    rmSync(VENDOR_DIR, { recursive: true, force: true })
    await rename(stagingVendor, VENDOR_DIR)

    const svgCount = execFileSync('bash', ['-lc', `ls -1 "${path.join(VENDOR_DIR, 'svgs')}" | wc -l`], {
      encoding: 'utf8',
    }).trim()
    console.log(`sync-ai-icons: wrote ${VENDOR_DIR} (${svgCount} SVG files, sha=${sha})`)
  } finally {
    rmSync(work, { recursive: true, force: true })
  }
}

main().catch((error) => {
  console.error(`sync-ai-icons: ${error instanceof Error ? error.message : String(error)}`)
  process.exit(1)
})
