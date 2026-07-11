import client, { unwrap, type PageData } from './client'

export interface Store {
  id: number
  tenantId: number
  code: string
  name: string
  shortName?: string
  status: number
  phone?: string
  province?: string
  city?: string
  district?: string
  address?: string
  businessHours?: string
  brandLogo?: string
  coverPic?: string
  photos?: string[]
  guideText?: string
  guidePics?: string[]
  longitude?: number
  latitude?: number
  mapLabel?: string
  remark?: string
}

export async function listStores(keyword = '', page = 1, pageSize = 20) {
  const res = await client.get('/stores', { params: { keyword, page, pageSize } })
  return unwrap<PageData<Store>>(res)
}

export async function createStore(data: Partial<Store>) {
  const res = await client.post('/stores', data)
  return unwrap<Store>(res)
}

export async function updateStore(id: number, data: Partial<Store>) {
  const res = await client.put(`/stores/${id}`, data)
  return unwrap<Store>(res)
}

export async function deleteStore(id: number) {
  const res = await client.delete(`/stores/${id}`)
  return unwrap<{ deleted: boolean }>(res)
}
