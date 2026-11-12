import i18n from '../../../shared/i18n'
import type { CursorProxyAuthStartResponse } from '../api/queries'

export function cursorAuthStartSuccessMessage(res: CursorProxyAuthStartResponse): string {
  const mode = res.browserMode ?? 'headed'
  const link = res.interactionUrl

  if (mode === 'headless') {
    return i18n.t('providers:authStart.headless', { link })
  }

  return i18n.t('providers:authStart.headed', { link })
}
