<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listPurchaseOrders, type PurchaseOrder } from '../../api/purchase'
import { purchaseStatusMap, useStores } from '../../composables/useStores'

const router = useRouter()
const { stores, storeId } = useStores()
const loading = ref(false)
const list = ref<PurchaseOrder[]>([])

async function load() {
  loading.value = true
  try {
    const data = await listPurchaseOrders(storeId.value)
    list.value = data.list
  } finally {
    loading.value = false
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
      <el-button type="primary" @click="router.push('/purchase-orders/create')">新建采购单</el-button>
    </div>
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="poNo" label="采购单号" width="200" />
      <el-table-column prop="purchaseType" label="类型" width="120" />
      <el-table-column prop="supplierName" label="供应商" width="160" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">{{ purchaseStatusMap[row.status] || row.status }}</template>
      </el-table-column>
      <el-table-column label="金额" width="100">
        <template #default="{ row }">¥{{ row.totalAmount?.toFixed(2) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/purchase-orders/${row.id}`)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; }
</style>
