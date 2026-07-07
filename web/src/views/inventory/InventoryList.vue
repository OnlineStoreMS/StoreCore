<script setup lang="ts">
import { onMounted, ref } from 'vue'
import client, { unwrap, type PageData } from '../../api/client'

interface InventoryRow {
  id: number
  skuCode: string
  productName: string
  specLabel?: string
  quantity: number
  safetyStock: number
}

const loading = ref(false)
const list = ref<InventoryRow[]>([])

async function load() {
  loading.value = true
  try {
    const res = await client.get('/inventories', { params: { page: 1, pageSize: 20 } })
    const data = unwrap<PageData<InventoryRow>>(res)
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
      title="门店库存"
      description="门店级库存是 OSMS 平台库存的子集，SKU 引用 ProductCore 中央底库。平台级 IMS/WMS 开发后可双向同步。"
      type="info"
      show-icon
      :closable="false"
      class="mb-16"
    />
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="skuCode" label="SKU" width="140" />
      <el-table-column prop="productName" label="商品" min-width="160" />
      <el-table-column prop="specLabel" label="规格" width="120" />
      <el-table-column prop="quantity" label="可用" width="80" />
      <el-table-column prop="safetyStock" label="安全库存" width="100" />
    </el-table>
  </el-card>
</template>

<style scoped>
.mb-16 { margin-bottom: 16px; }
</style>
