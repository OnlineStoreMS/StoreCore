import client, { unwrap, type PageData } from './client'
import type { OrderLine } from './salesOrder'

export interface PurchaseOrder {
  id: number
  storeId: number
  poNo: string
  purchaseType: string
  supplierId?: number
  supplierName?: string
  refSalesOrderId?: number
  status: string
  totalAmount: number
  remark?: string
  items?: OrderLine[]
}

export interface PurchaseOrderInput {
  storeId: number
  purchaseType?: string
  supplierId?: number
  supplierName?: string
  refSalesOrderId?: number
  remark?: string
  items: OrderLine[]
}

export async function listPurchaseOrders(storeId?: number, page = 1, pageSize = 20) {
  const res = await client.get('/purchase-orders', { params: { storeId, page, pageSize } })
  return unwrap<PageData<PurchaseOrder>>(res)
}

export async function getPurchaseOrder(id: number) {
  const res = await client.get(`/purchase-orders/${id}`)
  return unwrap<PurchaseOrder>(res)
}

export async function createPurchaseOrder(data: PurchaseOrderInput) {
  const res = await client.post('/purchase-orders', data)
  return unwrap<PurchaseOrder>(res)
}

export async function submitPurchaseOrder(id: number) {
  const res = await client.post(`/purchase-orders/${id}/submit`)
  return unwrap<PurchaseOrder>(res)
}

export async function receivePurchaseOrder(id: number) {
  const res = await client.post(`/purchase-orders/${id}/receive`)
  return unwrap<PurchaseOrder>(res)
}

export async function cancelPurchaseOrder(id: number) {
  const res = await client.post(`/purchase-orders/${id}/cancel`)
  return unwrap<PurchaseOrder>(res)
}
