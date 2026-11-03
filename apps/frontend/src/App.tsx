import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import { AppShell } from './components/AppShell'
import { useOperationalEvents } from './hooks/useOperationalEvents'
import { OverviewPage } from './pages/OverviewPage'
import './styles.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchInterval: false,
      refetchOnWindowFocus: false,
    },
  },
})

function RootLayout() {
  const { connectionState, reconnect } = useOperationalEvents(queryClient)
  return <AppShell connectionState={connectionState} onReconnect={reconnect} />
}

const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    children: [{ index: true, element: <OverviewPage /> }],
  },
])

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}
