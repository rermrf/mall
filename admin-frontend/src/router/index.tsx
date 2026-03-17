import { lazy, Suspense } from 'react'
import { createBrowserRouter } from 'react-router-dom'
import { Spin } from 'antd'
import MainLayout from '@/components/layout/MainLayout'
import AuthGuard from '@/components/AuthGuard'
import ErrorBoundary from '@/components/ErrorBoundary'

const LoginPage = lazy(() => import('@/pages/login'))
const Dashboard = lazy(() => import('@/pages/dashboard'))
const UserList = lazy(() => import('@/pages/user/UserList'))
const RoleList = lazy(() => import('@/pages/role/RoleList'))
const TenantList = lazy(() => import('@/pages/tenant/TenantList'))
const TenantDetail = lazy(() => import('@/pages/tenant/TenantDetail'))
const PlanList = lazy(() => import('@/pages/plan/PlanList'))
const CategoryList = lazy(() => import('@/pages/category/CategoryList'))
const BrandList = lazy(() => import('@/pages/brand/BrandList'))
const OrderList = lazy(() => import('@/pages/order/OrderList'))
const OrderDetail = lazy(() => import('@/pages/order/OrderDetail'))
const TemplateList = lazy(() => import('@/pages/notification/TemplateList'))
const TemplateForm = lazy(() => import('@/pages/notification/TemplateForm'))
const SendNotification = lazy(() => import('@/pages/notification/SendNotification'))
const StockQuery = lazy(() => import('@/pages/inventory/StockQuery'))
const StockLog = lazy(() => import('@/pages/inventory/StockLog'))
const CouponList = lazy(() => import('@/pages/marketing/CouponList'))
const SeckillList = lazy(() => import('@/pages/marketing/SeckillList'))
const SeckillDetail = lazy(() => import('@/pages/marketing/SeckillDetail'))
const PromotionList = lazy(() => import('@/pages/marketing/PromotionList'))
const FreightList = lazy(() => import('@/pages/logistics/FreightList'))
const FreightDetail = lazy(() => import('@/pages/logistics/FreightDetail'))

function Loading() {
  return <div style={{ display: 'flex', justifyContent: 'center', padding: '20vh 0' }}><Spin size="large" /></div>
}

function L({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<Loading />}>{children}</Suspense>
}

export const router = createBrowserRouter([
  { path: '/login', element: <L><LoginPage /></L> },
  {
    element: <AuthGuard><MainLayout /></AuthGuard>,
    errorElement: <ErrorBoundary><div /></ErrorBoundary>,
    children: [
      { path: '/', element: <L><Dashboard /></L> },
      { path: '/dashboard', element: <L><Dashboard /></L> },
      { path: '/users', element: <L><UserList /></L> },
      { path: '/roles', element: <L><RoleList /></L> },
      { path: '/tenants', element: <L><TenantList /></L> },
      { path: '/tenants/:id', element: <L><TenantDetail /></L> },
      { path: '/plans', element: <L><PlanList /></L> },
      { path: '/categories', element: <L><CategoryList /></L> },
      { path: '/brands', element: <L><BrandList /></L> },
      { path: '/orders', element: <L><OrderList /></L> },
      { path: '/orders/:orderNo', element: <L><OrderDetail /></L> },
      { path: '/notification-templates', element: <L><TemplateList /></L> },
      { path: '/notification-templates/create', element: <L><TemplateForm /></L> },
      { path: '/notification-templates/:id/edit', element: <L><TemplateForm /></L> },
      { path: '/notifications/send', element: <L><SendNotification /></L> },
      { path: '/inventory', element: <L><StockQuery /></L> },
      { path: '/inventory/logs', element: <L><StockLog /></L> },
      { path: '/coupons', element: <L><CouponList /></L> },
      { path: '/seckill', element: <L><SeckillList /></L> },
      { path: '/seckill/:id', element: <L><SeckillDetail /></L> },
      { path: '/promotions', element: <L><PromotionList /></L> },
      { path: '/freight-templates', element: <L><FreightList /></L> },
      { path: '/freight-templates/:id', element: <L><FreightDetail /></L> },
    ],
  },
])
