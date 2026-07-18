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
  originalPrice?: number
  discount?: number
  unitPrice: number
  totalAmount: number
  durationMin?: number
  pic?: string
}

export interface ServiceProcessMedia {
  url: string
  mediaType: 'image' | 'video' | string
}

export interface ServiceProcessRecord {
  id: number
  serviceOrderId: number
  phase: 'before' | 'after' | string
  note?: string
  media?: ServiceProcessMedia[]
  createdAt?: string
  updatedAt?: string
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
  reportHtml?: string
  reminderEnabled?: boolean
  reminderAt?: string
  reminderChannel?: string
  reminderStatus?: string
  remark?: string
  items?: ServiceOrderItem[]
  processRecords?: ServiceProcessRecord[]
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
  originalPrice?: number
  discount?: number
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

export async function listServiceOrders(params?: {
  storeId?: number
  status?: string
  payStatus?: string
  orderMode?: string
  keyword?: string
  page?: number
  pageSize?: number
}) {
  const res = await client.get('/service-orders', {
    params: {
      storeId: params?.storeId,
      status: params?.status,
      payStatus: params?.payStatus,
      orderMode: params?.orderMode,
      keyword: params?.keyword,
      page: params?.page ?? 1,
      pageSize: params?.pageSize ?? 20,
    },
  })
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
  data: { paymentMethod?: string; paymentProofUrl?: string; paidAt?: string },
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

export async function mergeServiceReceipt(ids: number[], includeReport = false) {
  const res = await client.post('/service-orders/merge-receipt', { ids, includeReport })
  return unwrap<ServiceMergeReceiptResult>(res)
}

export async function createServiceProcessRecord(
  orderId: number,
  data: { phase: 'before' | 'after'; note?: string; media?: ServiceProcessMedia[] },
) {
  const res = await client.post(`/service-orders/${orderId}/process-records`, data)
  return unwrap<ServiceOrder>(res)
}

export async function updateServiceProcessRecord(
  orderId: number,
  recordId: number,
  data: { phase: 'before' | 'after'; note?: string; media?: ServiceProcessMedia[] },
) {
  const res = await client.put(`/service-orders/${orderId}/process-records/${recordId}`, data)
  return unwrap<ServiceOrder>(res)
}

export async function deleteServiceProcessRecord(orderId: number, recordId: number) {
  const res = await client.delete(`/service-orders/${orderId}/process-records/${recordId}`)
  return unwrap<ServiceOrder>(res)
}

export async function refreshServiceReport(id: number) {
  const res = await client.post(`/service-orders/${id}/refresh-report`)
  return unwrap<ServiceOrder>(res)
}

export async function serviceDocBundle(
  id: number,
  opts?: { includeReceipt?: boolean; includeReport?: boolean },
) {
  const res = await client.post(`/service-orders/${id}/doc-bundle`, {
    includeReceipt: opts?.includeReceipt ?? true,
    includeReport: opts?.includeReport ?? true,
  })
  return unwrap<{ html: string }>(res)
}
