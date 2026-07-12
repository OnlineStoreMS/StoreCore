<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deleteSalesOrder, listSalesOrders, type SalesOrder } from '../../api/salesOrder'
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

function canDelete(row: SalesOrder) {
  return ['draft', 'preview', 'cancelled'].includes(row.status)
}

async function remove(row: SalesOrder) {
  try {
    await ElMessageBox.confirm(`确认删除销售单「${row.orderNo}」？删除后不可恢复。`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
    await deleteSalesOrder(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message || '删除失败')
  }
}

onMounted(load)
</script>

<template>
  <el-card class="sales-list-card">
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
    <el-table v-loading="loading" :data="list" stripe style="width: 100%" table-layout="auto">
      <el-table-column prop="orderNo" label="单号" min-width="170" show-overflow-tooltip />
      <el-table-column label="履约方式" min-width="100">
        <template #default="{ row }">{{ fulfillmentMap[row.fulfillmentType] || row.fulfillmentType }}</template>
      </el-table-column>
      <el-table-column label="订单状态" min-width="90">
        <template #default="{ row }">
          <el-tag size="small" effect="plain">{{ salesStatusMap[row.status] || row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="采购状态" min-width="100">
        <template #default="{ row }">{{ purchaseStatusMap[row.purchaseStatus || 'none'] }}</template>
      </el-table-column>
      <el-table-column label="履约进度" min-width="100">
        <template #default="{ row }">{{ fulfillStatusMap[row.fulfillStatus || 'none'] }}</template>
      </el-table-column>
      <el-table-column prop="customerName" label="顾客" min-width="110" show-overflow-tooltip />
      <el-table-column prop="customerPhone" label="电话" min-width="120" show-overflow-tooltip />
      <el-table-column label="应付金额" min-width="110" align="right">
        <template #default="{ row }">¥{{ row.totalAmount?.toFixed(2) }}</template>
      </el-table-column>
      <el-table-column label="需采购" min-width="80" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.needProcurement" type="warning" size="small">是</el-tag>
          <span v-else class="muted">否</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right" align="center">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/sales-orders/${row.id}`)">详情</el-button>
          <el-button v-if="canDelete(row)" link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<style scoped>
.sales-list-card { width: 100%; }
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.muted { color: #c0c4cc; font-size: 13px; }
</style>
