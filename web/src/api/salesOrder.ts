import client, { unwrap, type PageData } from './client'

export interface OrderLine {
  skuId: number
  productName: string
  skuCode?: string
  specLabel?: string
  quantity: number
  unitPrice: number
  totalAmount?: number
}

export interface SalesOrder {
  id: number
  storeId: number
  orderNo: string
  fulfillmentType: string
  status: string
  customerName?: string
  customerPhone?: string
  shippingAddress?: string
  totalAmount: number
  payStatus: string
  needProcurement: boolean
  remark?: string
  items?: OrderLine[]
}

export interface SalesOrderInput {
  storeId: number
  fulfillmentType?: string
  customerName?: string
  customerPhone?: string
  shippingAddress?: string
  needProcurement?: boolean
  remark?: string
  items: OrderLine[]
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

export async function createPurchaseFromSales(salesId: number, data: Record<string, unknown>) {
  const res = await client.post(`/sales-orders/${salesId}/purchase-orders`, data)
  return unwrap(res)
}
