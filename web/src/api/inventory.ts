import client, { unwrap, type PageData } from './client'

export interface InventoryRow {
  id: number
  storeId: number
  skuId: number
  skuCode: string
  productName: string
  specLabel?: string
  quantity: number
  safetyStock: number
}

export async function listInventories(storeId?: number, keyword = '', page = 1, pageSize = 20) {
  const res = await client.get('/inventories', { params: { storeId, keyword, page, pageSize } })
  return unwrap<PageData<InventoryRow>>(res)
}

export async function adjustInventory(data: {
  storeId: number
  skuId: number
  skuCode?: string
  productName?: string
  specLabel?: string
  quantity: number
  safetyStock?: number
}) {
  const res = await client.post('/inventories/adjust', data)
  return unwrap<InventoryRow>(res)
}
