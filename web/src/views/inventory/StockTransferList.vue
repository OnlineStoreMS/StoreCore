<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Picture, Plus } from '@element-plus/icons-vue'
import {
  cancelStockTransfer,
  confirmStockTransfer,
  createStockTransfer,
  listStockTransfers,
  type StockTransferOrder,
} from '../../api/stockTransfer'
import { displaySpecValues, resolvePic } from '../../api/catalog'
import PosProductCatalog from '../../components/PosProductCatalog.vue'
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
  pic?: string
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
    specLabel: displaySpecValues(sku.specLabel, sku.specs),
    pic: resolvePic(sku.pic, sku.productPic),
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
  if (!v) return ''
  return v.replace('T', ' ').slice(0, 16)
}

function itemsSkuSummary(row: StockTransferOrder) {
  if (!row.items?.length) return '-'
  return row.items.map((i) => `${i.skuCode || i.skuId}×${i.quantity}`).join('、')
}

function statusTagType(status: string) {
  if (status === 'received') return 'success'
  if (status === 'cancelled') return 'info'
  return 'warning'
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

    <el-table v-loading="loading" :data="list" stripe row-key="id">
      <el-table-column type="expand">
        <template #default="{ row }">
          <el-table :data="row.items || []" size="small" border class="inner-table">
            <el-table-column label="规格" min-width="120">
              <template #default="{ row: item }">{{ displaySpecValues(item.specLabel) }}</template>
            </el-table-column>
            <el-table-column label="图片" width="72" align="center">
              <template #default="{ row: item }">
                <el-image
                  v-if="resolvePic(item.pic)"
                  :src="resolvePic(item.pic)"
                  :preview-src-list="[resolvePic(item.pic)]"
                  fit="cover"
                  class="line-thumb"
                  preview-teleported
                />
                <div v-else class="line-thumb empty"><el-icon><Picture /></el-icon></div>
              </template>
            </el-table-column>
            <el-table-column prop="skuCode" label="SKU" width="140" />
            <el-table-column prop="productName" label="商品" min-width="140" />
            <el-table-column prop="quantity" label="数量" width="80" />
          </el-table>
        </template>
      </el-table-column>
      <el-table-column prop="orderNo" label="单号" min-width="180" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.status)" size="small" effect="plain">
            {{ statusMap[row.status] || row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="期望入库" width="150">
        <template #default="{ row }">{{ formatTime(row.expectedAt) || '-' }}</template>
      </el-table-column>
      <el-table-column label="入库时间" width="150">
        <template #default="{ row }">
          <span v-if="row.status === 'received'">{{ formatTime(row.receivedAt) || '-' }}</span>
          <span v-else class="muted" />
        </template>
      </el-table-column>
      <el-table-column label="明细(SKU)" min-width="220">
        <template #default="{ row }">{{ itemsSkuSummary(row) }}</template>
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

  <el-dialog
    v-model="dialogVisible"
    title="创建仓库调货入库工单"
    width="960px"
    top="4vh"
    destroy-on-close
    class="transfer-dialog"
  >
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
          <div class="picker-tip">按分类浏览，或搜索商品/SKU；先选商品再选规格，可预览图片</div>
          <PosProductCatalog :store-id="storeId" :require-store-stock="false" @select="pickSku" />
          <el-table v-if="lines.length" :data="lines" size="small" border class="mt">
            <el-table-column label="规格" min-width="120">
              <template #default="{ row }">{{ displaySpecValues(row.specLabel) }}</template>
            </el-table-column>
            <el-table-column label="图片" width="72" align="center">
              <template #default="{ row }">
                <el-image
                  v-if="resolvePic(row.pic)"
                  :src="resolvePic(row.pic)"
                  :preview-src-list="[resolvePic(row.pic)]"
                  fit="cover"
                  class="line-thumb"
                  preview-teleported
                />
                <div v-else class="line-thumb empty"><el-icon><Picture /></el-icon></div>
              </template>
            </el-table-column>
            <el-table-column prop="skuCode" label="SKU" width="130" />
            <el-table-column prop="productName" label="商品" min-width="140" />
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
.picker-tip { color: #909399; font-size: 12px; margin-bottom: 8px; }
.mt { margin-top: 12px; }
.ml { margin-left: 10px; }
.inner-table { margin: 8px 16px 12px; }
.line-thumb {
  width: 40px; height: 40px; border-radius: 4px;
}
.line-thumb.empty {
  display: inline-flex; align-items: center; justify-content: center;
  background: #f5f7fa; color: #c0c4cc;
}
</style>
