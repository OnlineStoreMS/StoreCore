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
  completed: '已完成',
  cancelled: '已取消',
}

export const purchaseStatusMap: Record<string, string> = {
  draft: '草稿',
  submitted: '已提交',
  received: '已到货',
  cancelled: '已取消',
}

export const serviceTypeOptions = [
  { value: 'repair', label: '维修' },
  { value: 'maintenance', label: '保养' },
  { value: 'appointment', label: '预约服务' },
  { value: 'pack_bike', label: '自行车打包' },
]
