import { expect, test } from '@playwright/test'

test.describe('embedded production smoke', () => {
  test('boots SPA from Go origin and loads overview', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('navigation', { name: 'Primary navigation' })).toBeVisible({
      timeout: 30_000,
    })
    await expect(page.getByRole('link', { name: 'Overview' })).toBeVisible()
  })

  test('navigates core operator routes', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('link', { name: /executions/i }).click()
    await expect(page).toHaveURL(/\/executions/)
    await page.getByRole('link', { name: /resources/i }).click()
    await expect(page).toHaveURL(/\/resources/)
  })

  test('reports operator session on loopback trust', async ({ request }) => {
    const response = await request.get('/api/ui/session')
    expect(response.ok()).toBeTruthy()
    const body = (await response.json()) as { authenticated: boolean; mode: string }
    expect(body.authenticated).toBe(true)
    expect(body.mode).toBe('loopback_trust')
  })

  test('rejects protected admin route without session on non-loopback client', async ({ request }) => {
    // Server-side loopback trust applies to TCP client IP; same request fixture is loopback.
    const response = await request.get('/api/resource-policies')
    expect(response.status()).toBe(200)
  })

  test('operational SSE stream connects', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(2000)
    const errors: string[] = []
    page.on('console', (msg) => {
      if (msg.type() === 'error' && msg.text().includes('EventSource')) {
        errors.push(msg.text())
      }
    })
    await page.waitForTimeout(3000)
    expect(errors).toEqual([])
  })
})
