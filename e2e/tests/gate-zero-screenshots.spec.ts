import { expect, test } from '@playwright/test'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const reportDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../reports/gate-zero-8.1')

const routes = [
  { slug: 'overview', path: '/' },
  { slug: 'providers', path: '/providers' },
  { slug: 'catalog', path: '/catalog' },
  { slug: 'playground', path: '/playground' },
  { slug: 'executions', path: '/executions' },
  { slug: 'resources', path: '/resources' },
] as const

const viewports = [
  { name: '1280', width: 1280, height: 800 },
  { name: '768', width: 768, height: 1024 },
  { name: '390', width: 390, height: 844 },
] as const

test.describe('gate zero visual capture', () => {
  for (const route of routes) {
    for (const viewport of viewports) {
      test(`${route.slug} @ ${viewport.name}px`, async ({ page }) => {
        await page.setViewportSize({ width: viewport.width, height: viewport.height })
        await page.goto(route.path)
        await expect(page.getByRole('navigation', { name: 'Primary navigation' })).toBeVisible({
          timeout: 30_000,
        })
        await page.screenshot({
          path: path.join(reportDir, `${route.slug}-${viewport.name}.png`),
          fullPage: true,
        })
      })
    }
  }
})
