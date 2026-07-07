<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import {
  getSalesOrder,
  confirmSalesOrder,
  cancelSalesOrder,
  markSalesReady,
  shipSalesOrder,
  completeSalesOrder,
  createPurchaseFromSales,
  type SalesOrder,
} from '../../api/salesOrder'
import { salesStatusMap } from '../../composables/useStores'

const route = useRoute()
const router = useRouter()
const id = Number(route.params.id)
const loading = ref(false)
const order = ref<SalesOrder | null>(null)

async function load() {
  loading.value = true
  try {
    order.value = await getSalesOrder(id)
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

async function createPO() {
  try {
    const po = await createPurchaseFromSales(id, { supplierName: '', items: [] })
    ElMessage.success('已生成采购单')
    router.push(`/purchase-orders/${(po as { id: number }).id}`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading">
    <el-page-header :icon="ArrowLeft" @back="router.push('/sales-orders')">
      <template #content>销售订单详情</template>
    </el-page-header>
    <el-card v-if="order" class="mt-16">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="单号">{{ order.orderNo }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ salesStatusMap[order.status] || order.status }}</el-descriptions-item>
        <el-descriptions-item label="顾客">{{ order.customerName }} {{ order.customerPhone }}</el-descriptions-item>
        <el-descriptions-item label="履约">{{ order.fulfillmentType }}</el-descriptions-item>
        <el-descriptions-item label="金额">¥{{ order.totalAmount?.toFixed(2) }}</el-descriptions-item>
        <el-descriptions-item label="需采购">{{ order.needProcurement ? '是' : '否' }}</el-descriptions-item>
        <el-descriptions-item v-if="order.shippingAddress" label="地址" :span="2">{{ order.shippingAddress }}</el-descriptions-item>
        <el-descriptions-item v-if="order.remark" label="备注" :span="2">{{ order.remark }}</el-descriptions-item>
      </el-descriptions>
      <el-table :data="order.items || []" stripe class="mt-16">
        <el-table-column prop="productName" label="商品" />
        <el-table-column prop="skuCode" label="SKU" width="120" />
        <el-table-column prop="quantity" label="数量" width="80" />
        <el-table-column label="单价" width="100">
          <template #default="{ row }">¥{{ row.unitPrice?.toFixed(2) }}</template>
        </el-table-column>
      </el-table>
      <div class="actions">
        <el-button v-if="order.status === 'draft'" @click="router.push(`/sales-orders/${id}/edit`)">编辑</el-button>
        <el-button v-if="order.status === 'draft'" type="primary" @click="act(() => confirmSalesOrder(id), '已确认')">确认订单</el-button>
        <el-button v-if="order.status === 'confirmed' && order.fulfillmentType === 'pickup'" type="success" @click="act(() => markSalesReady(id), '待提货')">标记待提货</el-button>
        <el-button v-if="order.status === 'confirmed' && order.fulfillmentType !== 'pickup'" type="success" @click="act(() => shipSalesOrder(id), '配送中')">发货</el-button>
        <el-button v-if="['ready','shipping'].includes(order.status)" type="primary" @click="act(() => completeSalesOrder(id), '已完成')">完成</el-button>
        <el-button v-if="order.needProcurement && order.status === 'confirmed'" type="warning" @click="createPO">生成采购单</el-button>
        <el-button v-if="!['completed','cancelled'].includes(order.status)" type="danger" @click="act(() => cancelSalesOrder(id), '已取消')">取消</el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.mt-16 { margin-top: 16px; }
.actions { margin-top: 16px; display: flex; flex-wrap: wrap; gap: 8px; }
</style>
