<script setup lang="ts">
import { onMounted, ref } from 'vue'
import client, { unwrap, type PageData } from '../../api/client'

interface SalesOrder {
  id: number
  orderNo: string
  fulfillmentType: string
  status: string
  totalAmount: number
  customerName?: string
}

const loading = ref(false)
const list = ref<SalesOrder[]>([])

async function load() {
  loading.value = true
  try {
    const res = await client.get('/sales-orders', { params: { page: 1, pageSize: 20 } })
    const data = unwrap<PageData<SalesOrder>>(res)
    list.value = data.list
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <el-card>
    <el-alert
      title="销售订单 — 非即时零售"
      description="支持手动创建线下订单：订货后提货、送货上门、发快递等。与收银台即时零售区分。"
      type="info"
      show-icon
      :closable="false"
      class="mb-16"
    />
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="orderNo" label="单号" width="200" />
      <el-table-column prop="fulfillmentType" label="履约方式" width="120" />
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column prop="customerName" label="顾客" width="120" />
      <el-table-column prop="totalAmount" label="金额" width="100" />
    </el-table>
  </el-card>
</template>

<style scoped>
.mb-16 { margin-bottom: 16px; }
</style>
