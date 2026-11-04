import { chromium } from '@playwright/test'
import path from 'node:path'

const artifactDir = '/home/rafael/.gemini/antigravity/brain/e1e4e070-c4f9-41dc-b139-57ef8092ae9b'
const reportDir = '/home/rafael/dev-rafael/connexto-aihub/reports/gate-zero-8.1'

const viewports = [
  { name: '1280x800', width: 1280, height: 800 },
  { name: '768x1024', width: 768, height: 1024 },
  { name: '390x844', width: 390, height: 844 },
]

async function capture() {
  const browser = await chromium.launch({
    channel: 'chrome',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage', '--disable-gpu'],
  })

  for (const vp of viewports) {
    const page = await browser.newPage({ viewport: { width: vp.width, height: vp.height } })
    console.log(`Capturing Overview @ ${vp.name}...`)
    await page.goto('http://127.0.0.1:5173/', { waitUntil: 'networkidle', timeout: 15000 })
    await page.waitForTimeout(1000)

    const artifactPath = path.join(artifactDir, `overview-validated-${vp.name}.png`)
    const reportPath = path.join(reportDir, `overview-${vp.name}.png`)

    await page.screenshot({ path: artifactPath, fullPage: true })
    await page.screenshot({ path: reportPath, fullPage: true })
    console.log(`Saved screenshot to ${artifactPath}`)
    await page.close()
  }

  await browser.close()
  console.log('All viewports captured successfully!')
}

capture().catch((err) => {
  console.error('Capture error:', err)
  process.exit(1)
})
