import type { GenerationFilterDraft } from '../components/GenerationsFilters'

export function validateGenerationFilterDraft(
  filters: GenerationFilterDraft,
): { ok: true } | { ok: false; message: string } {
  if (filters.start_date && filters.end_date && filters.start_date > filters.end_date) {
    return {
      ok: false,
      message: 'A data inicial deve ser anterior ou igual à data final.',
    }
  }
  return { ok: true }
}
