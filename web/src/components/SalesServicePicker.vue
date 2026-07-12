<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Delete, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { listServiceItems, type ServiceItem } from '../api/serviceCatalog'
import type { SalesServiceLine } from '../api/salesOrder'

const lines = defineModel<SalesServiceLine[]>({ required: true })
const keyword = ref('')
const loading = ref(false)
const catalog = ref<ServiceItem[]>([])

async function search() {
  loading.value = true
  try {
    const data = await listServiceItems({ keyword: keyword.value.trim() || undefined, page: 1, pageSize: 50, status: 1 })
    catalog.value = data.list || []
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

function addItem(it: ServiceItem) {
  const existing = lines.value.find((l) => l.serviceItemId === it.id)
  if (existing) {
    existing.quantity += 1
    return
  }
  lines.value.push({
    serviceItemId: it.id,
    serviceName: it.name,
    serviceCode: it.code,
    quantity: 1,
    unitPrice: it.price || 0,
    durationMin: it.durationMin || 0,
    pic: it.pic,
  })
}

function removeLine(index: number) {
  lines.value.splice(index, 1)
}

onMounted(search)
</script>

<template>
  <div>
    <div class="toolbar">
      <el-input v-model="keyword" clearable placeholder="搜索服务" style="max-width: 280px" @keyup.enter="search">
        <template #append>
          <el-button :icon="Search" :loading="loading" @click="search" />
        </template>
      </el-input>
    </div>
    <div v-loading="loading" class="service-list">
      <el-button
        v-for="it in catalog"
        :key="it.id"
        size="small"
        :icon="Plus"
        @click="addItem(it)"
      >
        {{ it.name }} · ¥{{ (it.price || 0).toFixed(2) }}
      </el-button>
    </div>
    <el-table :data="lines" stripe class="mt-12">
      <el-table-column prop="serviceName" label="服务" min-width="160" />
      <el-table-column label="单价" width="110">
        <template #default="{ row }">¥{{ (row.unitPrice || 0).toFixed(2) }}</template>
      </el-table-column>
      <el-table-column label="数量" width="100">
        <template #default="{ row }">
          <el-input-number v-model="row.quantity" :min="1" size="small" />
        </template>
      </el-table-column>
      <el-table-column label="小计" width="90">
        <template #default="{ row }">¥{{ ((row.unitPrice || 0) * row.quantity).toFixed(2) }}</template>
      </el-table-column>
      <el-table-column width="56">
        <template #default="{ $index }">
          <el-button link type="danger" :icon="Delete" @click="removeLine($index)" />
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.toolbar { margin-bottom: 8px; }
.service-list { display: flex; flex-wrap: wrap; gap: 8px; min-height: 32px; }
.mt-12 { margin-top: 12px; }
</style>
