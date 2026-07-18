import client, { unwrap, type PageData } from './client'

export interface ServiceOrderItem {
  id?: number
  itemType?: 'service' | 'product' | string
  serviceItemId?: number
  serviceName?: string
  serviceCode?: string
  skuId?: number
  skuCode?: string
  productName?: string
  specLabel?: string
  quantity: number
  unitPrice: number
  totalAmount: number
  durationMin?: number
  pic?: string
}

export interface ServiceOrder {
  id: number
  storeId: number
  orderNo: string
  orderMode: 'instant' | 'appointment' | string
  serviceType?: string
  status: string
  payStatus?: string
  customerName?: string
  customerPhone?: string
  deviceInfo?: string
  faultDesc?: string
  appointmentAt?: string
  engineerName?: string
  estimatedAmount: number
  posOrderId?: number
  posOrderNo?: string
  salesOrderId?: number
  salesOrderNo?: string
  paymentMethod?: string
  paymentProofUrl?: string
  paidAt?: string
  paidBy?: number
  receiptHtml?: string
  reminderEnabled?: boolean
  reminderAt?: string
  reminderChannel?: string
  reminderStatus?: string
  remark?: string
  items?: ServiceOrderItem[]
  createdAt?: string
  updatedAt?: string
}

export interface ServiceOrderLineInput {
  itemType?: 'service' | 'product'
  serviceItemId?: number
  skuId?: number
  productName?: string
  skuCode?: string
  specLabel?: string
  pic?: string
  quantity: number
  unitPrice?: number
}

export interface ServiceOrderInput {
  storeId: number
  orderMode: 'instant' | 'appointment'
  customerName?: string
  customerPhone?: string
  deviceInfo?: string
  faultDesc?: string
  appointmentAt?: string
  engineerName?: string
  remark?: string
  items: ServiceOrderLineInput[]
  reminderEnabled?: boolean
  reminderAt?: string
}

export interface ServiceMergeReceiptResult {
  html: string
  totalAmount: number
  orderNos: string[]
}

export async function listServiceOrders(storeId?: number, page = 1, pageSize = 20) {
  const res = await client.get('/service-orders', { params: { storeId, page, pageSize } })
  return unwrap<PageData<ServiceOrder>>(res)
}

export async function getServiceOrder(id: number) {
  const res = await client.get(`/service-orders/${id}`)
  return unwrap<ServiceOrder>(res)
}

export async function createServiceOrder(data: ServiceOrderInput) {
  const res = await client.post('/service-orders', data)
  return unwrap<ServiceOrder>(res)
}

export async function updateServiceOrder(id: number, data: ServiceOrderInput) {
  const res = await client.put(`/service-orders/${id}`, data)
  return unwrap<ServiceOrder>(res)
}

export async function updateServiceStatus(id: number, status: string) {
  const res = await client.post(`/service-orders/${id}/status`, { status })
  return unwrap<ServiceOrder>(res)
}

export async function markServicePaid(
  id: number,
  data: { paymentMethod?: string; paymentProofUrl?: string },
) {
  const res = await client.post(`/service-orders/${id}/mark-paid`, data)
  return unwrap<ServiceOrder>(res)
}

export async function deleteServiceOrder(id: number) {
  const res = await client.delete(`/service-orders/${id}`)
  return unwrap<null>(res)
}

export async function refreshServiceReceipt(id: number) {
  const res = await client.post(`/service-orders/${id}/refresh-receipt`)
  return unwrap<ServiceOrder>(res)
}

export async function mergeServiceReceipt(ids: number[]) {
  const res = await client.post('/service-orders/merge-receipt', { ids })
  return unwrap<ServiceMergeReceiptResult>(res)
}
