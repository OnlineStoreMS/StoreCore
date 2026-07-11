<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Plus } from '@element-plus/icons-vue'
import {
  cancelStockTransfer,
  confirmStockTransfer,
  createStockTransfer,
  listStockTransfers,
  type StockTransferOrder,
} from '../../api/stockTransfer'
import PosSkuPicker from '../../components/PosSkuPicker.vue'
import type { ProductSkuSearchItem } from '../../api/productSku'
import { reminderStatusMap, useStores } from '../../composables/useStores'

const statusMap: Record<string, string> = {
  pending: '待入库',
  received: '已入库',
  cancelled: '已取消',
}

interface Line {
  skuId: number
  skuCode?: string
  productName: string
  specLabel?: string
  quantity: number
}

const { stores, storeId } = useStores()
const loading = ref(false)
const list = ref<StockTransferOrder[]>([])
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive({
  expectedAt: '' as string,
  remark: '',
  reminderEnabled: true,
  reminderAt: '' as string,
})
const lines = ref<Line[]>([])

async function load() {
  loading.value = true
  try {
    const data = await listStockTransfers(storeId.value, 1, 50)
    list.value = data.list
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { expectedAt: '', remark: '', reminderEnabled: true, reminderAt: '' })
  lines.value = []
  dialogVisible.value = true
}

function pickSku(sku: ProductSkuSearchItem) {
  const existing = lines.value.find((l) => l.skuId === sku.skuId)
  if (existing) {
    existing.quantity += 1
    return
  }
  lines.value.push({
    skuId: sku.skuId,
    skuCode: sku.skuCode,
    productName: sku.productName,
    specLabel: sku.specLabel,
    quantity: 1,
  })
}

function removeLine(index: number) {
  lines.value.splice(index, 1)
}

function toApiTime(v: string) {
  if (!v) return undefined
  return v.length === 16 ? `${v}:00` : v
}

async function submit() {
  if (!storeId.value) {
    ElMessage.warning('请选择门店')
    return
  }
  if (lines.value.length === 0) {
    ElMessage.warning('请添加调货商品')
    return
  }
  saving.value = true
  try {
    await createStockTransfer({
      storeId: storeId.value,
      expectedAt: toApiTime(form.expectedAt),
      remark: form.remark,
      items: lines.value,
      reminderEnabled: form.reminderEnabled,
      reminderAt: form.reminderEnabled ? toApiTime(form.reminderAt || form.expectedAt) : undefined,
    })
    ElMessage.success('调货入库工单已创建')
    dialogVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function confirm(row: StockTransferOrder) {
  await ElMessageBox.confirm(
    `确认「${row.orderNo}」已入库？将增加门店库存。仓库侧库存扣减暂不生效（库存系统未就绪）。`,
    '确认入库',
    { type: 'warning' },
  )
  await confirmStockTransfer(row.id)
  ElMessage.success('已确认入库，门店库存已更新')
  await load()
}

async function cancel(row: StockTransferOrder) {
  await ElMessageBox.confirm(`取消调货单「${row.orderNo}」？`, '取消确认', { type: 'warning' })
  await cancelStockTransfer(row.id)
  ElMessage.success('已取消')
  await load()
}

function formatTime(v?: string) {
  if (!v) return '-'
  return v.replace('T', ' ').slice(0, 16)
}

function itemsSummary(row: StockTransferOrder) {
  if (!row.items?.length) return '-'
  return row.items.map((i) => `${i.productName}×${i.quantity}`).join('、')
}

onMounted(load)
</script>

<template>
  <el-card>
    <div class="toolbar">
      <el-select v-model="storeId" style="width: 180px" @change="load">
        <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
      </el-select>
      <el-button @click="load">刷新</el-button>
      <el-button type="primary" @click="openCreate">创建调货入库</el-button>
    </div>

    <el-alert
      class="tip"
      type="info"
      :closable="false"
      show-icon
      title="调货入库工单用于记录「需从仓库调货到门店」的商品，防止遗忘。确认入库后更新门店库存；仓库中央库存扣减待库存系统就绪后再对接。提醒功能预留。"
    />

    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="orderNo" label="单号" min-width="180" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">{{ statusMap[row.status] || row.status }}</template>
      </el-table-column>
      <el-table-column label="期望入库" width="150">
        <template #default="{ row }">{{ formatTime(row.expectedAt) }}</template>
      </el-table-column>
      <el-table-column label="明细" min-width="200">
        <template #default="{ row }">{{ itemsSummary(row) }}</template>
      </el-table-column>
      <el-table-column label="提醒" width="100">
        <template #default="{ row }">
          <el-tag v-if="row.reminderEnabled" size="small" type="info" effect="plain">
            {{ reminderStatusMap[row.reminderStatus || 'pending'] || '待发送' }}
          </el-tag>
          <span v-else class="muted">-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button v-if="row.status === 'pending'" link type="success" @click="confirm(row)">确认入库</el-button>
          <el-button v-if="row.status === 'pending'" link type="danger" @click="cancel(row)">取消</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="dialogVisible" title="创建仓库调货入库工单" width="720px" destroy-on-close>
    <el-form label-width="100px">
      <el-form-item label="期望入库">
        <el-date-picker
          v-model="form.expectedAt"
          type="datetime"
          value-format="YYYY-MM-DD HH:mm:ss"
          format="YYYY-MM-DD HH:mm"
          placeholder="选择期望到店/入库时间"
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="调货商品" required>
        <div class="lines">
          <PosSkuPicker @select="pickSku" />
          <el-table v-if="lines.length" :data="lines" size="small" border class="mt">
            <el-table-column prop="productName" label="商品" min-width="140" />
            <el-table-column prop="specLabel" label="规格" width="120" />
            <el-table-column label="数量" width="120">
              <template #default="{ row }">
                <el-input-number v-model="row.quantity" :min="1" size="small" />
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
      <el-form-item label="备注">
        <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="如：收银台缺货需调货" />
      </el-form-item>
      <el-form-item label="开启提醒">
        <el-switch v-model="form.reminderEnabled" />
        <el-tag size="small" type="success" effect="plain" class="ml">渠道预留：微信/平台消息</el-tag>
      </el-form-item>
      <el-form-item v-if="form.reminderEnabled" label="提醒时间">
        <el-date-picker
          v-model="form.reminderAt"
          type="datetime"
          value-format="YYYY-MM-DD HH:mm:ss"
          format="YYYY-MM-DD HH:mm"
          placeholder="默认用期望入库时间"
          style="width: 100%"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" :icon="Plus" @click="submit">创建</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.tip { margin-bottom: 14px; }
.muted { color: #c0c4cc; }
.lines { width: 100%; }
.mt { margin-top: 12px; }
.ml { margin-left: 10px; }
</style>
