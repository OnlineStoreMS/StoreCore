<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deleteSalesOrder, listSalesOrders, type SalesOrder } from '../../api/salesOrder'
import {
  fulfillStatusMap,
  fulfillmentMap,
  purchaseStatusMap,
  salesPayStatusMap,
  salesStatusMap,
  useStores,
} from '../../composables/useStores'

const router = useRouter()
const { stores, storeId } = useStores()
const loading = ref(false)
const list = ref<SalesOrder[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const statusFilter = ref('')
const payStatusFilter = ref('')
const fulfillmentFilter = ref('')
const purchaseStatusFilter = ref('')
const keyword = ref('')

const purchaseFilterOptions = [
  { value: 'none', label: '无需采购' },
  { value: 'pending', label: '待采购' },
  { value: 'ordered', label: '已下采购单' },
  { value: 'received', label: '已到货' },
]

function resetPageAndLoad() {
  page.value = 1
  void load()
}

async function load() {
  loading.value = true
  try {
    const data = await listSalesOrders({
      storeId: storeId.value,
      status: statusFilter.value || undefined,
      payStatus: payStatusFilter.value || undefined,
      fulfillmentType: fulfillmentFilter.value || undefined,
      purchaseStatus: purchaseStatusFilter.value || undefined,
      keyword: keyword.value.trim() || undefined,
      page: page.value,
      pageSize,
    })
    list.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function canDelete(row: SalesOrder) {
  return ['draft', 'preview', 'cancelled'].includes(row.status)
}

async function remove(row: SalesOrder) {
  try {
    const tips = [`确认删除销售单「${row.orderNo}」？删除后不可恢复。`]
    if (row.serviceOrderId) {
      tips.push(`将同时删除关联服务工单 ${row.serviceOrderNo || '#' + row.serviceOrderId}，及其关联收银订单（如有）。`)
    }
    await ElMessageBox.confirm(tips.join('\n'), '删除确认', {
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
      <el-select v-model="storeId" placeholder="门店" style="width: 160px" @change="resetPageAndLoad">
        <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
      </el-select>
      <el-select v-model="statusFilter" clearable placeholder="订单状态" style="width: 120px" @change="resetPageAndLoad">
        <el-option v-for="(label, value) in salesStatusMap" :key="value" :label="label" :value="value" />
      </el-select>
      <el-select v-model="fulfillmentFilter" clearable placeholder="履约方式" style="width: 120px" @change="resetPageAndLoad">
        <el-option v-for="(label, value) in fulfillmentMap" :key="value" :label="label" :value="value" />
      </el-select>
      <el-select v-model="payStatusFilter" clearable placeholder="付款状态" style="width: 110px" @change="resetPageAndLoad">
        <el-option v-for="(label, value) in salesPayStatusMap" :key="value" :label="label" :value="value" />
      </el-select>
      <el-select v-model="purchaseStatusFilter" clearable placeholder="采购状态" style="width: 120px" @change="resetPageAndLoad">
        <el-option v-for="o in purchaseFilterOptions" :key="o.value" :label="o.label" :value="o.value" />
      </el-select>
      <el-input
        v-model="keyword"
        clearable
        placeholder="单号/顾客/电话"
        style="width: 180px"
        @keyup.enter="resetPageAndLoad"
        @clear="resetPageAndLoad"
      />
      <el-button @click="resetPageAndLoad">查询</el-button>
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
      <el-table-column label="付款" min-width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.payStatus === 'paid' ? 'success' : 'warning'" size="small">
            {{ salesPayStatusMap[row.payStatus] || row.payStatus || '未付款' }}
          </el-tag>
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
      <el-table-column label="补货" min-width="100" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.needProcurement" type="warning" size="small">需采购</el-tag>
          <el-tag v-else-if="row.stockTransferOrderId" type="info" size="small">调货</el-tag>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right" align="center">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/sales-orders/${row.id}`)">详情</el-button>
          <el-button v-if="canDelete(row)" link type="danger" @click="remove(row)">删除</el-button>
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
</template>

<style scoped>
.sales-list-card { width: 100%; }
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
.muted { color: #c0c4cc; font-size: 13px; }
</style>
