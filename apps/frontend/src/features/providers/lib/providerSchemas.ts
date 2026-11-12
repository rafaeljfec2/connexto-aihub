import { z } from 'zod'
import i18n from '../../../shared/i18n'

export function createProviderPatchSchema() {
  return z.object({
    name: z.string().optional(),
    endpoint: z.string().url(i18n.t('providers:validation.endpointUrl')),
    credential: z.string().optional(),
    management_credential: z.string().optional(),
    oauth_redirect_uri: z
      .string()
      .url(i18n.t('providers:validation.oauthRedirectUrl'))
      .optional()
      .or(z.literal('')),
  })
}
