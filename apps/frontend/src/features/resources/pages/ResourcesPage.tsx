import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQueries, useQuery, useQueryClient } from '@tanstack/react-query'
import { appPageClassName } from '../../../app/layouts/appShellLayout'
import { listProviders, providersQueryKeys } from '../../providers'
import { Banner } from '../../../shared/ui/banner'
import { RefreshButton } from '../../../shared/ui'
import {
  getAdmissionsAnalytics,
  getConcurrencySnapshot,
  getProviderUsageAnalytics,
  getProviderUsageDistribution,
  getQuotaUtilization,
  listConcurrencyPolicies,
  listResourcePolicies,
  resourcesQueryKeys,
} from '../api/queries'
import { ConsumptionChart } from '../components/ConsumptionChart'
import { ConcurrencyPanel } from '../components/ConcurrencyPanel'
import { DataSourceFootnote } from '../components/DataSourceFootnote'
import { PoliciesHighlightTable } from '../components/PoliciesHighlightTable'
import { PoliciesTable } from '../components/PoliciesTable'
import { ResourcesFilters } from '../components/ResourcesFilters'
import { ResourcesMetricGrid } from '../components/ResourcesMetricGrid'
import { ResourcesPageHead } from '../components/ResourcesPageHead'
import { ResourcesViewTabs } from '../components/ResourcesViewTabs'
import { UsageDistribution } from '../components/UsageDistribution'
import {
  defaultResourcesUiFilter,
  resourcesRangeFromUi,
  toAnalyticsScopeFilter,
} from '../hooks/resourcesFilter'
import { resourcesErrorMessage } from '../lib/resourcesError'
import { maxDataUpdatedAt } from '../lib/queryTimestamps'
import { computeResourcesSummary } from '../lib/summaryMetrics'
import type { ResourcesTabId } from '../types'

