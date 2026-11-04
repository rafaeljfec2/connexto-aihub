import { expect, test } from '@playwright/test'

const operatorPassphrase =
  process.env.AIHUB_OPERATOR_PASSPHRASE ?? 'e2e-operator-passphrase-value-minimum'

test.describe('operator passphrase session', () => {
  test('redirects to login and rejects protected API without session', async ({ page, request }) => {
    await page.goto('/')
    await expect(page).toHaveURL(/\/login/)
    await expect(page.getByRole('heading', { name: 'Operator sign-in' })).toBeVisible()

    const protectedResponse = await request.get('/api/resource-policies')
    expect(protectedResponse.status()).toBe(401)
  })

  test('rejects invalid passphrase', async ({ page }) => {
    await page.goto('/login')
    await page.getByLabel('Passphrase').fill('wrong-passphrase-value')
    await page.getByRole('button', { name: 'Sign in' }).click()
    await expect(page.getByText('Invalid passphrase')).toBeVisible()
  })

  test('full operator journey with HttpOnly session cookie', async ({ page, context, request }) => {
    await page.goto('/login')
    await page.getByLabel('Passphrase').fill(operatorPassphrase)
    await page.getByRole('button', { name: 'Sign in' }).click()
    await expect(page).toHaveURL('/')
    await expect(page.getByRole('navigation', { name: 'Primary navigation' })).toBeVisible()

    const cookies = await context.cookies()
    const sessionCookie = cookies.find((cookie) => cookie.name === 'aihub_ui_session')
    expect(sessionCookie).toBeDefined()
    expect(sessionCookie?.httpOnly).toBe(true)

    const sessionResponse = await request.get('/api/ui/session')
    expect(sessionResponse.ok()).toBeTruthy()
    const sessionBody = (await sessionResponse.json()) as { authenticated: boolean; mode: string }
    expect(sessionBody.authenticated).toBe(true)
    expect(sessionBody.mode).toBe('passphrase')

    await page.getByRole('link', { name: /executions/i }).click()
    await expect(page).toHaveURL(/\/executions/)
    await page.getByRole('link', { name: /resources/i }).click()
    await expect(page).toHaveURL(/\/resources/)

    await page.getByRole('button', { name: 'Sign out' }).click()
    await expect(page).toHaveURL(/\/login/)

    const afterLogout = await request.get('/api/resource-policies')
    expect(afterLogout.status()).toBe(401)
  })
})
