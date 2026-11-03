import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { CatalogPage } from '../features/catalog'
import { OverviewPage } from '../features/overview'
import { PlaygroundPage } from '../features/playground'
import { ProvidersPage } from '../features/providers'
import { RootLayout } from './layouts/RootLayout'

const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    children: [
      { index: true, element: <OverviewPage /> },
      { path: 'providers', element: <ProvidersPage /> },
      { path: 'catalog', element: <CatalogPage /> },
      { path: 'playground', element: <PlaygroundPage /> },
    ],
  },
])

export function AppRouter() {
  return <RouterProvider router={router} />
}
