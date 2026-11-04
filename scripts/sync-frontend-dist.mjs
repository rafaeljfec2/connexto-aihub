import { cpSync, mkdirSync, rmSync, existsSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const source = path.join(root, 'apps/frontend/dist')
const target = path.join(root, 'internal/static/dist')

if (!existsSync(source)) {
  console.error('sync-frontend-dist: missing apps/frontend/dist — run pnpm --filter @connexto/frontend build first')
  process.exit(1)
}

rmSync(target, { recursive: true, force: true })
mkdirSync(target, { recursive: true })
cpSync(source, target, { recursive: true })
console.log('sync-frontend-dist: copied to internal/static/dist')
