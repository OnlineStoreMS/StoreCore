import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import { getToken, redirectToPortal, ensureSession, clearToken } from '../utils/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/auth/callback',
      name: 'AuthCallback',
      component: () => import('../views/AuthCallback.vue'),
      meta: { public: true },
    },
    {
      path: '/auth/logout',
      name: 'AuthLogout',
      component: () => import('../views/AuthLogout.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '工作台' } },
        { path: 'stores', name: 'StoreList', component: () => import('../views/store/StoreList.vue'), meta: { title: '门店档案' } },
        { path: 'pos', name: 'PosCashier', component: () => import('../views/pos/PosCashier.vue'), meta: { title: '收银台' } },
        { path: 'pos/orders', name: 'PosOrderList', component: () => import('../views/pos/PosOrderList.vue'), meta: { title: '收银订单' } },
        { path: 'pos/orders/:id', name: 'PosOrderDetail', component: () => import('../views/pos/PosOrderDetail.vue'), meta: { title: '收银订单详情' } },
        { path: 'receipt-templates', name: 'ReceiptTemplateList', component: () => import('../views/pos/ReceiptTemplateList.vue'), meta: { title: '小票模板' } },
        { path: 'service-catalog', name: 'ServiceCatalog', component: () => import('../views/service/ServiceCatalog.vue'), meta: { title: '服务目录' } },
        { path: 'sales-orders', name: 'SalesOrderList', component: () => import('../views/sales/SalesOrderList.vue'), meta: { title: '销售订单' } },
        { path: 'sales-orders/create', name: 'SalesOrderCreate', component: () => import('../views/sales/SalesOrderForm.vue'), meta: { title: '新建销售订单' } },
        { path: 'sales-orders/:id/edit', name: 'SalesOrderEdit', component: () => import('../views/sales/SalesOrderForm.vue'), meta: { title: '编辑销售订单' } },
        { path: 'sales-orders/:id', name: 'SalesOrderDetail', component: () => import('../views/sales/SalesOrderDetail.vue'), meta: { title: '销售订单详情' } },
        { path: 'sales-templates', name: 'SalesTemplateList', component: () => import('../views/sales/SalesTemplateList.vue'), meta: { title: '销售单模板' } },
        { path: 'service-orders', name: 'ServiceOrderList', component: () => import('../views/service/ServiceOrderList.vue'), meta: { title: '服务工单' } },
        { path: 'service-orders/:id', name: 'ServiceOrderDetail', component: () => import('../views/service/ServiceOrderDetail.vue'), meta: { title: '服务工单详情' } },
        { path: 'service-templates', name: 'ServiceTemplateList', component: () => import('../views/service/ServiceTemplateList.vue'), meta: { title: '服务工单模板' } },
        { path: 'inventory', name: 'InventoryList', component: () => import('../views/inventory/InventoryList.vue'), meta: { title: '门店库存' } },
        { path: 'stock-transfers', name: 'StockTransferList', component: () => import('../views/inventory/StockTransferList.vue'), meta: { title: '调货入库' } },
        { path: 'purchase-orders', name: 'PurchaseOrderList', component: () => import('../views/purchase/PurchaseOrderList.vue'), meta: { title: '门店采购' } },
        { path: 'purchase-orders/create', name: 'PurchaseOrderCreate', component: () => import('../views/purchase/PurchaseOrderForm.vue'), meta: { title: '新建采购单' } },
        { path: 'purchase-orders/:id', name: 'PurchaseOrderDetail', component: () => import('../views/purchase/PurchaseOrderDetail.vue'), meta: { title: '采购单详情' } },
        { path: 'surveillance', name: 'SurveillanceList', component: () => import('../views/surveillance/SurveillanceList.vue'), meta: { title: '监控管理' } },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  if (!getToken()) {
    redirectToPortal()
    return false
  }
  const ok = await ensureSession()
  if (!ok) {
    clearToken()
    redirectToPortal()
    return false
  }
  return true
})

export default router
