import client, { unwrap, type PageData } from './client'

export interface ServiceCategory {
  id: number
  parentId: number
  name: string
  sort: number
  status: number
  itemCount?: number
  children?: ServiceCategory[]
}

export interface ServiceItem {
  id: number
  categoryId: number
  categoryName?: string
  code?: string
  name: string
  description?: string
  price: number
  durationMin?: number
  pic?: string
  sort: number
  status: number
}

export async function listServiceCategoryTree() {
  const res = await client.get('/service-categories/tree')
  return unwrap<ServiceCategory[]>(res)
}

export async function createServiceCategory(data: {
  parentId?: number
  name: string
  sort?: number
  status?: number
}) {
  const res = await client.post('/service-categories', data)
  return unwrap<ServiceCategory>(res)
}

export async function updateServiceCategory(
  id: number,
  data: { parentId?: number; name: string; sort?: number; status?: number },
) {
  const res = await client.put(`/service-categories/${id}`, data)
  return unwrap<ServiceCategory>(res)
}

export async function deleteServiceCategory(id: number) {
  const res = await client.delete(`/service-categories/${id}`)
  return unwrap<null>(res)
}

export async function listServiceItems(params: {
  categoryId?: number
  keyword?: string
  status?: number
  page?: number
  pageSize?: number
}) {
  const res = await client.get('/service-items', { params })
  return unwrap<PageData<ServiceItem>>(res)
}

export async function createServiceItem(data: Partial<ServiceItem> & { categoryId: number; name: string }) {
  const res = await client.post('/service-items', data)
  return unwrap<ServiceItem>(res)
}

export async function updateServiceItem(id: number, data: Partial<ServiceItem> & { categoryId: number; name: string }) {
  const res = await client.put(`/service-items/${id}`, data)
  return unwrap<ServiceItem>(res)
}

export async function deleteServiceItem(id: number) {
  const res = await client.delete(`/service-items/${id}`)
  return unwrap<null>(res)
}

export interface ServicePriceListResult {
  html: string
  itemCount: number
  storeName: string
}

export async function previewServicePriceList(data: {
  storeId: number
  templateId?: number
  serviceItemIds: number[]
  groupByCategory?: boolean
}) {
  const res = await client.post('/service-price-list/preview', data)
  return unwrap<ServicePriceListResult>(res)
}
