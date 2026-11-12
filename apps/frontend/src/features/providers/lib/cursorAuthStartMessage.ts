import type { CursorProxyAuthStartResponse } from '../api/queries'

export function cursorAuthStartSuccessMessage(res: CursorProxyAuthStartResponse): string {
  const mode = res.browserMode ?? 'headed'
  const link = res.interactionUrl

  if (mode === 'headless') {
    return (
      'Login iniciado no proxy em modo headless (comum no Docker): nenhuma janela Chrome abre no seu computador. ' +
      'Login Google/GitHub exige proxy no host: pare o Docker na porta 9090, rode `pnpm dev:proxy` (ou `pnpm --dir apps/proxy dev`) ' +
      'com CURSOR_USAGE_AUTH_HEADLESS=false e repita Login web. ' +
      `Referência: ${link}`
    )
  }

  return (
    'Login iniciado. Use a janela «Sign in — Google Chrome» que o proxy abriu (e-mail/senha ou Google/GitHub). ' +
    'Não use outra aba do navegador — o status atualiza aqui automaticamente. ' +
    `Instruções: ${link}`
  )
}
