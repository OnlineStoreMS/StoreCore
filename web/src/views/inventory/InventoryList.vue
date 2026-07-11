<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { listInventories, adjustInventory, type InventoryRow } from '../../api/inventory'
import PosSkuPicker from '../../components/PosSkuPicker.vue'
import type { ProductSkuSearchItem } from '../../api/productSku'
import { useStores } from '../../composables/useStores'

const router = useRouter()
const { stores, storeId } = useStores()
const loading = ref(false)
const list = ref<InventoryRow[]>([])
const keyword = ref('')
const dialogVisible = ref(false)
const adjustForm = reactive({
  skuId: 0,
  skuCode: '',
  productName: '',
  specLabel: '',
  quantity: 0,
  safetyStock: 0,
})

async function load() {
  loading.value = true
  try {
    const data = await listInventories(storeId.value, keyword.value)
    list.value = data.list
  } finally {
    loading.value = false
  }
}

function openAdjust(row?: InventoryRow) {
  if (row) {
    Object.assign(adjustForm, {
      skuId: row.skuId, skuCode: row.skuCode, productName: row.productName,
      specLabel: row.specLabel || '', quantity: row.quantity, safetyStock: row.safetyStock,
    })
  } else {
    Object.assign(adjustForm, {
      skuId: 0, skuCode: '', productName: '', specLabel: '', quantity: 0, safetyStock: 0,
    })
  }
  dialogVisible.value = true
}

function pickSku(sku: ProductSkuSearchItem) {
  adjustForm.skuId = sku.skuId
  adjustForm.skuCode = sku.skuCode
  adjustForm.productName = sku.productName
  adjustForm.specLabel = sku.specLabel || ''
}

async function submitAdjust() {
  if (!storeId.value || !adjustForm.skuId) {
    ElMessage.warning('请选择 SKU')
    return
  }
  try {
    await adjustInventory({ storeId: storeId.value, ...adjustForm })
    ElMessage.success('库存已更新')
    dialogVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(load)
</script>

<template>
  <el-card>
    <div class="toolbar">
      <el-select v-model="storeId" style="width: 180px" @change="load">
        <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
      </el-select>
      <el-input v-model="keyword" placeholder="搜索 SKU/商品" clearable style="width: 200px" @keyup.enter="load" />
      <el-button @click="load">查询</el-button>
      <el-button type="primary" @click="openAdjust()">盘点调整</el-button>
      <el-button type="warning" plain @click="router.push('/stock-transfers')">调货入库工单</el-button>
    </div>
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="skuCode" label="SKU" width="140" />
      <el-table-column prop="productName" label="商品" min-width="160" />
      <el-table-column prop="specLabel" label="规格" width="120" />
      <el-table-column prop="quantity" label="可用" width="80" />
      <el-table-column prop="safetyStock" label="安全库存" width="100" />
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button link type="primary" @click="openAdjust(row)">调整</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="dialogVisible" title="库存调整" width="520px">
    <PosSkuPicker v-if="!adjustForm.skuId" @select="pickSku" />
    <el-form v-else label-width="90px" class="mt-8">
      <el-form-item label="商品">{{ adjustForm.productName }} ({{ adjustForm.skuCode }})</el-form-item>
      <el-form-item label="当前数量"><el-input-number v-model="adjustForm.quantity" :min="0" /></el-form-item>
      <el-form-item label="安全库存"><el-input-number v-model="adjustForm.safetyStock" :min="0" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submitAdjust">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.mt-8 { margin-top: 8px; }
</style>
