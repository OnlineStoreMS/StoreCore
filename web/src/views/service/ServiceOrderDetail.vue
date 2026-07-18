<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Plus } from '@element-plus/icons-vue'
import {
  deleteServiceOrder,
  getServiceOrder,
  markServicePaid,
  refreshServiceReceipt,
  updateServiceOrder,
  updateServiceStatus,
  type ServiceOrder,
  type ServiceOrderItem,
} from '../../api/serviceOrder'
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

const canEdit = computed(() => order.value?.status === 'pending')
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
/** 服务已做完（待付款）且已收款时，可单独确认工单履约完成 */
const canCompleteWork = computed(() =>
  order.value?.status === 'awaiting_payment' && order.value.payStatus === 'paid',
)
const canCancel = computed(() =>
  !!order.value && ['pending', 'in_progress', 'awaiting_payment'].includes(order.value.status),
)
const canDelete = computed(() => !!order.value)
const flowTip = computed(() => {
  if (order.value?.salesOrderId) {
    return '关联销售单：销售单付款后服务单记为已付款；完成服务后若已付款将直接完成工单。零元服务同样无需收银。'
  }
  return '付款与工单完成相互独立：可先收款（上传转账截图/收银台），服务做完后再点完成服务；也可先完成服务再收款。'
})

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
    .map((it: ServiceOrderItem) => ({
      serviceItemId: it.serviceItemId || 0,
      name: it.serviceName || '',
      code: it.serviceCode,
      unitPrice: it.unitPrice,
      durationMin: it.durationMin,
      quantity: it.quantity,
    }))
  productLines.value = (order.value.items || [])
    .filter((it: ServiceOrderItem) => it.itemType === 'product' || (!!it.skuId && !it.serviceItemId))
    .map((it: ServiceOrderItem) => ({
      skuId: it.skuId || 0,
      productName: it.productName || '',
      skuCode: it.skuCode,
      specLabel: it.specLabel,
      pic: it.pic,
      quantity: it.quantity,
      originalPrice: it.unitPrice,
      discount: 10,
      unitPrice: it.unitPrice,
    }))
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
        })),
        ...productLines.value.map((l) => ({
          itemType: 'product' as const,
          skuId: l.skuId,
          productName: l.productName,
          skuCode: l.skuCode,
          specLabel: l.specLabel,
          pic: l.pic,
          quantity: l.quantity,
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

async function setStatus(status: string) {
  if (!order.value) return
  const labels: Record<string, string> = {
    in_progress: '确认开始工单？状态将变为进行中',
    awaiting_payment: skipCashier.value
      ? '确认服务已完成？已收款，将标记工单为已完成'
      : '确认服务已完成？状态将变为待付款，可再收款或去收银台结算',
    completed: '确认将工单标记为已完成？',
    cancelled: '确认取消该工单？',
  }
  await ElMessageBox.confirm(labels[status] || '确认操作？', '确认', { type: 'warning' })
  try {
    await updateServiceStatus(order.value.id, status)
    ElMessage.success('状态已更新')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
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

function openMarkPaid() {
  markPaidForm.paymentMethod = 'transfer'
  markPaidForm.paymentProofUrl = ''
  markPaidForm.paidAt = ''
  markPaidVisible.value = true
}

async function submitMarkPaid() {
  if (!order.value) return
  if (markPaidForm.paymentMethod === 'transfer' && !markPaidForm.paymentProofUrl) {
    ElMessage.warning('转账收款请先上传付款截图')
    return
  }
  markingPaid.value = true
  try {
    order.value = await markServicePaid(order.value.id, {
      paymentMethod: markPaidForm.paymentMethod,
      paymentProofUrl: markPaidForm.paymentProofUrl || undefined,
      paidAt: paidAtToApi(markPaidForm.paidAt),
    })
    markPaidVisible.value = false
    ElMessage.success('已确认收款')
  } catch (e) {
    ElMessage.error((e as Error).message || '确认收款失败')
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
      <el-button v-if="canSettle" type="success" @click="goSettle">去收银台结算</el-button>
      <el-button v-if="canMarkPaid" type="success" plain @click="openMarkPaid">确认收款（上传截图）</el-button>
      <el-button type="warning" plain @click="doRefreshReceipt">刷新工单票据</el-button>
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
            <el-table-column label="单价" width="90">
              <template #default="{ row }">¥{{ Number(row.unitPrice).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column prop="quantity" label="数量" width="70" />
            <el-table-column label="小计" width="100">
              <template #default="{ row }">¥{{ Number(row.totalAmount).toFixed(2) }}</template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!productItemsView.length" description="无商品" :image-size="48" />
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
              <span class="sum">预估 <strong>¥{{ estimatedAmount.toFixed(2) }}</strong></span>
            </div>
            <el-table v-if="selected.length" :data="selected" size="small" border>
              <el-table-column prop="name" label="服务" min-width="140" />
              <el-table-column label="单价" width="90">
                <template #default="{ row }">¥{{ Number(row.unitPrice).toFixed(2) }}</template>
              </el-table-column>
              <el-table-column label="数量" width="120">
                <template #default="{ row }">
                  <el-input-number v-model="row.quantity" :min="1" :max="99" size="small" controls-position="right" />
                </template>
              </el-table-column>
              <el-table-column label="" width="60">
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

    <el-dialog v-model="markPaidVisible" title="确认收款" width="560px" destroy-on-close>
      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="适用于微信/支付宝/银行等转账收款：上传付款截图后自动识别付款时间（优先收款时间）。"
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
        <el-button type="primary" :loading="markingPaid" @click="submitMarkPaid">确认已付款</el-button>
      </template>
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
.muted { color: #c0c4cc; }
.proof-img { width: 160px; height: 160px; border-radius: 6px; border: 1px solid #ebeef5; }
.mark-paid-alert { margin-bottom: 16px; }
.receipt-card { position: sticky; top: 16px; }
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
