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

export interface BrandItem {
  id: number
  name: string
}

export interface GroupItem {
  id: number
  name: string
}

export interface CatalogProduct {
  id: number
  name: string
  pic: string
  price: number
  stock: number
  skuCount?: number
  brandId?: number
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

export async function listBrands() {
  const res = await client.get('/product-catalog/brands')
  return unwrap<BrandItem[]>(res)
}

export async function listGroups() {
  const res = await client.get('/product-catalog/groups')
  return unwrap<GroupItem[]>(res)
}

export async function listCatalogProducts(params: {
  categoryId?: number
  brandId?: number
  groupId?: number
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

/** 规格值展示：仅值，去掉「颜色分类：」等前缀 */
export function formatSpecLabel(specs: Record<string, string>): string {
  const values = Object.values(specs || {}).filter(Boolean)
  return values.length ? values.join(' / ') : '-'
}

export function displaySpecValues(label?: string, specs?: Record<string, string>): string {
  if (specs && Object.keys(specs).length) {
    return formatSpecLabel(specs)
  }
  const raw = (label || '').trim()
  if (!raw) return '-'
  const parts = raw.split(/[/|｜,，]/).map((s) => s.trim()).filter(Boolean)
  const values = parts.map((part) => {
    const m = part.match(/^[^:：]+[:：]\s*(.+)$/)
    return (m ? m[1] : part).trim()
  }).filter(Boolean)
  return values.length ? values.join(' / ') : '-'
}

export function resolvePic(...candidates: (string | undefined)[]): string {
  for (const c of candidates) {
    if (c?.trim()) return c.trim()
  }
  return ''
}
