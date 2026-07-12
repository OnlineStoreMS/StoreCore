import client, { unwrap, type PageData } from './client'

export interface InventoryRow {
  id: number
  storeId: number
  skuId: number
  skuCode: string
  productName: string
  specLabel?: string
  pic?: string
  quantity: number
  safetyStock: number
}

export interface StoreSkuQty {
  skuId: number
  quantity: number
}

export async function listInventories(
  storeId?: number,
  keyword = '',
  page = 1,
  pageSize = 20,
  filters?: { brandId?: number; categoryId?: number; groupId?: number },
) {
  const res = await client.get('/inventories', {
    params: {
      storeId,
      keyword,
      page,
      pageSize,
      brandId: filters?.brandId || undefined,
      categoryId: filters?.categoryId || undefined,
      groupId: filters?.groupId || undefined,
    },
  })
  return unwrap<PageData<InventoryRow>>(res)
}

export async function listInventoriesByStore(storeId: number) {
  const res = await client.get('/inventories/by-store', { params: { storeId } })
  return unwrap<InventoryRow[]>(res)
}

export async function listInventoryBySkus(storeId: number, skuIds: number[]) {
  const res = await client.get('/inventories/by-skus', {
    params: { storeId, skuIds: skuIds.join(',') },
  })
  return unwrap<StoreSkuQty[]>(res)
}

export async function adjustInventory(data: {
  storeId: number
  skuId: number
  skuCode?: string
  productName?: string
  specLabel?: string
  pic?: string
  quantity: number
  safetyStock?: number
}) {
  const res = await client.post('/inventories/adjust', data)
  return unwrap<InventoryRow>(res)
}
