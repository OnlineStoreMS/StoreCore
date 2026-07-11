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
  receiptHtml?: string
  paidAt?: string
  createdAt?: string
  items?: OrderLine[]
}

export async function listPosOrders(storeId?: number, page = 1, pageSize = 20) {
  const res = await client.get('/pos-orders', { params: { storeId, page, pageSize } })
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
  receiptType?: string
  customerName?: string
  customerPhone?: string
  remark?: string
  items: OrderLine[]
}) {
  const res = await client.post('/pos-orders', data)
  return unwrap<PosOrder>(res)
}

export async function markPosPaid(id: number) {
  const res = await client.post(`/pos-orders/${id}/mark-paid`)
  return unwrap<PosOrder>(res)
}

export async function deletePosOrder(id: number) {
  const res = await client.delete(`/pos-orders/${id}`)
  return unwrap<null>(res)
}
