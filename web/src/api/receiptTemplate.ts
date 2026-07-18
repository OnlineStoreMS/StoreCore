import client, { unwrap, type PageData } from './client'

export interface ReceiptTemplate {
  id: number
  storeId: number
  name: string
  receiptType: string
  headerTitle: string
  headerSubtitle: string
  headerExtra: string
  footerThanks: string
  footerExtra: string
  showSkuPic: boolean
  showStorePhone: boolean
  showStoreAddress: boolean
  showBusinessHours: boolean
  showBrandLogo: boolean
  showCoverPic: boolean
  showGuideText: boolean
  showMapLabel: boolean
  showDescription?: boolean
  showDuration?: boolean
  isDefault: boolean
  status: number
  createdAt?: string
  updatedAt?: string
}

export interface ReceiptTemplateInput {
  storeId?: number
  name: string
  receiptType?: string
  headerTitle?: string
  headerSubtitle?: string
  headerExtra?: string
  footerThanks?: string
  footerExtra?: string
  showSkuPic?: boolean
  showStorePhone?: boolean
  showStoreAddress?: boolean
  showBusinessHours?: boolean
  showBrandLogo?: boolean
  showCoverPic?: boolean
  showGuideText?: boolean
  showMapLabel?: boolean
  showDescription?: boolean
  showDuration?: boolean
  isDefault?: boolean
  status?: number
}

export async function listReceiptTemplates(storeId?: number, page = 1, pageSize = 50, receiptType?: string) {
  const res = await client.get('/receipt-templates', { params: { storeId, page, pageSize, receiptType } })
  return unwrap<PageData<ReceiptTemplate>>(res)
}

export async function getReceiptTemplate(id: number) {
  const res = await client.get(`/receipt-templates/${id}`)
  return unwrap<ReceiptTemplate>(res)
}

export async function createReceiptTemplate(data: ReceiptTemplateInput) {
  const res = await client.post('/receipt-templates', data)
  return unwrap<ReceiptTemplate>(res)
}

export async function updateReceiptTemplate(id: number, data: ReceiptTemplateInput) {
  const res = await client.put(`/receipt-templates/${id}`, data)
  return unwrap<ReceiptTemplate>(res)
}

export async function deleteReceiptTemplate(id: number) {
  const res = await client.delete(`/receipt-templates/${id}`)
  return unwrap<null>(res)
}
