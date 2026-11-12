import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { ForgotPasswordPage, LoginPage, OperatorAuthGate, ResetPasswordPage, SignUpPage } from '../features/auth'
import { CatalogPage } from '../features/catalog'
import { ExecutionsPage } from '../features/executions'
import { ResourcesPage } from '../features/resources'
import { IntelligenceHelpPage, IntelligencePage } from '../features/intelligence'
import { OverviewPage } from '../features/overview'
import { PlaygroundPage } from '../features/playground'
import { ProvidersPage } from '../features/providers'
import { ProfilePage, SettingsPage } from '../features/account'
import { RootLayout } from './layouts/RootLayout'

const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/signup', element: <SignUpPage /> },
  { path: '/forgot-password', element: <ForgotPasswordPage /> },
  { path: '/reset-password', element: <ResetPasswordPage /> },
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
      { path: 'intelligence/help', element: <IntelligenceHelpPage /> },
      { path: 'executions', element: <ExecutionsPage /> },
      { path: 'resources', element: <ResourcesPage /> },
      { path: 'profile', element: <ProfilePage /> },
      { path: 'settings', element: <SettingsPage /> },
    ],
  },
])

export function AppRouter() {
  return <RouterProvider router={router} />
}
