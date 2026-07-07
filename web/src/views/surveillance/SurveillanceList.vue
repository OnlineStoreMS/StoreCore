<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import client, { unwrap, type PageData } from '../../api/client'
import { useStores } from '../../composables/useStores'

interface Device {
  id: number
  name: string
  location?: string
  vendor?: string
  streamUrl?: string
  playbackUrl?: string
  status: number
}

const { stores, storeId } = useStores()
const loading = ref(false)
const list = ref<Device[]>([])
const dialogVisible = ref(false)
const form = reactive({
  name: '',
  location: '',
  vendor: '',
  streamUrl: '',
  playbackUrl: '',
  remark: '',
})

async function load() {
  loading.value = true
  try {
    const res = await client.get('/surveillance-devices', { params: { storeId: storeId.value, page: 1, pageSize: 50 } })
    list.value = unwrap<PageData<Device>>(res).list
  } finally {
    loading.value = false
  }
}

async function submit() {
  if (!storeId.value || !form.name) {
    ElMessage.warning('请填写设备名称')
    return
  }
  try {
    await client.post('/surveillance-devices', { storeId: storeId.value, ...form })
    ElMessage.success('已添加')
    dialogVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(load)
</script>

<template>
  <el-card>
    <div class="toolbar">
      <el-select v-model="storeId" style="width: 180px" @change="load">
        <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
      </el-select>
      <el-button type="primary" @click="dialogVisible = true">添加设备</el-button>
    </div>
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="name" label="设备" min-width="140" />
      <el-table-column prop="location" label="位置" width="120" />
      <el-table-column prop="vendor" label="厂商" width="100" />
      <el-table-column label="预览" width="100">
        <template #default="{ row }">
          <el-link v-if="row.streamUrl" :href="row.streamUrl" target="_blank" type="primary">打开</el-link>
        </template>
      </el-table-column>
      <el-table-column label="回放" width="100">
        <template #default="{ row }">
          <el-link v-if="row.playbackUrl" :href="row.playbackUrl" target="_blank">录像</el-link>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="dialogVisible" title="添加监控设备" width="520px">
    <el-form label-width="90px">
      <el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="位置"><el-input v-model="form.location" /></el-form-item>
      <el-form-item label="厂商"><el-input v-model="form.vendor" /></el-form-item>
      <el-form-item label="预览地址"><el-input v-model="form.streamUrl" placeholder="RTSP/HLS/平台链接" /></el-form-item>
      <el-form-item label="回放地址"><el-input v-model="form.playbackUrl" /></el-form-item>
      <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; }
</style>
