import { Spin } from 'antd'
import { useEffect } from 'react'
import { Navigate, Outlet, createBrowserRouter, useLocation } from 'react-router-dom'
import { AppShell } from '../components/common/AppShell'
import { useAuth } from '../hooks/useAuth'
import { AuditPage } from '../pages/AuditPage'
import { BatchDetailPage } from '../pages/BatchDetailPage'
import { BatchesPage } from '../pages/BatchesPage'
import { InspectionsPage } from '../pages/InspectionsPage'
import { LinesPage } from '../pages/LinesPage'
import { LoginPage } from '../pages/LoginPage'
import { ReleasePage } from '../pages/ReleasePage'

function ProtectedRoute({ permission }: { permission?: string }) {
	const { user, can, initialized, refresh } = useAuth()
	const location = useLocation()
	useEffect(() => { if (!initialized) void refresh() }, [initialized, refresh])
	if (!initialized) return <div className="route-loading"><Spin size="large" /></div>
  if (!user) return <Navigate to="/login" state={{ from: location.pathname }} replace />
  if (permission && !can(permission)) return <Navigate to="/lines" replace />
  return <Outlet />
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    element: <ProtectedRoute />,
    children: [{
      element: <AppShell />,
      children: [
        { index: true, element: <Navigate to="/lines" replace /> },
        { path: '/lines', element: <LinesPage /> },
        { path: '/batches', element: <BatchesPage /> },
        { path: '/batches/:id', element: <BatchDetailPage /> },
        { path: '/inspections', element: <InspectionsPage /> },
        { path: '/release', element: <ReleasePage /> },
        {
          element: <ProtectedRoute permission="audit:read" />,
          children: [{ path: '/audit', element: <AuditPage /> }],
        },
      ],
    }],
  },
  { path: '*', element: <Navigate to="/lines" replace /> },
])
