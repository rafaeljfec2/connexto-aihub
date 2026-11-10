import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { LoginPage, OperatorAuthGate } from '../features/auth'
import { CatalogPage } from '../features/catalog'
import { ExecutionsPage } from '../features/executions'
import { ResourcesPage } from '../features/resources'
import { IntelligencePage } from '../features/intelligence'
import { OverviewPage } from '../features/overview'
import { PlaygroundPage } from '../features/playground'
import { ProvidersPage } from '../features/providers'
import { RootLayout } from './layouts/RootLayout'

const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/',
    element: (
      <OperatorAuthGate>
        <RootLayout />
      </OperatorAuthGate>
    ),
    children: [
      { index: true, element: <OverviewPage /> },
      { path: 'providers', element: <ProvidersPage /> },
      { path: 'catalog', element: <CatalogPage /> },
      { path: 'playground', element: <PlaygroundPage /> },
      { path: 'intelligence', element: <IntelligencePage /> },
      { path: 'executions', element: <ExecutionsPage /> },
      { path: 'resources', element: <ResourcesPage /> },
    ],
  },
])

export function AppRouter() {
  return <RouterProvider router={router} />
}
