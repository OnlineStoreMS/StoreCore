import client, { unwrap, type PageData } from './client'

export interface OrderLine {
  skuId: number
  productName: string
  skuCode?: string
  specLabel?: string
  pic?: string
  quantity: number
  originalPrice?: number
  discount?: number
  unitPrice: number
  totalAmount?: number
}

export interface SalesServiceLine {
  serviceItemId: number
  serviceName?: string
  serviceCode?: string
  quantity: number
  originalPrice?: number
  discount?: number
  unitPrice?: number
  durationMin?: number
  pic?: string
  totalAmount?: number
}

export interface SalesOrder {
  id: number
  storeId: number
  orderNo: string
  fulfillmentType: string
  status: string
  purchaseStatus?: string
  serviceStatus?: string
  fulfillStatus?: string
  customerName?: string
  customerPhone?: string
  appointmentAt?: string
  pickupPersonName?: string
  pickupPersonPhone?: string
  pickupCode?: string
  deliveryType?: string
  expectedDeliveryAt?: string
  receiverName?: string
  receiverPhone?: string
  shippingAddress?: string
  expressCompany?: string
  expressNo?: string
  expressScheduledAt?: string
  originalAmount?: number
  discountAmount?: number
  totalAmount: number
  payStatus: string
  needProcurement: boolean
  purchaseOrderId?: number
  serviceOrderId?: number
  serviceOrderNo?: string
  receiptHtml?: string
  remark?: string
  items?: OrderLine[]
  serviceItems?: SalesServiceLine[]
}

export interface SalesOrderInput {
  storeId: number
  fulfillmentType?: string
  isPreview?: boolean
  customerName?: string
  customerPhone?: string
  appointmentAt?: string | null
  pickupPersonName?: string
  pickupPersonPhone?: string
  pickupCode?: string
  deliveryType?: string
  expectedDeliveryAt?: string | null
  receiverName?: string
  receiverPhone?: string
  shippingAddress?: string
  expressCompany?: string
  expressNo?: string
  expressScheduledAt?: string | null
  needProcurement?: boolean
  remark?: string
  items: OrderLine[]
  serviceItems?: SalesServiceLine[]
}

export async function listSalesOrders(params?: { storeId?: number; status?: string; page?: number; pageSize?: number }) {
  const res = await client.get('/sales-orders', { params })
  return unwrap<PageData<SalesOrder>>(res)
}

export async function getSalesOrder(id: number) {
  const res = await client.get(`/sales-orders/${id}`)
  return unwrap<SalesOrder>(res)
}

export async function createSalesOrder(data: SalesOrderInput) {
  const res = await client.post('/sales-orders', data)
  return unwrap<SalesOrder>(res)
}

export async function updateSalesOrder(id: number, data: SalesOrderInput) {
  const res = await client.put(`/sales-orders/${id}`, data)
  return unwrap<SalesOrder>(res)
}

export async function confirmSalesOrder(id: number) {
  const res = await client.post(`/sales-orders/${id}/confirm`)
  return unwrap<SalesOrder>(res)
}

export async function cancelSalesOrder(id: number) {
  const res = await client.post(`/sales-orders/${id}/cancel`)
  return unwrap<SalesOrder>(res)
}

export async function deleteSalesOrder(id: number) {
  const res = await client.delete(`/sales-orders/${id}`)
  return unwrap<null>(res)
}

export async function markSalesReady(id: number) {
  const res = await client.post(`/sales-orders/${id}/mark-ready`)
  return unwrap<SalesOrder>(res)
}

export async function shipSalesOrder(id: number) {
  const res = await client.post(`/sales-orders/${id}/ship`)
  return unwrap<SalesOrder>(res)
}

export async function completeSalesOrder(id: number) {
  const res = await client.post(`/sales-orders/${id}/complete`)
  return unwrap<SalesOrder>(res)
}

export async function scheduleSalesExpress(id: number, data: { scheduledAt?: string; company?: string }) {
  const res = await client.post(`/sales-orders/${id}/schedule-express`, data)
  return unwrap<SalesOrder>(res)
}

export async function refreshSalesReceipt(id: number, preview = true) {
  const res = await client.post(`/sales-orders/${id}/refresh-receipt`, null, { params: { preview: preview ? '1' : '0' } })
  return unwrap<SalesOrder>(res)
}

export async function createPurchaseFromSales(salesId: number, data: Record<string, unknown>) {
  const res = await client.post(`/sales-orders/${salesId}/purchase-orders`, data)
  return unwrap(res)
}
