<script setup lang="ts">
import { onMounted, ref } from 'vue'
import client, { unwrap, type PageData } from '../../api/client'

interface Device {
  id: number
  name: string
  location?: string
  vendor?: string
  streamUrl?: string
  status: number
}

const loading = ref(false)
const list = ref<Device[]>([])

async function load() {
  loading.value = true
  try {
    const res = await client.get('/surveillance-devices', { params: { page: 1, pageSize: 20 } })
    const data = unwrap<PageData<Device>>(res)
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
      title="监控管理"
      description="对接门店室内外监控设备，支持实时预览与录像查阅。当前为设备档案与流地址预留，后续对接 NVR/云平台 SDK。"
      type="info"
      show-icon
      :closable="false"
      class="mb-16"
    />
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="name" label="设备名称" min-width="160" />
      <el-table-column prop="location" label="位置" width="140" />
      <el-table-column prop="vendor" label="厂商" width="120" />
      <el-table-column prop="streamUrl" label="预览地址" min-width="200" show-overflow-tooltip />
      <el-table-column label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '在线' : '停用' }}</el-tag>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<style scoped>
.mb-16 { margin-bottom: 16px; }
</style>
