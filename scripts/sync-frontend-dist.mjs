import { cpSync, existsSync, mkdirSync, renameSync, rmSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const source = path.join(root, 'apps/frontend/dist')
const target = path.join(root, 'apps/backend/internal/static/dist')

if (!existsSync(source)) {
  console.error('sync-frontend-dist: missing apps/frontend/dist — run pnpm --filter @connexto/frontend build first')
  process.exit(1)
}

const staging = `${target}.staging`

rmSync(staging, { recursive: true, force: true })
mkdirSync(staging, { recursive: true })
cpSync(source, staging, { recursive: true })
rmSync(target, { recursive: true, force: true })
renameSync(staging, target)
console.log('sync-frontend-dist: copied to apps/backend/internal/static/dist')
