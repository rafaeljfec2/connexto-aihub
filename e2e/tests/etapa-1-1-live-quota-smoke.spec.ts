import path from 'node:path'
import { expect, test, type Page } from '@playwright/test'

const quotaEndpoint = process.env.AIHUB_E2E_QUOTA_API_URL ?? ''
const useLiveProxy = process.env.AIHUB_E2E_LIVE_PROXY === '1'
const displayedFields = ['used', 'limit', 'remaining'] as const

type QuotaMetric = Record<string, unknown> & { metric: string; unit?: string }
type QuotaSnapshot = {
  account_id: string
  scope: string
  status: string
  quotas: QuotaMetric[]
}

const sensitiveFieldNames = new Set([
  'credential',
  'managementcredential',
  'managementapikey',
  'managementkey',
  'apikey',
  'providerapikey',
  'inferenceapikey',
  'apitoken',
  'authorization',
  'accesstoken',
  'refreshtoken',
  'bearertoken',
  'token',
  'secret',
  'secretkey',
  'privatekey',
  'rawbody',
  'rawpayload',
  'providerpayload',
  'providerresponse',
])

function isSensitiveKey(key: string): boolean {
  const normalized = key.replace(/[^a-z0-9]/gi, '').toLowerCase()
  return sensitiveFieldNames.has(normalized)
}

function sensitiveKeys(value: unknown): string[] {
  if (Array.isArray(value)) return value.flatMap(sensitiveKeys)
  if (typeof value !== 'object' || value === null) return []
  return Object.entries(value).flatMap(([key, nested]) => [
    ...(isSensitiveKey(key) ? [key] : []),
    ...sensitiveKeys(nested),
  ])
}

function formatProviderValue(value: number, unit?: string, scope?: string, metricName?: string): string {
  const isCount = unit === 'requests'
  const isAccountUsage = scope === 'account' && metricName === 'total_usage' && unit === 'USD'
  const formatted = value.toLocaleString('pt-BR', {
    minimumFractionDigits: isCount ? 0 : 2,
    maximumFractionDigits: isCount ? 0 : isAccountUsage ? 2 : 20,
  })
  return unit === 'USD' ? `US$ ${formatted}` : unit ? `${formatted} ${unit}` : formatted
}

