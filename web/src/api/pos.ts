import client, { unwrap, type PageData } from './client'
import type { ProductSkuSearchItem } from './productSku'

export interface OrderLine {
  skuId: number
  productName: string
  skuCode?: string
  specLabel?: string
  quantity: number
  unitPrice: number
}

export interface PosOrder {
  id: number
  storeId: number
  orderNo: string
  status: string
  paymentMethod: string
  payStatus: string
  totalAmount: number
  paidAmount: number
  receiptHtml?: string
  items?: OrderLine[]
}

export async function listPosOrders(storeId?: number, page = 1, pageSize = 20) {
  const res = await client.get('/pos-orders', { params: { storeId, page, pageSize } })
  return unwrap<PageData<PosOrder>>(res)
}

export async function createPosOrder(data: {
  storeId: number
  paymentMethod: string
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

export type SkuSearchItem = ProductSkuSearchItem
