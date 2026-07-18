import client, { unwrap, type PageData } from './client'

export interface OrderLine {
  itemType?: 'product' | 'service'
  skuId?: number
  serviceItemId?: number
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

export interface PosOrder {
  id: number
  storeId: number
  orderNo: string
  status: string
  paymentMethod: string
  payStatus: string
  originalAmount?: number
  discountAmount?: number
  totalAmount: number
  paidAmount: number
  customerName?: string
  customerPhone?: string
  serviceOrderId?: number
  serviceOrderNo?: string
  receiptHtml?: string
  paidAt?: string
  createdAt?: string
  items?: OrderLine[]
}

export async function listPosOrders(params?: {
  storeId?: number
  status?: string
  payStatus?: string
  paymentMethod?: string
  keyword?: string
  page?: number
  pageSize?: number
}) {
  const res = await client.get('/pos-orders', {
    params: {
      storeId: params?.storeId,
      status: params?.status,
      payStatus: params?.payStatus,
      paymentMethod: params?.paymentMethod,
      keyword: params?.keyword,
      page: params?.page ?? 1,
      pageSize: params?.pageSize ?? 20,
    },
  })
  return unwrap<PageData<PosOrder>>(res)
}

export async function getPosOrder(id: number) {
  const res = await client.get(`/pos-orders/${id}`)
  return unwrap<PosOrder>(res)
}

export async function createPosOrder(data: {
  storeId: number
  paymentMethod?: string
  isPreview?: boolean
  isHeld?: boolean
  resumeOrderId?: number
  receiptType?: string
  customerName?: string
  customerPhone?: string
  remark?: string
  serviceOrderId?: number
  items: OrderLine[]
}) {
  const res = await client.post('/pos-orders', data)
  return unwrap<PosOrder>(res)
}

export async function markPosPaid(id: number, paymentMethod?: string) {
  const res = await client.post(`/pos-orders/${id}/mark-paid`, {
    paymentMethod: paymentMethod || undefined,
  })
  return unwrap<PosOrder>(res)
}

export async function deletePosOrder(id: number) {
  const res = await client.delete(`/pos-orders/${id}`)
  return unwrap<null>(res)
}

/** 可回载继续收银的状态 */
export function canResumePosOrder(o: Pick<PosOrder, 'status' | 'payStatus'>) {
  if (o.payStatus === 'paid') return false
  return o.status === 'preview' || o.status === 'held' || o.status === 'pending'
}
