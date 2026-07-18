<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Plus } from '@element-plus/icons-vue'
import {
  createServiceProcessRecord,
  deleteServiceOrder,
  deleteServiceProcessRecord,
  getServiceOrder,
  markServicePaid,
  refreshServiceReceipt,
  refreshServiceReport,
  serviceDocBundle,
  updateServiceOrder,
  updateServiceProcessRecord,
  updateServiceStatus,
  type ServiceOrder,
  type ServiceOrderItem,
  type ServiceProcessRecord,
} from '../../api/serviceOrder'
import type { MediaItem } from '../../api/upload'
import MediaUploadField from '../../components/MediaUploadField.vue'
import {
  listServiceCategoryTree,
  listServiceItems,
  type ServiceCategory,
  type ServiceItem,
} from '../../api/serviceCatalog'
import OrderLineEditor, { type OrderLine } from '../../components/OrderLineEditor.vue'
import PaymentProofField from '../../components/PaymentProofField.vue'
import PosReceiptPanel from '../../components/PosReceiptPanel.vue'
import { paidAtToApi } from '../../utils/payTimeOcr'
import {
  reminderStatusMap,
  serviceOrderModeMap,
  serviceOrderModeOptions,
  servicePayStatusMap,
  serviceStatusMap,
  useStores,
} from '../../composables/useStores'

interface SelectedLine {
  serviceItemId: number
  name: string
  code?: string
  originalPrice: number
  discount: number
  unitPrice: number
  durationMin?: number
  quantity: number
}

const route = useRoute()
const router = useRouter()
const { stores } = useStores()
const loading = ref(false)
const order = ref<ServiceOrder | null>(null)
const editVisible = ref(false)
const saving = ref(false)
const pickerVisible = ref(false)
const markPaidVisible = ref(false)
const markingPaid = ref(false)
const markPaidForm = reactive({
  paymentMethod: 'transfer' as 'transfer' | 'cash' | 'other',
  paymentProofUrl: '',
  paidAt: '',
})

const processVisible = ref(false)
const processSaving = ref(false)
const processPhase = ref<'before' | 'after'>('before')
const processNote = ref('')
const processMedia = ref<MediaItem[]>([])
const processNextStatus = ref('') // 保存后要推进的状态；空则仅保存纪录
/** 编辑中的纪录 id；0 表示新增 */
const processEditingId = ref(0)
const bundleVisible = ref(false)
const bundleHtml = ref('')
const bundleLoading = ref(false)

const paymentMethodLabel: Record<string, string> = {
  transfer: '转账',
  wechat_transfer: '转账',
  cash: '现金',
  other: '其他',
  pos: '收银台',
  sales: '销售单',
  static_qr: '收款码',
}

const form = reactive({
  orderMode: 'appointment' as 'instant' | 'appointment',
  customerName: '',
  customerPhone: '',
  deviceInfo: '',
  faultDesc: '',
  appointmentAt: '' as string,
  engineerName: '',
  remark: '',
  reminderEnabled: false,
  reminderAt: '' as string,
})
const selected = ref<SelectedLine[]>([])
const productLines = ref<OrderLine[]>([])

const categories = ref<ServiceCategory[]>([])
const flatCategories = computed(() => {
  const out: { id: number; name: string }[] = []
  function walk(list: ServiceCategory[], prefix = '') {
    for (const c of list) {
      if (c.status === 0) continue
      out.push({ id: c.id, name: prefix ? `${prefix} / ${c.name}` : c.name })
      if (c.children?.length) walk(c.children, prefix ? `${prefix} / ${c.name}` : c.name)
    }
  }
  walk(categories.value)
  return out
})
const catalogItems = ref<ServiceItem[]>([])
const activeCategoryId = ref(0)
const keyword = ref('')
const catalogLoading = ref(false)

const estimatedAmount = computed(() => {
  const svc = selected.value.reduce((sum, l) => sum + l.unitPrice * l.quantity, 0)
  const prod = productLines.value.reduce((sum, l) => sum + l.unitPrice * l.quantity, 0)
  return Math.round((svc + prod) * 100) / 100
})
const originalEstimate = computed(() => {
  const svc = selected.value.reduce((sum, l) => sum + (l.originalPrice || l.unitPrice) * l.quantity, 0)
  const prod = productLines.value.reduce((sum, l) => sum + (l.originalPrice || l.unitPrice) * l.quantity, 0)
  return Math.round((svc + prod) * 100) / 100
})

function onServiceDiscountChange(row: SelectedLine) {
  let d = Number(row.discount)
  if (!Number.isFinite(d) || d <= 0) d = 10
  if (d > 10) d = 10
  row.discount = Math.round(d * 100) / 100
  const orig = row.originalPrice > 0 ? row.originalPrice : row.unitPrice
  row.originalPrice = orig
  row.unitPrice = Math.round(orig * (row.discount / 10) * 100) / 100
}

function onServiceUnitPriceChange(row: SelectedLine) {
  const orig = row.originalPrice > 0 ? row.originalPrice : row.unitPrice
  row.originalPrice = orig
  if (orig > 0) {
    row.discount = Math.round((row.unitPrice / orig) * 10 * 100) / 100
  } else {
    row.discount = 10
  }
}

const serviceItemsView = computed(() =>
  (order.value?.items || []).filter((i) => (i.itemType || 'service') !== 'product'),
)
const productItemsView = computed(() =>
  (order.value?.items || []).filter((i) => i.itemType === 'product' || (!!i.skuId && !i.serviceItemId)),
)

const storeName = computed(() => {
  if (!order.value) return '-'
  return stores.value.find((s) => s.id === order.value!.storeId)?.name || `#${order.value.storeId}`
})

