<script setup lang="ts">
import { onMounted, ref } from 'vue'
import client, { unwrap, type PageData } from '../../api/client'

interface PurchaseOrder {
  id: number
  poNo: string
  purchaseType: string
  supplierName?: string
  status: string
  totalAmount: number
}

const loading = ref(false)
const list = ref<PurchaseOrder[]>([])

async function load() {
  loading.value = true
  try {
    const res = await client.get('/purchase-orders', { params: { page: 1, pageSize: 20 } })
    const data = unwrap<PageData<PurchaseOrder>>(res)
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
      title="门店采购"
      description="销售订单驱动采购、门店备货采购等。供应商数据来自 SupplyCore，商品 SKU 来自 ProductCore。"
      type="info"
      show-icon
      :closable="false"
      class="mb-16"
    />
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="poNo" label="采购单号" width="200" />
      <el-table-column prop="purchaseType" label="类型" width="120" />
      <el-table-column prop="supplierName" label="供应商" width="160" />
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column prop="totalAmount" label="金额" width="100" />
    </el-table>
  </el-card>
</template>

<style scoped>
.mb-16 { margin-bottom: 16px; }
</style>