export function ResourcesPage() {
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<ResourcesTabId>('overview')
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [draftFilter, setDraftFilter] = useState(defaultResourcesUiFilter)
  const [appliedFilter, setAppliedFilter] = useState(defaultResourcesUiFilter)

  const scopeFilter = useMemo(() => toAnalyticsScopeFilter(appliedFilter), [appliedFilter])
  const range = useMemo(() => resourcesRangeFromUi(appliedFilter), [appliedFilter])
  const periodLabel =
    appliedFilter.from === defaultResourcesUiFilter.from &&
    appliedFilter.to === defaultResourcesUiFilter.to &&
    appliedFilter.provider_id === '' &&
    appliedFilter.model_id === ''
      ? 'Últimos 7 dias'
      : `${range.from} – ${range.to}`

  const providersQuery = useQuery({
    queryKey: providersQueryKeys.list,
    queryFn: listProviders,
    refetchInterval: false,
    retry: 1,
  })

  const resourcePoliciesQuery = useQuery({
    queryKey: resourcesQueryKeys.resourcePolicies,
    queryFn: listResourcePolicies,
    refetchInterval: false,
    retry: 1,
  })

  const quotaQuery = useQuery({
    queryKey: resourcesQueryKeys.quotaUtilization(scopeFilter),
    queryFn: () => getQuotaUtilization(scopeFilter),
    refetchInterval: false,
    retry: 1,
  })

  const concurrencyPoliciesQuery = useQuery({
    queryKey: resourcesQueryKeys.concurrencyPolicies,
    queryFn: listConcurrencyPolicies,
    refetchInterval: false,
    retry: 1,
  })

  const concurrencySnapshotQuery = useQuery({
    queryKey: resourcesQueryKeys.concurrencySnapshot(scopeFilter),
    queryFn: () => getConcurrencySnapshot(scopeFilter),
    refetchInterval: false,
    retry: 1,
  })

  const analyticsQueries = useQueries({
    queries: [
      {
        queryKey: resourcesQueryKeys.admissions(range, scopeFilter),
        queryFn: () => getAdmissionsAnalytics(range, scopeFilter),
        refetchInterval: false,
        retry: 1,
      },
      {
        queryKey: resourcesQueryKeys.providerUsage(range, scopeFilter),
        queryFn: () => getProviderUsageAnalytics(range, scopeFilter),
        refetchInterval: false,
        retry: 1,
      },
      {
        queryKey: resourcesQueryKeys.providerUsageDistribution(range, scopeFilter),
        queryFn: () => getProviderUsageDistribution(range, scopeFilter),
        refetchInterval: false,
        retry: 1,
      },
    ],
  })

  const [admissionsQuery, providerUsageQuery, providerDistributionQuery] = analyticsQueries

  const providerOptions = useMemo(() => {
    const ids = providersQuery.data?.providers.map((provider) => provider.id) ?? []
    return [...new Set(ids)].sort()
  }, [providersQuery.data?.providers])

  const resourcePolicies = resourcePoliciesQuery.data?.policies ?? []
  const quotaUtilization = quotaQuery.data?.quota_utilization ?? []
  const concurrencyPolicies = concurrencyPoliciesQuery.data?.policies ?? []
  const concurrencySnapshots = concurrencySnapshotQuery.data?.concurrency ?? []

  const summary = computeResourcesSummary({
    resourcePolicyCount: resourcePolicies.length,
    concurrencyPolicyCount: concurrencyPolicies.length,
    quotaUtilization,
    concurrencySnapshots,
  })

  const policiesLoading = resourcePoliciesQuery.isPending || quotaQuery.isPending
  const concurrencyLoading =
    concurrencyPoliciesQuery.isPending || concurrencySnapshotQuery.isPending
  const distributionLoading =
    providerUsageQuery.isPending || providerDistributionQuery.isPending

  const summaryMetricsLoading =
    resourcePoliciesQuery.data === undefined &&
    concurrencyPoliciesQuery.data === undefined &&
    (resourcePoliciesQuery.isPending ||
      quotaQuery.isPending ||
      concurrencyPoliciesQuery.isPending ||
      concurrencySnapshotQuery.isPending)

  const policiesFetchedAtMs = maxDataUpdatedAt(
    resourcePoliciesQuery.dataUpdatedAt,
    quotaQuery.dataUpdatedAt,
  )
  const concurrencySnapshotFetchedAtMs = maxDataUpdatedAt(concurrencySnapshotQuery.dataUpdatedAt)
  const concurrencyPoliciesFetchedAtMs = maxDataUpdatedAt(concurrencyPoliciesQuery.dataUpdatedAt)
  const distributionFetchedAtMs = maxDataUpdatedAt(
    providerUsageQuery.dataUpdatedAt,
    providerDistributionQuery.dataUpdatedAt,
  )

  const errors = [
    providersQuery.isError
      ? resourcesErrorMessage(providersQuery.error, 'Provider list unavailable for filters.')
      : undefined,
    resourcePoliciesQuery.isError
      ? resourcesErrorMessage(
          resourcePoliciesQuery.error,
          'Resource policies are unavailable. Verify admin API access and PostgreSQL mode.',
        )
      : undefined,
    quotaQuery.isError
      ? resourcesErrorMessage(quotaQuery.error, 'Quota utilization snapshot failed.')
      : undefined,
    concurrencyPoliciesQuery.isError
      ? resourcesErrorMessage(concurrencyPoliciesQuery.error, 'Concurrency policies failed.')
      : undefined,
    concurrencySnapshotQuery.isError
      ? resourcesErrorMessage(concurrencySnapshotQuery.error, 'Concurrency snapshot failed.')
      : undefined,
    admissionsQuery.isError
      ? resourcesErrorMessage(admissionsQuery.error, 'Admissions analytics failed.')
      : undefined,
    providerUsageQuery.isError
      ? resourcesErrorMessage(providerUsageQuery.error, 'Provider usage analytics failed.')
      : undefined,
    providerDistributionQuery.isError
      ? resourcesErrorMessage(
          providerDistributionQuery.error,
          'Provider usage distribution failed.',
        )
      : undefined,
  ].filter((message): message is string => message !== undefined)

  const uniqueErrors = [...new Set(errors)]

  const latestUpdatedAt = maxDataUpdatedAt(
    resourcePoliciesQuery.dataUpdatedAt,
    quotaQuery.dataUpdatedAt,
    concurrencyPoliciesQuery.dataUpdatedAt,
    concurrencySnapshotQuery.dataUpdatedAt,
    admissionsQuery.dataUpdatedAt,
    providerUsageQuery.dataUpdatedAt,
    providerDistributionQuery.dataUpdatedAt,
    providersQuery.dataUpdatedAt,
  )

  async function handleRefresh(): Promise<void> {
    await queryClient.invalidateQueries({ queryKey: ['resources'] })
    await queryClient.invalidateQueries({ queryKey: providersQueryKeys.list })
  }

  return (
    <div className={appPageClassName('resources-page')}>
      <ResourcesPageHead
        actions={
          <div className="resources-pagehead-actions">
            <Link to="/providers" className="resources-providers-link">
              Ver franquias por provedor ↗
            </Link>
            <RefreshButton
              className="resources-outline-btn"
              onRefresh={handleRefresh}
              isRefreshing={
                admissionsQuery.isFetching ||
                providerUsageQuery.isFetching ||
                providerDistributionQuery.isFetching ||
                providersQuery.isFetching
              }
            />
          </div>
        }
      />

      {uniqueErrors.length > 0 ? (
        <Banner variant="error" role="alert">
          <ul className="list-disc space-y-1 pl-4">
            {uniqueErrors.map((message) => (
              <li key={message}>{message}</li>
            ))}
          </ul>
        </Banner>
      ) : null}

      <ResourcesMetricGrid {...summary} isLoading={summaryMetricsLoading} />

      <ResourcesViewTabs active={activeTab} onChange={setActiveTab} />

      {filtersOpen ? (
        <ResourcesFilters
          draft={draftFilter}
          providerOptions={providerOptions}
          onChange={setDraftFilter}
          onApply={() => setAppliedFilter(draftFilter)}
        />
      ) : null}

      <div className="resources-stage">
      {activeTab === 'overview' ? (
        <div className="resources-overview">
          <div className="resources-overview-grid">
            <ConsumptionChart
              daily={admissionsQuery.data?.daily ?? []}
              totalAdmitted={admissionsQuery.data?.total_admitted ?? 0}
              quotaUtilization={quotaUtilization}
              periodLabel={periodLabel}
              onTogglePeriod={() => setFiltersOpen((open) => !open)}
              isLoading={admissionsQuery.isPending}
              isError={admissionsQuery.isError}
              fetchedAtMs={admissionsQuery.dataUpdatedAt}
            />
            <UsageDistribution
              distribution={providerDistributionQuery.data}
              usage={providerUsageQuery.data}
              isLoading={distributionLoading}
              isError={providerDistributionQuery.isError || providerUsageQuery.isError}
              fetchedAtMs={distributionFetchedAtMs}
            />
          </div>
          <PoliciesHighlightTable
            policies={resourcePolicies}
            utilization={quotaUtilization}
            isLoading={policiesLoading}
            fetchedAtMs={policiesFetchedAtMs}
            onViewPolicies={() => setActiveTab('policies')}
          />
        </div>
      ) : null}

      {activeTab === 'policies' ? (
        <div className="h-full min-h-0">
        <PoliciesTable
          policies={resourcePolicies}
          utilization={quotaUtilization}
          isLoading={policiesLoading}
          fetchedAtMs={policiesFetchedAtMs}
        />
        </div>
      ) : null}

      {activeTab === 'concurrency' ? (
        <ConcurrencyPanel
          policies={concurrencyPolicies}
          snapshots={concurrencySnapshots}
          isLoading={concurrencyLoading}
          snapshotFetchedAtMs={concurrencySnapshotFetchedAtMs}
          policiesFetchedAtMs={concurrencyPoliciesFetchedAtMs}
        />
      ) : null}
      </div>

      <DataSourceFootnote
        sources={[
          '/api/resource-policies',
          '/api/resource-analytics/quota-utilization',
          '/api/concurrency-policies',
          '/api/resource-analytics/concurrency',
          '/api/resource-analytics/admissions',
          '/api/resource-analytics/provider-usage',
          '/api/resource-analytics/provider-usage/distribution',
        ]}
        updatedAt={latestUpdatedAt}
      />
    </div>
  )
}