function expectedAvailableBalance(snapshot: QuotaSnapshot): string {
  const credits = snapshot.quotas.find((metric) => metric.metric === 'total_credits')
  const usage = snapshot.quotas.find((metric) => metric.metric === 'total_usage')
  if (
    typeof credits?.limit !== 'number'
    || typeof usage?.used !== 'number'
    || credits.unit !== 'USD'
    || usage.unit !== 'USD'
  ) {
    throw new Error('Live account snapshot must include USD total_credits and total_usage values')
  }

  const balance = credits.limit - usage.used
  return `US$ ${balance.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

async function expectSnapshotMetricRendered(page: Page, snapshot: QuotaSnapshot, metricName: string) {
  const metric = snapshot.quotas.find((candidate) => candidate.metric === metricName)
  expect(metric).toBeDefined()
  const row = page.getByTestId(`quota-metric-${snapshot.account_id}-${snapshot.scope}-${metricName}`)
  let displayedFieldCount = 0
  for (const field of displayedFields) {
    const value = metric?.[field]
    if (typeof value === 'number') {
      displayedFieldCount += 1
      await expect(row).toContainText(formatProviderValue(value, metric?.unit, snapshot.scope, metricName))
    }
  }
  expect(displayedFieldCount).toBeGreaterThan(0)
}

test('live quota values render responsively and refresh only on demand', async ({ page }) => {
  test.skip(!quotaEndpoint && !useLiveProxy, 'Set AIHUB_E2E_QUOTA_API_URL or AIHUB_E2E_LIVE_PROXY=1 for an authorized live quota API')
  expect(sensitiveKeys({ management_api_key: 'hidden', accessToken: 'hidden' }))
    .toEqual(['management_api_key', 'accessToken'])
  let quotaRequests = 0
  let quotaResponses = 0
  let sanitizedResponse = ''
  const quotaStatuses: number[] = []
  const browserErrors: string[] = []
  page.on('pageerror', (error) => browserErrors.push(error.message))
  page.on('response', (response) => {
    if (new URL(response.url()).pathname === '/api/ui/providers/quotas') {
      quotaStatuses.push(response.status())
      void response.text().then((body) => {
        sanitizedResponse = body
        quotaResponses += 1
      })
    }
  })
  page.on('console', (message) => {
    if (message.type() === 'error') browserErrors.push(message.text())
  })

  await page.route('**/*', async (route) => {
    const pathName = new URL(route.request().url()).pathname
    if (!pathName.startsWith('/api/')) {
      await route.continue()
      return
    }
    if (pathName === '/api/ui/session') {
      await route.fulfill({ json: { authenticated: true, mode: 'loopback_trust' } })
      return
    }
    if (pathName === '/api/ui/providers/quotas') {
      quotaRequests += 1
      if (quotaEndpoint) {
        const response = await route.fetch({ url: quotaEndpoint })
        sanitizedResponse = await response.text()
        await route.fulfill({ response, body: sanitizedResponse })
      } else {
        await route.continue()
      }
      return
    }
    if (pathName === '/api/runtime/meta') {
      await route.fulfill({ json: { environment: 'development', data_origin: 'live', version: 'validation', gateway_status: 'ready' } })
      return
    }
    if (pathName === '/api/telemetry/traffic-summary') {
      await route.fulfill({ json: { range: '24h', total_requests: 0, trend_percentage: 0, buckets: [], has_sufficient_history: false } })
      return
    }
    if (pathName === '/api/events') {
      await route.fulfill({ status: 200, contentType: 'text/event-stream', body: ': connected\n\n' })
      return
    }
    await route.fulfill({ json: {} })
  })
  await page.route('**/healthz', (route) => route.fulfill({ json: { status: 'ok' } }))
  await page.route('**/readyz', (route) => route.fulfill({ json: { status: 'ready', checks: {} } }))
  await page.route('**/diagnostics/providers', (route) => route.fulfill({ json: { providers: [] } }))

  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.goto('/')
  await expect(page.getByText('Consumo de franquias')).toBeVisible()
  await expect(page.getByTestId('provider-summary-openrouter')).toHaveCount(1)
  await expect(page.getByText('Chave API')).toHaveCount(0)
  await expect.poll(() => sanitizedResponse.length).toBeGreaterThan(0)
  const snapshots = JSON.parse(sanitizedResponse) as {
    snapshots: QuotaSnapshot[]
  }
  const keySnapshot = snapshots.snapshots.find((snapshot) => snapshot.account_id === 'openrouter' && snapshot.scope === 'api_key')
  const accountSnapshot = snapshots.snapshots.find((snapshot) => snapshot.account_id === 'openrouter' && snapshot.scope === 'account')
  if (keySnapshot === undefined || accountSnapshot === undefined) {
    throw new Error('Live OpenRouter response must include both key and account scopes')
  }
  expect(keySnapshot.status).toBe('available')
  expect(keySnapshot.quotas.find((metric) => metric.metric === 'usage')?.used).toBe(0)
  await expect(page.getByTestId('quota-scope-openrouter-api_key')).toHaveCount(0)
  expect(accountSnapshot.status).toBe('available')
  await expect(page.getByTestId('quota-scope-openrouter-account')).toBeVisible()
  await expectSnapshotMetricRendered(page, accountSnapshot, 'total_usage')
  await expectSnapshotMetricRendered(page, accountSnapshot, 'total_credits')
  await expect(page.getByText('Management API Key não configurada')).toHaveCount(0)
  await expect(page.getByTestId(`quota-balance-${accountSnapshot.account_id}`))
    .toContainText(expectedAvailableBalance(accountSnapshot))
  expect(sensitiveKeys(JSON.parse(sanitizedResponse))).toEqual([])

  const initialRequests = quotaRequests
  const initialResponses = quotaResponses
  expect(initialRequests).toBeGreaterThan(0)
  await page.waitForTimeout(3500)
  expect(quotaRequests, `Unexpected automatic quota request; response statuses: ${quotaStatuses.join(', ')}`)
    .toBe(initialRequests)
  await page.getByRole('button', { name: 'Atualizar' }).click()
  await expect.poll(() => quotaRequests).toBe(initialRequests + 1)
  await expect.poll(() => quotaResponses).toBe(initialResponses + 1)

  const evidencePath = (name: string) => path.resolve(process.cwd(), '../docs/validation', name)
  await page.screenshot({ path: evidencePath('etapa-1.1-live-quota-desktop.png'), fullPage: true })
  const quotaPanel = page.locator('.overview-panel').nth(1)
  await quotaPanel.screenshot({ path: evidencePath('etapa-1.1-live-quota-desktop-panel.png') })
  await page.getByTestId('quota-scope-openrouter-account').scrollIntoViewIfNeeded()
  await quotaPanel.screenshot({ path: evidencePath('etapa-1.1-live-quota-account-scope-desktop.png') })

  await page.setViewportSize({ width: 390, height: 844 })
  await page.getByTestId('quota-metric-openrouter-account-total_usage').scrollIntoViewIfNeeded()
  const mobilePanel = await quotaPanel.boundingBox()
  expect(mobilePanel?.height ?? 0).toBeGreaterThan(100)
  await page.screenshot({ path: evidencePath('etapa-1.1-live-quota-mobile.png'), fullPage: true })
  await quotaPanel.screenshot({ path: evidencePath('etapa-1.1-live-quota-mobile-panel.png') })
  await page.getByTestId('quota-scope-openrouter-account').scrollIntoViewIfNeeded()
  await quotaPanel.screenshot({ path: evidencePath('etapa-1.1-live-quota-account-scope-mobile.png') })
  expect(browserErrors).toEqual([])
})
