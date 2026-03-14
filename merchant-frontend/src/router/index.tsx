import { lazy, Suspense } from 'react'
import { createBrowserRouter } from 'react-router-dom'
import { Spin } from 'antd'
import MainLayout from '@/components/layout/MainLayout'
import AuthGuard from '@/components/AuthGuard'

const LoginPage = lazy(() => import('@/pages/login'))
const Dashboard = lazy(() => import('@/pages/dashboard'))
const ProductList = lazy(() => import('@/pages/product/ProductList'))
const ProductForm = lazy(() => import('@/pages/product/ProductForm'))
const CategoryList = lazy(() => import('@/pages/product/CategoryList'))
const BrandList = lazy(() => import('@/pages/product/BrandList'))
const OrderList = lazy(() => import('@/pages/order/OrderList'))
const OrderDetail = lazy(() => import('@/pages/order/OrderDetail'))
const RefundList = lazy(() => import('@/pages/order/RefundList'))
const StockList = lazy(() => import('@/pages/inventory/StockList'))
const StockLog = lazy(() => import('@/pages/inventory/StockLog'))
const CouponList = lazy(() => import('@/pages/marketing/CouponList'))
const CouponForm = lazy(() => import('@/pages/marketing/CouponForm'))
const SeckillList = lazy(() => import('@/pages/marketing/SeckillList'))
const SeckillForm = lazy(() => import('@/pages/marketing/SeckillForm'))
const PromotionList = lazy(() => import('@/pages/marketing/PromotionList'))
const TemplateList = lazy(() => import('@/pages/logistics/TemplateList'))
const TemplateForm = lazy(() => import('@/pages/logistics/TemplateForm'))
const ShopSettings = lazy(() => import('@/pages/shop/ShopSettings'))
const StaffList = lazy(() => import('@/pages/staff/StaffList'))
const RoleList = lazy(() => import('@/pages/staff/RoleList'))
const NotificationList = lazy(() => import('@/pages/notification/NotificationList'))
const PaymentList = lazy(() => import('@/pages/payment/PaymentList'))
const PaymentDetail = lazy(() => import('@/pages/payment/PaymentDetail'))
const ProfileEdit = lazy(() => import('@/pages/profile/ProfileEdit'))

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
    children: [
      { path: '/', element: <L><Dashboard /></L> },
      { path: '/product/list', element: <L><ProductList /></L> },
      { path: '/product/create', element: <L><ProductForm /></L> },
      { path: '/product/edit/:id', element: <L><ProductForm /></L> },
      { path: '/product/category', element: <L><CategoryList /></L> },
      { path: '/product/brand', element: <L><BrandList /></L> },
      { path: '/order/list', element: <L><OrderList /></L> },
      { path: '/order/:orderNo', element: <L><OrderDetail /></L> },
      { path: '/order/refund', element: <L><RefundList /></L> },
      { path: '/inventory', element: <L><StockList /></L> },
      { path: '/inventory/log', element: <L><StockLog /></L> },
      { path: '/marketing/coupon', element: <L><CouponList /></L> },
      { path: '/marketing/coupon/create', element: <L><CouponForm /></L> },
      { path: '/marketing/coupon/edit/:id', element: <L><CouponForm /></L> },
      { path: '/marketing/seckill', element: <L><SeckillList /></L> },
      { path: '/marketing/seckill/create', element: <L><SeckillForm /></L> },
      { path: '/marketing/seckill/edit/:id', element: <L><SeckillForm /></L> },
      { path: '/marketing/promotion', element: <L><PromotionList /></L> },
      { path: '/logistics/template', element: <L><TemplateList /></L> },
      { path: '/logistics/template/create', element: <L><TemplateForm /></L> },
      { path: '/logistics/template/edit/:id', element: <L><TemplateForm /></L> },
      { path: '/shop/settings', element: <L><ShopSettings /></L> },
      { path: '/staff/list', element: <L><StaffList /></L> },
      { path: '/staff/role', element: <L><RoleList /></L> },
      { path: '/notification', element: <L><NotificationList /></L> },
      { path: '/payment', element: <L><PaymentList /></L> },
      { path: '/payment/:paymentNo', element: <L><PaymentDetail /></L> },
      { path: '/profile', element: <L><ProfileEdit /></L> },
    ],
  },
])
