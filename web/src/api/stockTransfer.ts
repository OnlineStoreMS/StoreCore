import client, { unwrap, type PageData } from './client'

export interface StockTransferItem {
  id?: number
  skuId: number
  skuCode?: string
  productName: string
  specLabel?: string
  quantity: number
}

export interface StockTransferOrder {
  id: number
  storeId: number
  orderNo: string
  status: string
  expectedAt?: string
  remark?: string
  reminderEnabled?: boolean
  reminderAt?: string
  reminderChannel?: string
  reminderStatus?: string
  items?: StockTransferItem[]
  createdAt?: string
}

export interface StockTransferInput {
  storeId: number
  expectedAt?: string
  remark?: string
  items: StockTransferItem[]
  reminderEnabled?: boolean
  reminderAt?: string
}

export async function listStockTransfers(storeId?: number, page = 1, pageSize = 20) {
  const res = await client.get('/stock-transfers', { params: { storeId, page, pageSize } })
  return unwrap<PageData<StockTransferOrder>>(res)
}

export async function getStockTransfer(id: number) {
  const res = await client.get(`/stock-transfers/${id}`)
  return unwrap<StockTransferOrder>(res)
}

export async function createStockTransfer(data: StockTransferInput) {
  const res = await client.post('/stock-transfers', data)
  return unwrap<StockTransferOrder>(res)
}

export async function confirmStockTransfer(id: number) {
  const res = await client.post(`/stock-transfers/${id}/confirm`)
  return unwrap<StockTransferOrder>(res)
}

export async function cancelStockTransfer(id: number) {
  const res = await client.post(`/stock-transfers/${id}/cancel`)
  return unwrap<StockTransferOrder>(res)
}
