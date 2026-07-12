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

export interface FlatCategory {
  id: number
  name: string
  count?: number
  level: number
}

/** 递归展平分类树（跳过隐藏），用于侧栏/下拉全量展示 */
export function flattenCategoryTree(
  tree: CategoryItem[],
  level = 0,
  opts?: { skipHidden?: boolean },
): FlatCategory[] {
  const skipHidden = opts?.skipHidden !== false
  const out: FlatCategory[] = []
  for (const cat of tree) {
    if (skipHidden && cat.showStatus === 0) continue
    out.push({
      id: cat.id,
      name: cat.name,
      count: cat.productCount,
      level,
    })
    if (cat.children?.length) {
      out.push(...flattenCategoryTree(cat.children, level + 1, opts))
    }
  }
  return out
}

/** 自身 + 全部子孙分类 ID（用于按父分类筛选商品） */
export function collectCategoryAndDescendantIds(tree: CategoryItem[], rootId: number): number[] {
  if (!rootId) return []
  function findNode(list: CategoryItem[]): CategoryItem | null {
    for (const c of list) {
      if (c.id === rootId) return c
      if (c.children?.length) {
        const hit = findNode(c.children)
        if (hit) return hit
      }
    }
    return null
  }
  const root = findNode(tree)
  if (!root) return [rootId]
  const ids: number[] = []
  function walk(node: CategoryItem) {
    ids.push(node.id)
    for (const child of node.children || []) walk(child)
  }
  walk(root)
  return ids
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
