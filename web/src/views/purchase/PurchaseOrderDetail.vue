<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import {
  getPurchaseOrder,
  submitPurchaseOrder,
  receivePurchaseOrder,
  cancelPurchaseOrder,
  type PurchaseOrder,
} from '../../api/purchase'
import { purchaseStatusMap } from '../../composables/useStores'

const route = useRoute()
const router = useRouter()
const id = Number(route.params.id)
const loading = ref(false)
const order = ref<PurchaseOrder | null>(null)

async function load() {
  loading.value = true
  try {
    order.value = await getPurchaseOrder(id)
  } finally {
    loading.value = false
  }
}

async function act(fn: () => Promise<unknown>, msg: string) {
  try {
    await fn()
    ElMessage.success(msg)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading">
    <el-page-header :icon="ArrowLeft" @back="router.push('/purchase-orders')">
      <template #content>采购单详情</template>
    </el-page-header>
    <el-card v-if="order" class="mt-16">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="单号">{{ order.poNo }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ purchaseStatusMap[order.status] || order.status }}</el-descriptions-item>
        <el-descriptions-item label="供应商">{{ order.supplierName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ order.purchaseType }}</el-descriptions-item>
        <el-descriptions-item label="金额">¥{{ order.totalAmount?.toFixed(2) }}</el-descriptions-item>
      </el-descriptions>
      <el-table :data="order.items || []" stripe class="mt-16">
        <el-table-column prop="productName" label="商品" />
        <el-table-column prop="skuCode" label="SKU" width="120" />
        <el-table-column prop="quantity" label="数量" width="80" />
      </el-table>
      <div class="actions">
        <el-button v-if="order.status === 'draft'" type="primary" @click="act(() => submitPurchaseOrder(id), '已提交')">提交采购</el-button>
        <el-button v-if="order.status === 'submitted'" type="success" @click="act(() => receivePurchaseOrder(id), '已到货入库')">确认到货</el-button>
        <el-button v-if="['draft','submitted'].includes(order.status)" type="danger" @click="act(() => cancelPurchaseOrder(id), '已取消')">取消</el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.mt-16 { margin-top: 16px; }
.actions { margin-top: 16px; display: flex; gap: 8px; }
</style>
