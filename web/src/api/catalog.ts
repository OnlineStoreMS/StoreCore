import client, { unwrap, type PageData } from './client'

export interface CategoryItem {
  id: number
  parentId: number
  name: string
  level: number
  sort: number
  showStatus: number
  productCount?: number
  children?: CategoryItem[]
}

export interface CatalogProduct {
  id: number
  name: string
  pic: string
  price: number
  stock: number
  skuCount?: number
  categoryId: number
  categoryName?: string
  materialCode?: string
  publishStatus: number
}

export interface CatalogSku {
  id: number
  skuCode: string
  specs: Record<string, string>
  price: number
  stock: number
  pic?: string
}

export interface ProductSkusDetail {
  id: number
  name: string
  pic?: string
  skuCount: number
  price: number
  skus: CatalogSku[]
}

export async function listCategories() {
  const res = await client.get('/product-catalog/categories')
  return unwrap<CategoryItem[]>(res)
}

export async function listCatalogProducts(params: {
  categoryId?: number
  keyword?: string
  page?: number
  pageSize?: number
}) {
  const res = await client.get('/product-catalog/products', { params })
  return unwrap<PageData<CatalogProduct>>(res)
}

export async function getProductSkus(productId: number) {
  const res = await client.get(`/product-catalog/products/${productId}/skus`)
  return unwrap<ProductSkusDetail>(res)
}

export function formatSpecLabel(specs: Record<string, string>): string {
  const values = Object.values(specs || {}).filter(Boolean)
  return values.length ? values.join(' / ') : '-'
}

export function resolvePic(...candidates: (string | undefined)[]): string {
  for (const c of candidates) {
    if (c?.trim()) return c.trim()
  }
  return ''
}
