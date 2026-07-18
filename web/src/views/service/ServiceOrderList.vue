<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Plus } from '@element-plus/icons-vue'
import {
  createServiceOrder,
  deleteServiceOrder,
  listServiceOrders,
  mergeServiceReceipt,
  type ServiceOrder,
} from '../../api/serviceOrder'
import {
  listServiceCategoryTree,
  listServiceItems,
  type ServiceCategory,
  type ServiceItem,
} from '../../api/serviceCatalog'
import OrderLineEditor, { type OrderLine } from '../../components/OrderLineEditor.vue'
import PosReceiptPanel from '../../components/PosReceiptPanel.vue'
import {
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

const { stores, storeId } = useStores()
const router = useRouter()
const loading = ref(false)
const list = ref<ServiceOrder[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const statusFilter = ref('')
const payStatusFilter = ref('')
const orderModeFilter = ref('')
const searchKeyword = ref('')
const selectedRows = ref<ServiceOrder[]>([])
const dialogVisible = ref(false)
const saving = ref(false)
const pickerVisible = ref(false)
const mergeVisible = ref(false)
const mergeHtml = ref('')
const mergeTotal = ref(0)
const mergeNos = ref<string[]>([])
const merging = ref(false)

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

const totalDuration = computed(() =>
  selected.value.reduce((sum, l) => sum + (l.durationMin || 0) * l.quantity, 0),
)

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

// service picker state
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

watch(
  () => form.orderMode,
  (mode) => {
    if (mode === 'instant') {
      form.appointmentAt = ''
      // 即时单提醒可选，不强制
    }
  },
)

watch(
  () => form.appointmentAt,
  (v) => {
    if (!form.reminderEnabled || !v || form.reminderAt) return
    // 默认提醒时间 = 预约前 30 分钟
    const d = new Date(v.replace(' ', 'T'))
    if (!Number.isNaN(d.getTime())) {
      d.setMinutes(d.getMinutes() - 30)
      form.reminderAt = formatDateTimeLocal(d)
    }
  },
)

function formatDateTimeLocal(d: Date) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:00`
}

function toApiTime(v: string) {
  if (!v) return undefined
  // el-date-picker value-format -> keep as local datetime string backend accepts
  return v.length === 16 ? `${v}:00` : v
}

function formatDisplayTime(v?: string) {
  if (!v) return '-'
  return v.replace('T', ' ').slice(0, 16)
}

async function load() {
  loading.value = true
  try {
    const data = await listServiceOrders({
      storeId: storeId.value,
      status: statusFilter.value || undefined,
      payStatus: payStatusFilter.value || undefined,
      orderMode: orderModeFilter.value || undefined,
      keyword: searchKeyword.value.trim() || undefined,
      page: page.value,
      pageSize,
    })
    list.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function resetPageAndLoad() {
  page.value = 1
  void load()
}

function openCreate() {
  Object.assign(form, {
    orderMode: 'appointment',
    customerName: '',
    customerPhone: '',
    deviceInfo: '',
    faultDesc: '',
    appointmentAt: '',
    engineerName: '',
    remark: '',
    reminderEnabled: true,
    reminderAt: '',
  })
  selected.value = []
  productLines.value = []
  dialogVisible.value = true
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

async function submit() {
  if (!storeId.value) {
    ElMessage.warning('请选择门店')
    return
  }
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
    const items = [
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
    ]
    const created = await createServiceOrder({
      storeId: storeId.value,
      orderMode: form.orderMode,
      customerName: form.customerName,
      customerPhone: form.customerPhone,
      deviceInfo: form.deviceInfo,
      faultDesc: form.faultDesc,
      appointmentAt: form.orderMode === 'appointment' ? toApiTime(form.appointmentAt) : undefined,
      engineerName: form.engineerName,
      remark: form.remark,
      items,
      reminderEnabled: form.reminderEnabled,
      reminderAt: form.reminderEnabled ? toApiTime(form.reminderAt) : undefined,
    })
    ElMessage.success('工单已创建')
    dialogVisible.value = false
    router.push(`/service-orders/${created.id}`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

function modeLabel(row: ServiceOrder) {
  return serviceOrderModeMap[row.orderMode] || serviceOrderModeMap[row.serviceType || ''] || row.orderMode || row.serviceType || '-'
}

function itemsSummary(row: ServiceOrder) {
  if (!row.items?.length) return ''
  return row.items.map((i) => {
    const name = i.itemType === 'product' ? (i.productName || '商品') : (i.serviceName || '服务')
    return `${name}×${i.quantity}`
  }).join('、')
}

function onSelectionChange(rows: ServiceOrder[]) {
  selectedRows.value = rows
}

async function removeRow(row: ServiceOrder) {
  const tips = [`确认删除服务工单「${row.orderNo}」？删除后不可恢复。`]
  if (row.posOrderId) {
    tips.push(`将同时删除关联收银订单 ${row.posOrderNo || '#' + row.posOrderId}。`)
  }
  if (row.salesOrderId) {
    tips.push('销售单上的服务工单关联将被清除（销售单本身保留）。')
  }
  try {
    await ElMessageBox.confirm(tips.join('\n'), '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
    })
    await deleteServiceOrder(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message || '删除失败')
  }
}

async function doMergePrint() {
  const rows = selectedRows.value
  if (rows.length < 2) {
    ElMessage.warning('请至少勾选两个服务工单')
    return
  }
  const store = rows[0].storeId
  const name = (rows[0].customerName || '').trim()
  const phone = (rows[0].customerPhone || '').trim()
  if (!name || !phone) {
    ElMessage.warning('合并打印要求客户姓名与电话均已填写')
    return
  }
  for (const r of rows) {
    if (r.storeId !== store) {
      ElMessage.warning('仅同门店工单可合并打印')
      return
    }
    if ((r.customerName || '').trim() !== name || (r.customerPhone || '').trim() !== phone) {
      ElMessage.warning('仅同客户姓名与电话的工单可合并打印')
      return
    }
  }
  merging.value = true
  try {
    const res = await mergeServiceReceipt(rows.map((r) => r.id))
    mergeHtml.value = res.html
    mergeTotal.value = res.totalAmount
    mergeNos.value = res.orderNos
    mergeVisible.value = true
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    merging.value = false
  }
}

onMounted(load)
</script>

<template>
  <el-card>
    <div class="toolbar">
      <el-select v-model="storeId" style="width: 160px" @change="resetPageAndLoad">
        <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
      </el-select>
      <el-select v-model="orderModeFilter" clearable placeholder="类型" style="width: 110px" @change="resetPageAndLoad">
        <el-option v-for="o in serviceOrderModeOptions" :key="o.value" :label="o.label" :value="o.value" />
      </el-select>
      <el-select v-model="statusFilter" clearable placeholder="工单状态" style="width: 120px" @change="resetPageAndLoad">
        <el-option v-for="(label, value) in serviceStatusMap" :key="value" :label="label" :value="value" />
      </el-select>
      <el-select v-model="payStatusFilter" clearable placeholder="付款状态" style="width: 110px" @change="resetPageAndLoad">
        <el-option v-for="(label, value) in servicePayStatusMap" :key="value" :label="label" :value="value" />
      </el-select>
      <el-input
        v-model="searchKeyword"
        clearable
        placeholder="单号/顾客/电话"
        style="width: 180px"
        @keyup.enter="resetPageAndLoad"
        @clear="resetPageAndLoad"
      />
      <el-button @click="resetPageAndLoad">查询</el-button>
      <el-button type="primary" @click="openCreate">新建服务工单</el-button>
      <el-button type="warning" plain :loading="merging" :disabled="selectedRows.length < 2" @click="doMergePrint">
        合并打印（{{ selectedRows.length }}）
      </el-button>
    </div>

    <el-table v-loading="loading" :data="list" stripe @selection-change="onSelectionChange">
      <el-table-column type="selection" width="48" />
      <el-table-column prop="orderNo" label="工单号" min-width="180" />
      <el-table-column label="类型" width="90">
        <template #default="{ row }">
          <el-tag :type="(row.orderMode || row.serviceType) === 'instant' ? 'warning' : 'primary'" size="small">
            {{ modeLabel(row) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">{{ serviceStatusMap[row.status] || row.status }}</template>
      </el-table-column>
      <el-table-column prop="customerName" label="客户" width="100" />
      <el-table-column label="预约时间" width="150">
        <template #default="{ row }">{{ formatDisplayTime(row.appointmentAt) }}</template>
      </el-table-column>
      <el-table-column label="服务项目" min-width="180">
        <template #default="{ row }">
          <span v-if="row.items?.length">{{ itemsSummary(row) }}</span>
          <span v-else class="muted">-</span>
        </template>
      </el-table-column>
      <el-table-column label="预估费用" width="110">
        <template #default="{ row }">¥{{ Number(row.estimatedAmount || 0).toFixed(2) }}</template>
      </el-table-column>
      <el-table-column label="付款" width="90">
        <template #default="{ row }">
          <el-tag :type="row.payStatus === 'paid' ? 'success' : 'warning'" size="small">
            {{ servicePayStatusMap[row.payStatus || 'unpaid'] || row.payStatus }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="关联销售单" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">
          <el-button
            v-if="row.salesOrderId"
            link
            type="primary"
            @click="router.push(`/sales-orders/${row.salesOrderId}`)"
          >
            {{ row.salesOrderNo || `#${row.salesOrderId}` }}
          </el-button>
          <span v-else class="muted">-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/service-orders/${row.id}`)">详情</el-button>
          <el-button link type="danger" @click="removeRow(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        background
        @current-change="load"
      />
    </div>
  </el-card>

  <el-dialog v-model="dialogVisible" title="新建服务工单" width="920px" destroy-on-close top="3vh">
    <el-form label-width="96px" class="order-form">
      <el-form-item label="工单类型" required>
        <el-radio-group v-model="form.orderMode">
          <el-radio-button v-for="o in serviceOrderModeOptions" :key="o.value" :value="o.value">
            {{ o.label }}
          </el-radio-button>
        </el-radio-group>
        <span class="field-hint">预约为主要场景；即时适合到店即做</span>
      </el-form-item>

      <el-form-item v-if="form.orderMode === 'appointment'" label="预约时间" required>
        <el-date-picker
          v-model="form.appointmentAt"
          type="datetime"
          value-format="YYYY-MM-DD HH:mm:ss"
          format="YYYY-MM-DD HH:mm"
          placeholder="选择预约到店时间"
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
              <template v-if="totalDuration"> · 约 {{ totalDuration }} 分钟</template>
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
          <el-empty v-else description="尚未选择服务" :image-size="64" />
        </div>
      </el-form-item>

      <el-form-item label="商品明细">
        <OrderLineEditor v-model="productLines" :store-id="storeId" />
      </el-form-item>

      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item label="客户"><el-input v-model="form.customerName" /></el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="电话"><el-input v-model="form.customerPhone" /></el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="设备"><el-input v-model="form.deviceInfo" placeholder="如车辆型号" /></el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="工程师"><el-input v-model="form.engineerName" /></el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="说明"><el-input v-model="form.faultDesc" type="textarea" :rows="2" placeholder="故障/需求说明" /></el-form-item>
      <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item>

      <el-divider content-position="left">提醒（预留）</el-divider>
      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="设计为微信消息提醒，当前仅保存提醒设置，暂不实际发送"
        class="reminder-alert"
      />
      <el-form-item label="开启提醒">
        <el-switch v-model="form.reminderEnabled" />
        <el-tag size="small" type="success" effect="plain" class="channel-tag">渠道：微信消息</el-tag>
      </el-form-item>
      <el-form-item v-if="form.reminderEnabled" label="提醒时间">
        <el-date-picker
          v-model="form.reminderAt"
          type="datetime"
          value-format="YYYY-MM-DD HH:mm:ss"
          format="YYYY-MM-DD HH:mm"
          placeholder="默认预约前 30 分钟"
          style="width: 100%"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="submit">创建工单</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="pickerVisible" title="选择服务" width="720px" append-to-body destroy-on-close>
    <div class="picker">
      <aside class="picker-cats">
        <div
          class="cat"
          :class="{ active: activeCategoryId === 0 }"
          @click="selectCategory(0)"
        >
          全部
        </div>
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
        <el-input
          v-model="keyword"
          clearable
          placeholder="搜索服务"
          class="picker-search"
          @keyup.enter="loadCatalog"
          @clear="loadCatalog"
        />
        <el-table v-loading="catalogLoading" :data="catalogItems" height="360" size="small">
          <el-table-column prop="name" label="服务" min-width="140" />
          <el-table-column prop="code" label="编码" width="100" />
          <el-table-column label="价格" width="90">
            <template #default="{ row }">¥{{ Number(row.price).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="时长" width="80">
            <template #default="{ row }">{{ row.durationMin ? `${row.durationMin}分` : '-' }}</template>
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

  <el-dialog v-model="mergeVisible" title="合并打印预览" width="960px" destroy-on-close top="3vh" class="merge-receipt-dialog">
    <div class="merge-meta">
      工单：{{ mergeNos.join('、') }} · 合计 <strong>¥{{ mergeTotal.toFixed(2) }}</strong>
    </div>
    <div class="merge-receipt-body">
      <PosReceiptPanel
        v-if="mergeHtml"
        :html="mergeHtml"
        :order-no="mergeNos.join('-')"
        title="合并服务工单"
        variant="sales-doc"
      />
    </div>
  </el-dialog>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
.muted { color: #c0c4cc; }
.field-hint { margin-left: 12px; color: #909399; font-size: 12px; }
.services-block { width: 100%; }
.services-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  gap: 12px;
}
.sum { font-size: 13px; color: #606266; }
.sum strong { color: #f56c6c; font-size: 16px; }
.reminder-alert { margin-bottom: 14px; }
.channel-tag { margin-left: 12px; }
.picker {
  display: flex;
  gap: 12px;
  min-height: 400px;
}
.picker-cats {
  width: 140px;
  flex-shrink: 0;
  border-right: 1px solid #ebeef5;
  padding-right: 8px;
  overflow-y: auto;
}
.cat {
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: #606266;
}
.cat:hover { background: #f5f7fa; }
.cat.active { background: #ecf5ff; color: #409eff; font-weight: 600; }
.picker-main { flex: 1; min-width: 0; }
.picker-search { margin-bottom: 10px; }
.merge-meta { margin-bottom: 12px; color: #606266; font-size: 13px; }
.merge-meta strong { color: #f56c6c; font-size: 16px; }
.merge-receipt-body {
  max-height: calc(100vh - 180px);
  overflow: auto;
  padding-right: 4px;
}
</style>
