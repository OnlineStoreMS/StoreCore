import client, { unwrap, type PageData } from './client'

export interface ServiceOrderItem {
  id?: number
  serviceItemId: number
  serviceName: string
  serviceCode?: string
  quantity: number
  unitPrice: number
  totalAmount: number
  durationMin?: number
}

export interface ServiceOrder {
  id: number
  storeId: number
  orderNo: string
  orderMode: 'instant' | 'appointment' | string
  serviceType?: string
  status: string
  customerName?: string
  customerPhone?: string
  deviceInfo?: string
  faultDesc?: string
  appointmentAt?: string
  engineerName?: string
  estimatedAmount: number
  reminderEnabled?: boolean
  reminderAt?: string
  reminderChannel?: string
  reminderStatus?: string
  remark?: string
  items?: ServiceOrderItem[]
  createdAt?: string
}

export interface ServiceOrderInput {
  storeId: number
  orderMode: 'instant' | 'appointment'
  customerName?: string
  customerPhone?: string
  deviceInfo?: string
  faultDesc?: string
  appointmentAt?: string
  engineerName?: string
  remark?: string
  items: { serviceItemId: number; quantity: number }[]
  reminderEnabled?: boolean
  reminderAt?: string
}

export async function listServiceOrders(storeId?: number, page = 1, pageSize = 20) {
  const res = await client.get('/service-orders', { params: { storeId, page, pageSize } })
  return unwrap<PageData<ServiceOrder>>(res)
}

export async function getServiceOrder(id: number) {
  const res = await client.get(`/service-orders/${id}`)
  return unwrap<ServiceOrder>(res)
}

export async function createServiceOrder(data: ServiceOrderInput) {
  const res = await client.post('/service-orders', data)
  return unwrap<ServiceOrder>(res)
}

export async function updateServiceStatus(id: number, status: string) {
  const res = await client.post(`/service-orders/${id}/status`, { status })
  return unwrap<ServiceOrder>(res)
}