const canEdit = computed(() =>
  !!order.value && ['pending', 'in_progress', 'awaiting_payment'].includes(order.value.status),
)
const canStart = computed(() => order.value?.status === 'pending')
const canFinishService = computed(() => order.value?.status === 'in_progress')
const skipCashier = computed(() => {
  if (!order.value) return false
  if (order.value.payStatus === 'paid') return true
  if ((order.value.estimatedAmount || 0) <= 0) return true
  return false
})
const canSettle = computed(() =>
  order.value?.status === 'awaiting_payment' && !skipCashier.value,
)
const canMarkPaid = computed(() =>
  !!order.value
  && order.value.payStatus !== 'paid'
  && ['pending', 'in_progress', 'awaiting_payment', 'completed'].includes(order.value.status)
  && (order.value.estimatedAmount || 0) > 0,
)
/** 已付款且未完成：可改付款时间 / 截图 */
const canEditPayment = computed(() =>
  !!order.value
  && order.value.payStatus === 'paid'
  && ['pending', 'in_progress', 'awaiting_payment'].includes(order.value.status),
)
/** 服务已做完（待付款）且已收款时，可单独确认工单履约完成 */
const canCompleteWork = computed(() =>
  order.value?.status === 'awaiting_payment' && order.value.payStatus === 'paid',
)
/** 已完成工单可重开，改回进行中继续服务 */
const canReopen = computed(() => order.value?.status === 'completed')
const canCancel = computed(() =>
  !!order.value && ['pending', 'in_progress', 'awaiting_payment'].includes(order.value.status),
)
const canDelete = computed(() => !!order.value)
const canEditProcess = computed(() =>
  !!order.value && !['cancelled'].includes(order.value.status),
)
const beforeRecords = computed(() =>
  (order.value?.processRecords || []).filter((r) => r.phase === 'before'),
)
const afterRecords = computed(() =>
  (order.value?.processRecords || []).filter((r) => r.phase === 'after'),
)
const beforeMediaCount = computed(() =>
  beforeRecords.value.reduce((n, r) => n + (r.media?.length || 0), 0),
)
const afterMediaCount = computed(() =>
  afterRecords.value.reduce((n, r) => n + (r.media?.length || 0), 0),
)
const flowTip = computed(() => {
  if (order.value?.salesOrderId) {
    return '关联销售单：开始工单须填服务前纪录，完成服务须填服务后纪录；销售单付款后服务单记为已付款。'
  }
  return '服务过程纪录为必填：开始工单前需记录服务前（图片/视频），完成服务前需记录服务后；完成后可生成服务报告并与票据合并。'
})

const processDialogTitle = computed(() => {
  const phase = processPhase.value === 'before' ? '服务前' : '服务后'
  if (processEditingId.value) return `编辑${phase}过程纪录`
  return `新增${phase}过程纪录`
})

function latestRecord(phase: 'before' | 'after'): ServiceProcessRecord | undefined {
  const list = phase === 'before' ? beforeRecords.value : afterRecords.value
  if (!list.length) return undefined
  return list[list.length - 1]
}

function mediaFromRecord(rec?: ServiceProcessRecord): MediaItem[] {
  return (rec?.media || []).map((m) => ({
    url: m.url,
    mediaType: m.mediaType === 'video' ? 'video' : 'image',
  }))
}

function formatDisplayTime(v?: string) {
  if (!v) return '-'
  return v.replace('T', ' ').slice(0, 16)
}

function toApiTime(v: string) {
  if (!v) return undefined
  return v.length === 16 ? `${v}:00` : v
}

function modeLabel(o: ServiceOrder) {
  return serviceOrderModeMap[o.orderMode] || o.orderMode || '-'
}

