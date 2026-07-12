<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listSalesOrders, type SalesOrder } from '../../api/salesOrder'
import {
  fulfillStatusMap,
  fulfillmentMap,
  purchaseStatusMap,
  salesStatusMap,
  useStores,
} from '../../composables/useStores'

const router = useRouter()
const { stores, storeId } = useStores()
const loading = ref(false)
const list = ref<SalesOrder[]>([])
const statusFilter = ref('')

async function load() {
  loading.value = true
  try {
    const data = await listSalesOrders({ storeId: storeId.value, status: statusFilter.value || undefined })
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
      <el-select v-model="storeId" placeholder="门店" style="width: 180px" @change="load">
        <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
      </el-select>
      <el-select v-model="statusFilter" clearable placeholder="订单状态" style="width: 140px" @change="load">
        <el-option v-for="(label, value) in salesStatusMap" :key="value" :label="label" :value="value" />
      </el-select>
      <el-button @click="load">刷新</el-button>
      <el-button type="primary" @click="router.push('/sales-orders/create')">新建销售订单</el-button>
    </div>
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="orderNo" label="单号" width="180" />
      <el-table-column label="履约" width="100">
        <template #default="{ row }">{{ fulfillmentMap[row.fulfillmentType] || row.fulfillmentType }}</template>
      </el-table-column>
      <el-table-column label="订单" width="90">
        <template #default="{ row }">{{ salesStatusMap[row.status] || row.status }}</template>
      </el-table-column>
      <el-table-column label="采购" width="100">
        <template #default="{ row }">{{ purchaseStatusMap[row.purchaseStatus || 'none'] }}</template>
      </el-table-column>
      <el-table-column label="履约进度" width="110">
        <template #default="{ row }">{{ fulfillStatusMap[row.fulfillStatus || 'none'] }}</template>
      </el-table-column>
      <el-table-column prop="customerName" label="顾客" width="100" />
      <el-table-column label="金额" width="100">
        <template #default="{ row }">¥{{ row.totalAmount?.toFixed(2) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="90" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/sales-orders/${row.id}`)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
</style>
