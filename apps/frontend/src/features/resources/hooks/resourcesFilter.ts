import { defaultAnalyticsRange } from '../lib/dateRange'
import type { AnalyticsScopeFilter, ResourcesUiFilter } from '../types'

export function toAnalyticsScopeFilter(ui: ResourcesUiFilter): AnalyticsScopeFilter {
  return {
    provider_id: ui.provider_id.trim() || undefined,
    model_id: ui.model_id.trim() || undefined,
  }
}

export const defaultResourcesUiFilter: ResourcesUiFilter = {
  ...defaultAnalyticsRange(),
  provider_id: '',
  model_id: '',
}

export function resourcesRangeFromUi(ui: ResourcesUiFilter): { from: string; to: string } {
  return { from: ui.from.trim(), to: ui.to.trim() }
}
