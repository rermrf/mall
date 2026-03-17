import { lazy, Suspense } from 'react'
import { createBrowserRouter } from 'react-router-dom'
import { SpinLoading } from 'antd-mobile'
import TabBarLayout from '@/components/Layout/TabBarLayout'
import AuthGuard from '@/components/AuthGuard'

const HomePage = lazy(() => import('@/pages/home'))
const SearchPage = lazy(() => import('@/pages/search'))
const CartPage = lazy(() => import('@/pages/cart'))
const UserPage = lazy(() => import('@/pages/user'))
const LoginPage = lazy(() => import('@/pages/auth/Login'))
const SignupPage = lazy(() => import('@/pages/auth/Signup'))
const ProductDetail = lazy(() => import('@/pages/product/Detail'))
const OrderConfirm = lazy(() => import('@/pages/order/Confirm'))
const PaymentPage = lazy(() => import('@/pages/payment'))

// Phase 2 pages
const ProfilePage = lazy(() => import('@/pages/user/Profile'))
const AddressesPage = lazy(() => import('@/pages/user/Addresses'))
const AddressEditPage = lazy(() => import('@/pages/user/AddressEdit'))
const OrderListPage = lazy(() => import('@/pages/order/List'))
const OrderDetailPage = lazy(() => import('@/pages/order/Detail'))
const RefundListPage = lazy(() => import('@/pages/order/RefundList'))
const RefundDetailPage = lazy(() => import('@/pages/order/RefundDetail'))

// Phase 3 pages
const CategoryPage = lazy(() => import('@/pages/category'))
const NotificationPage = lazy(() => import('@/pages/notification'))
const CouponsPage = lazy(() => import('@/pages/marketing/Coupons'))
const MyCouponsPage = lazy(() => import('@/pages/marketing/MyCoupons'))
const SeckillPage = lazy(() => import('@/pages/marketing/Seckill'))

function Loading() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '40vh 0' }}>
      <SpinLoading color='default' />
    </div>
  )
}

function Lazy({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<Loading />}>{children}</Suspense>
}

export const router = createBrowserRouter([
  {
    element: <TabBarLayout />,
    children: [
      { path: '/', element: <Lazy><HomePage /></Lazy> },
      { path: '/category', element: <Lazy><CategoryPage /></Lazy> },
      {
        path: '/cart',
        element: <AuthGuard><Lazy><CartPage /></Lazy></AuthGuard>,
      },
      {
        path: '/me',
        element: <Lazy><UserPage /></Lazy>,
      },
    ],
  },
  { path: '/search', element: <Lazy><SearchPage /></Lazy> },
  { path: '/login', element: <Lazy><LoginPage /></Lazy> },
  { path: '/signup', element: <Lazy><SignupPage /></Lazy> },
  { path: '/product/:id', element: <Lazy><ProductDetail /></Lazy> },
  {
    path: '/order/confirm',
    element: <AuthGuard><Lazy><OrderConfirm /></Lazy></AuthGuard>,
  },
  {
    path: '/payment/:orderNo',
    element: <AuthGuard><Lazy><PaymentPage /></Lazy></AuthGuard>,
  },
  // Phase 2 routes
  {
    path: '/me/profile',
    element: <AuthGuard><Lazy><ProfilePage /></Lazy></AuthGuard>,
  },
  {
    path: '/me/addresses',
    element: <AuthGuard><Lazy><AddressesPage /></Lazy></AuthGuard>,
  },
  {
    path: '/me/addresses/edit',
    element: <AuthGuard><Lazy><AddressEditPage /></Lazy></AuthGuard>,
  },
  {
    path: '/orders',
    element: <AuthGuard><Lazy><OrderListPage /></Lazy></AuthGuard>,
  },
  {
    path: '/orders/:orderNo',
    element: <AuthGuard><Lazy><OrderDetailPage /></Lazy></AuthGuard>,
  },
  {
    path: '/refunds',
    element: <AuthGuard><Lazy><RefundListPage /></Lazy></AuthGuard>,
  },
  {
    path: '/refunds/:refundNo',
    element: <AuthGuard><Lazy><RefundDetailPage /></Lazy></AuthGuard>,
  },
  // Phase 3 routes
  {
    path: '/notifications',
    element: <AuthGuard><Lazy><NotificationPage /></Lazy></AuthGuard>,
  },
  {
    path: '/coupons',
    element: <Lazy><CouponsPage /></Lazy>,
  },
  {
    path: '/me/coupons',
    element: <AuthGuard><Lazy><MyCouponsPage /></Lazy></AuthGuard>,
  },
  {
    path: '/seckill',
    element: <Lazy><SeckillPage /></Lazy>,
  },
])
