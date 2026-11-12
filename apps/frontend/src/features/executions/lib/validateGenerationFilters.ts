import i18n from 'i18next'
import type { GenerationFilterDraft } from '../components/GenerationsFilters'

export function validateGenerationFilterDraft(
  filters: GenerationFilterDraft,
): { ok: true } | { ok: false; message: string } {
  if (filters.start_date && filters.end_date && filters.start_date > filters.end_date) {
    return {
      ok: false,
      message: i18n.t('executions:validation.startDateAfterEnd'),
    }
  }
  return { ok: true }
}
