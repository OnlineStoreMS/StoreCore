import client, { unwrap, type PageData } from './client'

export interface Supplier {
  id: number
  code: string
  name: string
  shortName?: string
}

export async function listSuppliers(keyword = '', page = 1, pageSize = 50) {
  const res = await client.get('/suppliers', { params: { keyword, page, pageSize } })
  return unwrap<PageData<Supplier>>(res)
}
