import { useMemo, useState } from 'react'
import { useQueries, useQuery, useQueryClient } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { AppPageHeader } from '../../../app/layouts/AppPageHeader'
import { listProviders, providersQueryKeys } from '../../providers'
import { Banner } from '../../../shared/ui/banner'
import { Button } from '../../../shared/ui/button'
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
import { AdmissionsDailyChart } from '../components/AdmissionsDailyChart'
import { ConcurrencySection } from '../components/ConcurrencySection'
import { DataSourceFootnote } from '../components/DataSourceFootnote'
import { ProviderUsageSection } from '../components/ProviderUsageSection'
import { QuotaPoliciesTable } from '../components/QuotaPoliciesTable'
import { ResourcesFilters } from '../components/ResourcesFilters'
import { ResourcesSummaryMetrics } from '../components/ResourcesSummaryMetrics'
import { ResourcesTabNav } from '../components/ResourcesTabNav'
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
  const [draftFilter, setDraftFilter] = useState(defaultResourcesUiFilter)
  const [appliedFilter, setAppliedFilter] = useState(defaultResourcesUiFilter)

  const scopeFilter = useMemo(() => toAnalyticsScopeFilter(appliedFilter), [appliedFilter])
  const range = useMemo(() => resourcesRangeFromUi(appliedFilter), [appliedFilter])

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
  const providerUsageLoading =
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
    <div className="flex min-h-0 flex-1 flex-col">
      <AppPageHeader
        actions={
          <Button variant="ghost" size="sm" onClick={() => void handleRefresh()}>
            <RefreshCw className="size-3.5" aria-hidden />
            Refresh
          </Button>
        }
      />

      <div className="min-h-0 flex-1 overflow-auto px-[30px] pb-5 pt-1">
        {uniqueErrors.length > 0 ? (
          <Banner variant="error" className="mb-3" role="alert">
            <ul className="list-disc space-y-1 pl-4">
              {uniqueErrors.map((message) => (
                <li key={message}>{message}</li>
              ))}
            </ul>
          </Banner>
        ) : null}

        <ResourcesSummaryMetrics {...summary} isLoading={summaryMetricsLoading} />

        <ResourcesFilters
          draft={draftFilter}
          providerOptions={providerOptions}
          onChange={setDraftFilter}
          onApply={() => setAppliedFilter(draftFilter)}
        />

        <ResourcesTabNav active={activeTab} onChange={setActiveTab} />

        {activeTab === 'overview' ? (
          <div className="grid gap-3">
            <div className="grid gap-3 lg:grid-cols-[1.12fr_0.88fr]">
              {admissionsQuery.isSuccess && admissionsQuery.data ? (
                <AdmissionsDailyChart
                  daily={admissionsQuery.data.daily}
                  totalAdmitted={admissionsQuery.data.total_admitted}
                  fetchedAtMs={admissionsQuery.dataUpdatedAt}
                />
              ) : (
                <div className="rounded-[10px] border border-line bg-surface p-4 text-[10px] text-muted">
                  {admissionsQuery.isPending
                    ? 'Loading admissions analytics…'
                    : 'Admissions series unavailable for the selected range.'}
                </div>
              )}
              <QuotaPoliciesTable
                policies={resourcePolicies}
                utilization={quotaUtilization}
                compact
                isLoading={policiesLoading}
                fetchedAtMs={policiesFetchedAtMs}
              />
            </div>
            <ConcurrencySection
              policies={concurrencyPolicies}
              snapshots={concurrencySnapshots}
              isLoading={concurrencyLoading}
              snapshotFetchedAtMs={concurrencySnapshotFetchedAtMs}
              policiesFetchedAtMs={concurrencyPoliciesFetchedAtMs}
            />
          </div>
        ) : null}

        {activeTab === 'policies' ? (
          <QuotaPoliciesTable
            policies={resourcePolicies}
            utilization={quotaUtilization}
            isLoading={policiesLoading}
            fetchedAtMs={policiesFetchedAtMs}
          />
        ) : null}

        {activeTab === 'concurrency' ? (
          <ConcurrencySection
            policies={concurrencyPolicies}
            snapshots={concurrencySnapshots}
            isLoading={concurrencyLoading}
            snapshotFetchedAtMs={concurrencySnapshotFetchedAtMs}
            policiesFetchedAtMs={concurrencyPoliciesFetchedAtMs}
          />
        ) : null}

        {activeTab === 'provider-usage' ? (
          <ProviderUsageSection
            usage={providerUsageQuery.data}
            distribution={providerDistributionQuery.data}
            isLoading={providerUsageLoading}
            isError={providerUsageQuery.isError}
            isDistributionError={providerDistributionQuery.isError}
            usageFetchedAtMs={providerUsageQuery.dataUpdatedAt}
            distributionFetchedAtMs={providerDistributionQuery.dataUpdatedAt}
          />
        ) : null}

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
    </div>
  )
}
