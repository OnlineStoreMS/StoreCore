import { ref, onMounted } from 'vue'
import { listStores, type Store } from '../api/store'

export function useStores() {
  const stores = ref<Store[]>([])
  const storeId = ref<number>()
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      const data = await listStores('', 1, 200)
      stores.value = data.list
      if (!storeId.value && data.list.length) {
        storeId.value = data.list[0].id
      }
    } finally {
      loading.value = false
    }
  }

  onMounted(load)

  return { stores, storeId, loading, reload: load }
}

export const fulfillmentOptions = [
  { value: 'pickup', label: '到店提货' },
  { value: 'delivery', label: '送货上门' },
  { value: 'express', label: '发快递' },
]

export const salesStatusMap: Record<string, string> = {
  draft: '草稿',
  confirmed: '已确认',
  ready: '待提货',
  shipping: '配送中',
  completed: '已完成',
  cancelled: '已取消',
}

export const serviceStatusMap: Record<string, string> = {
  pending: '待处理',
  in_progress: '进行中',
  awaiting_payment: '待付款',
  completed: '已完成',
  cancelled: '已取消',
}

export const servicePayStatusMap: Record<string, string> = {
  unpaid: '未付款',
  paid: '已付款',
}

export const serviceOrderModeOptions = [
  { value: 'appointment', label: '预约' },
  { value: 'instant', label: '即时' },
]

export const serviceOrderModeMap: Record<string, string> = {
  appointment: '预约',
  instant: '即时',
}

export const reminderStatusMap: Record<string, string> = {
  none: '未开启',
  pending: '待发送',
  sent: '已发送',
  failed: '发送失败',
}

export const purchaseStatusMap: Record<string, string> = {
  draft: '草稿',
  submitted: '已提交',
  received: '已到货',
  cancelled: '已取消',
}
