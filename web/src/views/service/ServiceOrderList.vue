<script setup lang="ts">
import { onMounted, ref } from 'vue'
import client, { unwrap, type PageData } from '../../api/client'

interface ServiceOrder {
  id: number
  orderNo: string
  serviceType: string
  status: string
  customerName?: string
  deviceInfo?: string
}

const loading = ref(false)
const list = ref<ServiceOrder[]>([])

async function load() {
  loading.value = true
  try {
    const res = await client.get('/service-orders', { params: { page: 1, pageSize: 20 } })
    const data = unwrap<PageData<ServiceOrder>>(res)
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
      title="服务工单"
      description="中高端自行车维修、保养、预约服务等。可与收银台收款单关联（见 STORE_COLLECTION.md）。"
      type="info"
      show-icon
      :closable="false"
      class="mb-16"
    />
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="orderNo" label="工单号" width="200" />
      <el-table-column prop="serviceType" label="服务类型" width="120" />
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column prop="customerName" label="客户" width="120" />
      <el-table-column prop="deviceInfo" label="设备" min-width="160" />
    </el-table>
  </el-card>
</template>

<style scoped>
.mb-16 { margin-bottom: 16px; }
</style>