async function load() {
  const id = Number(route.params.id)
  if (!id) return
  loading.value = true
  try {
    order.value = await getServiceOrder(id)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

function openEdit() {
  if (!order.value) return
  Object.assign(form, {
    orderMode: (order.value.orderMode === 'instant' ? 'instant' : 'appointment') as 'instant' | 'appointment',
    customerName: order.value.customerName || '',
    customerPhone: order.value.customerPhone || '',
    deviceInfo: order.value.deviceInfo || '',
    faultDesc: order.value.faultDesc || '',
    appointmentAt: order.value.appointmentAt ? formatDisplayTime(order.value.appointmentAt).replace(' ', ' ') + ':00' : '',
    engineerName: order.value.engineerName || '',
    remark: order.value.remark || '',
    reminderEnabled: !!order.value.reminderEnabled,
    reminderAt: order.value.reminderAt ? formatDisplayTime(order.value.reminderAt) + ':00' : '',
  })
  // normalize datetime for picker
  if (order.value.appointmentAt) {
    form.appointmentAt = order.value.appointmentAt.replace('T', ' ').slice(0, 19)
  }
  if (order.value.reminderAt) {
    form.reminderAt = order.value.reminderAt.replace('T', ' ').slice(0, 19)
  }
  selected.value = (order.value.items || [])
    .filter((it: ServiceOrderItem) => (it.itemType || 'service') !== 'product')
    .map((it: ServiceOrderItem) => {
      const unit = Number(it.unitPrice) || 0
      const orig = Number(it.originalPrice) > 0 ? Number(it.originalPrice) : unit
      const disc = Number(it.discount) > 0 ? Number(it.discount) : 10
      return {
        serviceItemId: it.serviceItemId || 0,
        name: it.serviceName || '',
        code: it.serviceCode,
        originalPrice: orig,
        discount: disc,
        unitPrice: unit,
        durationMin: it.durationMin,
        quantity: it.quantity,
      }
    })
  productLines.value = (order.value.items || [])
    .filter((it: ServiceOrderItem) => it.itemType === 'product' || (!!it.skuId && !it.serviceItemId))
    .map((it: ServiceOrderItem) => {
      const unit = Number(it.unitPrice) || 0
      const orig = Number(it.originalPrice) > 0 ? Number(it.originalPrice) : unit
      const disc = Number(it.discount) > 0 ? Number(it.discount) : 10
      return {
        skuId: it.skuId || 0,
        productName: it.productName || '',
        skuCode: it.skuCode,
        specLabel: it.specLabel,
        pic: it.pic,
        quantity: it.quantity,
        originalPrice: orig,
        discount: disc,
        unitPrice: unit,
      }
    })
  editVisible.value = true
}

async function openPicker() {
  pickerVisible.value = true
  if (!categories.value.length) {
    categories.value = await listServiceCategoryTree()
  }
  await loadCatalog()
}

async function loadCatalog() {
  catalogLoading.value = true
  try {
    const data = await listServiceItems({
      categoryId: activeCategoryId.value || undefined,
      keyword: keyword.value.trim() || undefined,
      status: 1,
      page: 1,
      pageSize: 50,
    })
    catalogItems.value = data.list
  } finally {
    catalogLoading.value = false
  }
}

function selectCategory(id: number) {
  activeCategoryId.value = id
  void loadCatalog()
}

function addLine(item: ServiceItem) {
  const existing = selected.value.find((l) => l.serviceItemId === item.id)
  if (existing) {
    existing.quantity += 1
    return
  }
  selected.value.push({
    serviceItemId: item.id,
    name: item.name,
    code: item.code,
    originalPrice: item.price || 0,
    discount: 10,
    unitPrice: item.price || 0,
    durationMin: item.durationMin,
    quantity: 1,
  })
}

function removeLine(index: number) {
  selected.value.splice(index, 1)
}

async function saveEdit() {
  if (!order.value) return
  if (selected.value.length === 0 && productLines.value.length === 0) {
    ElMessage.warning('请至少选择一项服务或商品')
    return
  }
  if (form.orderMode === 'appointment' && !form.appointmentAt) {
    ElMessage.warning('预约工单请填写预约时间')
    return
  }
  saving.value = true
  try {
    await updateServiceOrder(order.value.id, {
      storeId: order.value.storeId,
      orderMode: form.orderMode,
      customerName: form.customerName,
      customerPhone: form.customerPhone,
      deviceInfo: form.deviceInfo,
      faultDesc: form.faultDesc,
      appointmentAt: form.orderMode === 'appointment' ? toApiTime(form.appointmentAt) : undefined,
      engineerName: form.engineerName,
      remark: form.remark,
      items: [
        ...selected.value.map((l) => ({
          itemType: 'service' as const,
          serviceItemId: l.serviceItemId,
          quantity: l.quantity,
          originalPrice: l.originalPrice,
          discount: l.discount,
          unitPrice: l.unitPrice,
        })),
        ...productLines.value.map((l) => ({
          itemType: 'product' as const,
          skuId: l.skuId,
          productName: l.productName,
          skuCode: l.skuCode,
          specLabel: l.specLabel,
          pic: l.pic,
          quantity: l.quantity,
          originalPrice: l.originalPrice,
          discount: l.discount,
          unitPrice: l.unitPrice,
        })),
      ],
      reminderEnabled: form.reminderEnabled,
      reminderAt: form.reminderEnabled ? toApiTime(form.reminderAt) : undefined,
    })
    ElMessage.success('已更新')
    editVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function doRefreshReceipt() {
  if (!order.value) return
  try {
    order.value = await refreshServiceReceipt(order.value.id)
    ElMessage.success('已刷新工单票据')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function doRefreshReport() {
  if (!order.value) return
  try {
    order.value = await refreshServiceReport(order.value.id)
    ElMessage.success('已刷新服务报告')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function openDocBundle() {
  if (!order.value) return
  bundleLoading.value = true
  try {
    const res = await serviceDocBundle(order.value.id, { includeReceipt: true, includeReport: true })
    bundleHtml.value = res.html
    bundleVisible.value = true
  } catch (e) {
    ElMessage.error((e as Error).message || '合并单据失败')
  } finally {
    bundleLoading.value = false
  }
}

/** 优先编辑该阶段最近一条纪录；没有则新增 */
function openProcessDialog(phase: 'before' | 'after', nextStatus = '', forceNew = false) {
  processPhase.value = phase
  processNextStatus.value = nextStatus
  const latest = forceNew ? undefined : latestRecord(phase)
  if (latest) {
    processEditingId.value = latest.id
    processNote.value = latest.note || ''
    processMedia.value = mediaFromRecord(latest)
  } else {
    processEditingId.value = 0
    processNote.value = ''
    processMedia.value = []
  }
  processVisible.value = true
}

function openNewProcessRecord(phase: 'before' | 'after') {
  openProcessDialog(phase, '', true)
}

function openEditProcessRecord(rec: ServiceProcessRecord) {
  processPhase.value = rec.phase === 'after' ? 'after' : 'before'
  processEditingId.value = rec.id
  processNote.value = rec.note || ''
  processMedia.value = mediaFromRecord(rec)
  processNextStatus.value = ''
  processVisible.value = true
}

async function saveProcessRecord() {
  if (!order.value) return
  if (!processMedia.value.length) {
    ElMessage.warning('请至少上传一张图片或视频')
    return
  }
  processSaving.value = true
  try {
    const payload = {
      phase: processPhase.value,
      note: processNote.value.trim(),
      media: processMedia.value,
    }
    if (processEditingId.value) {
      order.value = await updateServiceProcessRecord(order.value.id, processEditingId.value, payload)
      ElMessage.success('过程纪录已更新')
    } else {
      order.value = await createServiceProcessRecord(order.value.id, payload)
      ElMessage.success('过程纪录已新增')
    }
    processVisible.value = false
    const next = processNextStatus.value
    processNextStatus.value = ''
    processEditingId.value = 0
    if (next) {
      await applyStatus(next)
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    processSaving.value = false
  }
}

async function removeProcessRecord(rec: ServiceProcessRecord) {
  if (!order.value) return
  await ElMessageBox.confirm('确认删除该条过程纪录？', '确认', { type: 'warning' })
  try {
    order.value = await deleteServiceProcessRecord(order.value.id, rec.id)
    ElMessage.success('已删除')
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

async function applyStatus(status: string) {
  if (!order.value) return
  const isReopen = status === 'in_progress' && order.value.status === 'completed'
  try {
    await updateServiceStatus(order.value.id, status)
    ElMessage.success(isReopen ? '工单已重开' : '状态已更新')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function setStatus(status: string) {
  if (!order.value) return
  const isReopen = status === 'in_progress' && order.value.status === 'completed'

  // 开始工单：强制服务前过程纪录
  if (status === 'in_progress' && order.value.status === 'pending') {
    if (beforeMediaCount.value < 1) {
      openProcessDialog('before', 'in_progress')
      return
    }
  }
  // 完成服务：强制服务后过程纪录
  if (status === 'awaiting_payment') {
    if (afterMediaCount.value < 1) {
      openProcessDialog('after', 'awaiting_payment')
      return
    }
  }

  const labels: Record<string, string> = {
    in_progress: isReopen
      ? '确认重开工单？状态将改回进行中，可继续服务后再次完成（付款状态不变）'
      : '确认开始工单？状态将变为进行中',
    awaiting_payment: skipCashier.value
      ? '确认服务已完成？已收款，将标记工单为已完成，并生成服务报告'
      : '确认服务已完成？将变为待付款并生成服务报告',
    completed: '确认将工单标记为已完成？',
    cancelled: '确认取消该工单？',
  }
  await ElMessageBox.confirm(labels[status] || '确认操作？', '确认', { type: 'warning' })
  await applyStatus(status)
}

async function remove() {
  if (!order.value) return
  const tips: string[] = [`确认删除服务工单「${order.value.orderNo}」？删除后不可恢复。`]
  if (order.value.posOrderId) {
    tips.push(`将同时删除关联收银订单 ${order.value.posOrderNo || '#' + order.value.posOrderId}。`)
  }
  if (order.value.salesOrderId) {
    tips.push('销售单上的服务工单关联将被清除（销售单本身保留）。')
  }
  try {
    await ElMessageBox.confirm(tips.join('\n'), '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
    await deleteServiceOrder(order.value.id)
    ElMessage.success('已删除')
    router.push('/service-orders')
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message || '删除失败')
  }
}

function goSettle() {
  if (!order.value) return
  router.push({ path: '/pos', query: { serviceOrderId: String(order.value.id) } })
}

function toPickerTime(v?: string) {
  if (!v) return ''
  return v.replace('T', ' ').slice(0, 19)
}

function openMarkPaid(edit = false) {
  if (edit && order.value) {
    const m = order.value.paymentMethod || 'transfer'
    markPaidForm.paymentMethod = (m === 'cash' || m === 'other' ? m : 'transfer') as 'transfer' | 'cash' | 'other'
    markPaidForm.paymentProofUrl = order.value.paymentProofUrl || ''
    markPaidForm.paidAt = toPickerTime(order.value.paidAt)
  } else {
    markPaidForm.paymentMethod = 'transfer'
    markPaidForm.paymentProofUrl = ''
    markPaidForm.paidAt = ''
  }
  markPaidVisible.value = true
}

async function submitMarkPaid() {
  if (!order.value) return
  if (markPaidForm.paymentMethod === 'transfer' && !markPaidForm.paymentProofUrl) {
    ElMessage.warning('转账收款请先上传付款截图')
    return
  }
  const editing = order.value.payStatus === 'paid'
  markingPaid.value = true
  try {
    order.value = await markServicePaid(order.value.id, {
      paymentMethod: markPaidForm.paymentMethod,
      paymentProofUrl: markPaidForm.paymentProofUrl || undefined,
      paidAt: paidAtToApi(markPaidForm.paidAt),
    })
    markPaidVisible.value = false
    ElMessage.success(editing ? '付款信息已更新' : '已确认收款')
  } catch (e) {
    ElMessage.error((e as Error).message || (editing ? '更新付款信息失败' : '确认收款失败'))
  } finally {
    markingPaid.value = false
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading">
    <div class="toolbar">
      <el-button @click="router.push('/service-orders')">返回列表</el-button>
      <el-button v-if="canEdit" type="primary" plain @click="openEdit">编辑</el-button>
      <el-button v-if="canStart" type="primary" @click="setStatus('in_progress')">开始工单</el-button>
      <el-button v-if="canFinishService" type="warning" @click="setStatus('awaiting_payment')">
        {{ skipCashier ? '完成服务' : '完成服务（待付款）' }}
      </el-button>
      <el-button v-if="canCompleteWork" type="warning" @click="setStatus('completed')">确认工单完成</el-button>
      <el-button v-if="canReopen" type="primary" plain @click="setStatus('in_progress')">重开工单</el-button>
      <el-button v-if="canSettle" type="success" @click="goSettle">去收银台结算</el-button>
      <el-button v-if="canMarkPaid" type="success" plain @click="openMarkPaid(false)">确认收款（上传截图）</el-button>
      <el-button v-if="canEditPayment" type="success" plain @click="openMarkPaid(true)">修改付款信息</el-button>
      <el-button v-if="canEditProcess && canStart" type="primary" plain @click="openProcessDialog('before')">
        {{ beforeRecords.length ? '编辑服务前' : '服务前纪录' }}
      </el-button>
      <el-button
        v-if="canEditProcess && canStart && beforeRecords.length"
        plain
        @click="openNewProcessRecord('before')"
      >
        新增服务前
      </el-button>
      <el-button
        v-if="canEditProcess && !canStart && order && order.status !== 'cancelled'"
        type="primary"
        plain
        @click="openProcessDialog('after')"
      >
        {{ afterRecords.length ? '编辑服务后' : '服务后纪录' }}
      </el-button>
      <el-button
        v-if="canEditProcess && !canStart && order && order.status !== 'cancelled' && afterRecords.length"
        plain
        @click="openNewProcessRecord('after')"
      >
        新增服务后
      </el-button>
      <el-button type="warning" plain @click="doRefreshReceipt">刷新工单票据</el-button>
      <el-button type="warning" plain @click="doRefreshReport">刷新服务报告</el-button>
      <el-button type="success" plain :loading="bundleLoading" @click="openDocBundle">票据+报告</el-button>
      <el-button v-if="canCancel" type="danger" plain @click="setStatus('cancelled')">取消</el-button>
      <el-button v-if="canDelete" type="danger" @click="remove">删除</el-button>
    </div>

    <el-alert
      class="flow-tip"
      type="info"
      :closable="false"
      show-icon
      :title="flowTip"
    />

    <el-row v-if="order" :gutter="16">
      <el-col :span="14">
        <el-card>
          <template #header>
            <div class="card-head">
              <span>服务工单 {{ order.orderNo }}</span>
              <div class="tags">
                <el-tag size="small">{{ modeLabel(order) }}</el-tag>
                <el-tag size="small" type="warning">{{ serviceStatusMap[order.status] || order.status }}</el-tag>
                <el-tag size="small" :type="order.payStatus === 'paid' ? 'success' : 'info'">
                  {{ servicePayStatusMap[order.payStatus || 'unpaid'] || '未付款' }}
                </el-tag>
              </div>
            </div>
          </template>

          <el-descriptions :column="2" border>
            <el-descriptions-item label="门店">{{ storeName }}</el-descriptions-item>
            <el-descriptions-item label="类型">{{ modeLabel(order) }}</el-descriptions-item>
            <el-descriptions-item label="客户">{{ order.customerName || '-' }}</el-descriptions-item>
            <el-descriptions-item label="电话">{{ order.customerPhone || '-' }}</el-descriptions-item>
            <el-descriptions-item label="预约时间">{{ formatDisplayTime(order.appointmentAt) }}</el-descriptions-item>
            <el-descriptions-item label="工程师">{{ order.engineerName || '-' }}</el-descriptions-item>
            <el-descriptions-item label="设备">{{ order.deviceInfo || '-' }}</el-descriptions-item>
            <el-descriptions-item label="预估费用">
              <strong class="amount">¥{{ Number(order.estimatedAmount || 0).toFixed(2) }}</strong>
            </el-descriptions-item>
            <el-descriptions-item label="说明" :span="2">{{ order.faultDesc || '-' }}</el-descriptions-item>
            <el-descriptions-item label="备注" :span="2">{{ order.remark || '-' }}</el-descriptions-item>
            <el-descriptions-item label="提醒">
              <template v-if="order.reminderEnabled">
                微信 · {{ reminderStatusMap[order.reminderStatus || 'pending'] }}
                · {{ formatDisplayTime(order.reminderAt) }}
              </template>
              <span v-else>-</span>
            </el-descriptions-item>
            <el-descriptions-item label="关联销售单">
              <el-button
                v-if="order.salesOrderId"
                link
                type="primary"
                @click="router.push(`/sales-orders/${order.salesOrderId}`)"
              >
                {{ order.salesOrderNo || `#${order.salesOrderId}` }}
              </el-button>
              <span v-else class="muted">无</span>
            </el-descriptions-item>
            <el-descriptions-item label="关联收银单">
              <el-button
                v-if="order.posOrderId"
                link
                type="primary"
                @click="router.push(`/pos/orders/${order.posOrderId}`)"
              >
                {{ order.posOrderNo || `#${order.posOrderId}` }}
              </el-button>
              <span v-else class="muted">尚未结算</span>
            </el-descriptions-item>
            <el-descriptions-item v-if="order.payStatus === 'paid'" label="付款方式">
              {{ paymentMethodLabel[order.paymentMethod || ''] || order.paymentMethod || '-' }}
            </el-descriptions-item>
            <el-descriptions-item v-if="order.payStatus === 'paid'" label="付款时间">
              {{ formatDisplayTime(order.paidAt) }}
            </el-descriptions-item>
            <el-descriptions-item v-if="order.paymentProofUrl" label="付款截图" :span="2">
              <el-image
                :src="order.paymentProofUrl"
                :preview-src-list="[order.paymentProofUrl]"
                preview-teleported
                fit="contain"
                class="proof-img"
              />
            </el-descriptions-item>
          </el-descriptions>

          <h4 class="section-title">服务项目</h4>
          <el-table :data="serviceItemsView" stripe>
            <el-table-column prop="serviceName" label="服务" min-width="140" />
            <el-table-column prop="serviceCode" label="编码" width="110" />
            <el-table-column label="原价" width="90">
              <template #default="{ row }">
                ¥{{ Number(row.originalPrice || row.unitPrice).toFixed(2) }}
              </template>
            </el-table-column>
            <el-table-column label="折扣" width="70">
              <template #default="{ row }">{{ row.discount ?? 10 }}</template>
            </el-table-column>
            <el-table-column label="单价" width="90">
              <template #default="{ row }">¥{{ Number(row.unitPrice).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column prop="quantity" label="数量" width="70" />
            <el-table-column label="小计" width="100">
              <template #default="{ row }">¥{{ Number(row.totalAmount).toFixed(2) }}</template>
            </el-table-column>
          </el-table>

          <h4 class="section-title">商品明细</h4>
          <el-table :data="productItemsView" stripe>
            <el-table-column prop="productName" label="商品" min-width="140" />
            <el-table-column prop="specLabel" label="规格" min-width="120" />
            <el-table-column prop="skuCode" label="SKU" width="110" />
            <el-table-column label="原价" width="90">
              <template #default="{ row }">
                ¥{{ Number(row.originalPrice || row.unitPrice).toFixed(2) }}
              </template>
            </el-table-column>
            <el-table-column label="折扣" width="70">
              <template #default="{ row }">{{ row.discount ?? 10 }}</template>
            </el-table-column>
            <el-table-column label="单价" width="90">
              <template #default="{ row }">¥{{ Number(row.unitPrice).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column prop="quantity" label="数量" width="70" />
            <el-table-column label="小计" width="100">
              <template #default="{ row }">¥{{ Number(row.totalAmount).toFixed(2) }}</template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!productItemsView.length" description="无商品" :image-size="48" />

          <div class="section-head">
            <h4 class="section-title">服务过程纪录</h4>
          </div>
          <el-alert
            v-if="canStart && beforeMediaCount < 1"
            type="warning"
            :closable="false"
            show-icon
            title="开始工单前须填写服务前过程纪录（至少一张图片或视频）"
            class="process-alert"
          />
          <el-alert
            v-else-if="canFinishService && afterMediaCount < 1"
            type="warning"
            :closable="false"
            show-icon
            title="完成服务前须填写服务后过程纪录（至少一张图片或视频）"
            class="process-alert"
          />

          <div class="process-phase">
            <div class="phase-label">
              <span>服务前 <el-tag size="small" type="info">{{ beforeMediaCount }} 个媒体</el-tag></span>
              <div v-if="canEditProcess" class="phase-actions">
                <el-button size="small" type="primary" plain @click="openProcessDialog('before')">
                  {{ beforeRecords.length ? '编辑上次' : '填写纪录' }}
                </el-button>
                <el-button v-if="beforeRecords.length" size="small" @click="openNewProcessRecord('before')">
                  新增纪录
                </el-button>
              </div>
            </div>
            <div v-if="beforeRecords.length" class="process-list">
              <div v-for="rec in beforeRecords" :key="rec.id" class="process-card">
                <div class="process-meta">
                  <span>{{ formatDisplayTime(rec.createdAt) }}</span>
                  <div v-if="canEditProcess" class="process-meta-actions">
                    <el-button link type="primary" @click="openEditProcessRecord(rec)">编辑</el-button>
                    <el-button link type="danger" @click="removeProcessRecord(rec)">删除</el-button>
                  </div>
                </div>
                <p v-if="rec.note" class="process-note">{{ rec.note }}</p>
                <div class="process-media">
                  <div v-for="(m, i) in rec.media || []" :key="m.url + i" class="pm-item">
                    <el-image
                      v-if="m.mediaType !== 'video'"
                      :src="m.url"
                      fit="cover"
                      class="pm-thumb"
                      :preview-src-list="(rec.media || []).filter(x => x.mediaType !== 'video').map(x => x.url)"
                    />
                    <a v-else :href="m.url" target="_blank" class="pm-thumb video">视频</a>
                  </div>
                </div>
              </div>
            </div>
            <el-empty v-else description="暂无服务前纪录" :image-size="40" />
          </div>

          <div class="process-phase">
            <div class="phase-label">
              <span>服务后 <el-tag size="small" type="success">{{ afterMediaCount }} 个媒体</el-tag></span>
              <div v-if="canEditProcess" class="phase-actions">
                <el-button size="small" type="primary" plain @click="openProcessDialog('after')">
                  {{ afterRecords.length ? '编辑上次' : '填写纪录' }}
                </el-button>
                <el-button v-if="afterRecords.length" size="small" @click="openNewProcessRecord('after')">
                  新增纪录
                </el-button>
              </div>
            </div>
            <div v-if="afterRecords.length" class="process-list">
              <div v-for="rec in afterRecords" :key="rec.id" class="process-card">
                <div class="process-meta">
                  <span>{{ formatDisplayTime(rec.createdAt) }}</span>
                  <div v-if="canEditProcess" class="process-meta-actions">
                    <el-button link type="primary" @click="openEditProcessRecord(rec)">编辑</el-button>
                    <el-button link type="danger" @click="removeProcessRecord(rec)">删除</el-button>
                  </div>
                </div>
                <p v-if="rec.note" class="process-note">{{ rec.note }}</p>
                <div class="process-media">
                  <div v-for="(m, i) in rec.media || []" :key="m.url + i" class="pm-item">
                    <el-image
                      v-if="m.mediaType !== 'video'"
                      :src="m.url"
                      fit="cover"
                      class="pm-thumb"
                      :preview-src-list="(rec.media || []).filter(x => x.mediaType !== 'video').map(x => x.url)"
                    />
                    <a v-else :href="m.url" target="_blank" class="pm-thumb video">视频</a>
                  </div>
                </div>
              </div>
            </div>
            <el-empty v-else description="暂无服务后纪录" :image-size="40" />
          </div>
        </el-card>
      </el-col>

      <el-col :span="10">
        <el-card class="receipt-card">
          <template #header>工单明细票据</template>
          <PosReceiptPanel
            v-if="order.receiptHtml"
            :html="order.receiptHtml"
            :order-no="order.orderNo"
            title="服务工单"
            variant="sales-doc"
            compact
          />
          <el-empty v-else description="点击「刷新工单票据」生成明细" />
        </el-card>
        <el-card class="receipt-card report-card">
          <template #header>服务报告</template>
          <PosReceiptPanel
            v-if="order.reportHtml"
            :html="order.reportHtml"
            :order-no="`${order.orderNo}-R`"
            title="服务报告"
            variant="sales-doc"
            compact
          />
          <el-empty v-else description="完成服务后自动生成；也可点「刷新服务报告」" />
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="editVisible" title="编辑服务工单" width="920px" destroy-on-close top="3vh">
      <el-form label-width="96px">
        <el-form-item label="工单类型">
          <el-radio-group v-model="form.orderMode">
            <el-radio-button v-for="o in serviceOrderModeOptions" :key="o.value" :value="o.value">
              {{ o.label }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.orderMode === 'appointment'" label="预约时间" required>
          <el-date-picker
            v-model="form.appointmentAt"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            format="YYYY-MM-DD HH:mm"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="服务项目">
          <div class="services-block">
            <div class="services-toolbar">
              <el-button type="primary" plain :icon="Plus" @click="openPicker">从服务目录添加</el-button>
              <span class="sum">
                <template v-if="originalEstimate > estimatedAmount + 0.001">
                  原价 ¥{{ originalEstimate.toFixed(2) }} /
                </template>
                预估 <strong>¥{{ estimatedAmount.toFixed(2) }}</strong>
              </span>
            </div>
            <el-table v-if="selected.length" :data="selected" size="small" border>
              <el-table-column prop="name" label="服务" min-width="120" />
              <el-table-column label="原价" width="88">
                <template #default="{ row }">¥{{ Number(row.originalPrice).toFixed(2) }}</template>
              </el-table-column>
              <el-table-column label="折扣" width="96">
                <template #default="{ row }">
                  <el-input-number
                    v-model="row.discount"
                    :min="0.1"
                    :max="10"
                    :step="0.5"
                    :precision="1"
                    size="small"
                    controls-position="right"
                    @change="onServiceDiscountChange(row)"
                  />
                </template>
              </el-table-column>
              <el-table-column label="单价" width="100">
                <template #default="{ row }">
                  <el-input-number
                    v-model="row.unitPrice"
                    :min="0"
                    :precision="2"
                    size="small"
                    controls-position="right"
                    @change="onServiceUnitPriceChange(row)"
                  />
                </template>
              </el-table-column>
              <el-table-column label="数量" width="100">
                <template #default="{ row }">
                  <el-input-number v-model="row.quantity" :min="1" :max="99" size="small" controls-position="right" />
                </template>
              </el-table-column>
              <el-table-column label="小计" width="90">
                <template #default="{ row }">¥{{ (row.unitPrice * row.quantity).toFixed(2) }}</template>
              </el-table-column>
              <el-table-column label="" width="50">
                <template #default="{ $index }">
                  <el-button link type="danger" :icon="Delete" @click="removeLine($index)" />
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-form-item>
        <el-form-item label="商品明细">
          <OrderLineEditor v-model="productLines" :store-id="order?.storeId" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="客户"><el-input v-model="form.customerName" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="电话"><el-input v-model="form.customerPhone" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="设备"><el-input v-model="form.deviceInfo" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="工程师"><el-input v-model="form.engineerName" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="说明"><el-input v-model="form.faultDesc" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="开启提醒"><el-switch v-model="form.reminderEnabled" /></el-form-item>
        <el-form-item v-if="form.reminderEnabled" label="提醒时间">
          <el-date-picker
            v-model="form.reminderAt"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            format="YYYY-MM-DD HH:mm"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="pickerVisible" title="选择服务" width="720px" append-to-body destroy-on-close>
      <div class="picker">
        <aside class="picker-cats">
          <div class="cat" :class="{ active: activeCategoryId === 0 }" @click="selectCategory(0)">全部</div>
          <div
            v-for="c in flatCategories"
            :key="c.id"
            class="cat"
            :class="{ active: activeCategoryId === c.id }"
            @click="selectCategory(c.id)"
          >
            {{ c.name }}
          </div>
        </aside>
        <div class="picker-main">
          <el-input v-model="keyword" clearable placeholder="搜索服务" class="picker-search" @keyup.enter="loadCatalog" />
          <el-table v-loading="catalogLoading" :data="catalogItems" height="360" size="small">
            <el-table-column prop="name" label="服务" min-width="140" />
            <el-table-column label="价格" width="90">
              <template #default="{ row }">¥{{ Number(row.price).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="" width="80">
              <template #default="{ row }">
                <el-button link type="primary" @click="addLine(row)">添加</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <template #footer>
        <el-button type="primary" @click="pickerVisible = false">完成</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="markPaidVisible"
      :title="order?.payStatus === 'paid' ? '修改付款信息' : '确认收款'"
      width="560px"
      destroy-on-close
    >
      <el-alert
        type="info"
        :closable="false"
        show-icon
        :title="order?.payStatus === 'paid'
          ? '可重新上传付款截图并识别/修改付款时间。'
          : '适用于微信/支付宝/银行等转账收款：上传付款截图后自动识别付款时间（优先收款时间）。'"
        class="mark-paid-alert"
      />
      <el-form label-width="96px">
        <el-form-item label="付款方式" required>
          <el-radio-group v-model="markPaidForm.paymentMethod">
            <el-radio-button value="transfer">转账</el-radio-button>
            <el-radio-button value="cash">现金</el-radio-button>
            <el-radio-button value="other">其他</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item
          :label="markPaidForm.paymentMethod === 'transfer' ? '付款截图' : '付款截图（选填）'"
          :required="markPaidForm.paymentMethod === 'transfer'"
        >
          <PaymentProofField
            v-model:proof-url="markPaidForm.paymentProofUrl"
            v-model:paid-at="markPaidForm.paidAt"
            subdir="payments/service"
            :require-proof="markPaidForm.paymentMethod === 'transfer'"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="markPaidVisible = false">取消</el-button>
        <el-button type="primary" :loading="markingPaid" @click="submitMarkPaid">
          {{ order?.payStatus === 'paid' ? '保存付款信息' : '确认已付款' }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="processVisible"
      :title="processDialogTitle"
      width="720px"
      destroy-on-close
      append-to-body
    >
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        class="process-alert"
        :title="processEditingId
          ? '正在编辑最近一条纪录，可改说明与媒体；若要另记一条请点「新增纪录」。'
          : (processPhase === 'before'
            ? '开始工单前须记录服务前状态，至少上传一张图片或视频，可附说明。'
            : '完成服务前须记录服务后变化，至少上传一张图片或视频，可附说明。')"
      />
      <el-form label-position="top" style="margin-top: 16px" class="process-form">
        <el-form-item label="说明">
          <el-input
            v-model="processNote"
            type="textarea"
            :rows="3"
            :placeholder="processPhase === 'before' ? '例如：故障现象、外观破损位置…' : '例如：维修结果、更换配件、测试情况…'"
          />
        </el-form-item>
        <el-form-item label="图片/视频（同一行展示，可左右滑动）" required>
          <MediaUploadField
            v-model="processMedia"
            :subdir="`service/process/${order?.id || 0}`"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button
          v-if="processEditingId && !processNextStatus"
          @click="openNewProcessRecord(processPhase)"
        >
          改为新增纪录
        </el-button>
        <el-button @click="processVisible = false">取消</el-button>
        <el-button type="primary" :loading="processSaving" @click="saveProcessRecord">
          {{ processNextStatus ? '保存并继续' : (processEditingId ? '保存修改' : '新增纪录') }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="bundleVisible"
      title="票据 + 服务报告"
      width="720px"
      destroy-on-close
      top="4vh"
    >
      <PosReceiptPanel
        v-if="bundleHtml"
        :html="bundleHtml"
        :order-no="order?.orderNo || ''"
        title="服务单据"
        variant="sales-doc"
      />
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.flow-tip { margin-bottom: 16px; }
.card-head { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.tags { display: flex; gap: 6px; }
.amount { color: #f56c6c; font-size: 16px; }
.section-title { margin: 20px 0 12px; font-size: 15px; }
.section-head {
  display: flex; justify-content: space-between; align-items: center; gap: 12px; margin-top: 8px;
}
.section-head .section-title { margin: 12px 0; }
.section-actions { display: flex; gap: 8px; }
.process-alert { margin: 8px 0 12px; }
.process-phase { margin-bottom: 16px; }
.phase-label {
  display: flex; align-items: center; justify-content: space-between; gap: 8px;
  font-weight: 600; margin-bottom: 8px; color: #303133; flex-wrap: wrap;
}
.phase-actions { display: flex; gap: 8px; font-weight: 400; }
.process-list { display: flex; flex-direction: column; gap: 10px; }
.process-card {
  border: 1px solid #ebeef5; border-radius: 8px; padding: 10px 12px; background: #fafafa;
}
.process-meta {
  display: flex; justify-content: space-between; align-items: center;
  font-size: 12px; color: #909399; margin-bottom: 6px;
}
.process-meta-actions { display: flex; gap: 4px; }
.process-note { margin: 0 0 8px; font-size: 13px; line-height: 1.5; white-space: pre-wrap; color: #303133; }
.process-media {
  display: flex; flex-direction: row; flex-wrap: nowrap; gap: 8px;
  overflow-x: auto; padding-bottom: 4px;
}
.pm-item {
  flex: 0 0 72px; width: 72px; height: 72px;
}
.pm-thumb {
  width: 72px !important; height: 72px !important; border-radius: 6px; border: 1px solid #e4e7ed;
  overflow: hidden; display: flex !important; align-items: center; justify-content: center;
  background: #fff; font-size: 12px; color: #409eff; text-decoration: none;
}
.pm-thumb :deep(.el-image__inner) { width: 72px; height: 72px; object-fit: cover; }
.pm-thumb.video { background: #ecf5ff; }
.process-form :deep(.el-form-item__content) {
  display: block !important;
  width: 100%;
  line-height: normal;
}
.process-form :deep(.media-field) { width: 100%; }
.muted { color: #c0c4cc; }
.proof-img { width: 160px; height: 160px; border-radius: 6px; border: 1px solid #ebeef5; }
.mark-paid-alert { margin-bottom: 16px; }
.receipt-card { margin-bottom: 16px; }
.report-card { position: sticky; top: 16px; }
.services-block { width: 100%; }
.services-toolbar {
  display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px;
}
.sum strong { color: #f56c6c; }
.picker { display: flex; gap: 12px; min-height: 400px; }
.picker-cats {
  width: 140px; flex-shrink: 0; border-right: 1px solid #ebeef5; padding-right: 8px; overflow-y: auto;
}
.cat { padding: 8px 10px; border-radius: 6px; cursor: pointer; font-size: 13px; color: #606266; }
.cat:hover { background: #f5f7fa; }
.cat.active { background: #ecf5ff; color: #409eff; font-weight: 600; }
.picker-main { flex: 1; min-width: 0; }
.picker-search { margin-bottom: 10px; }
</style>
