<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listServiceOrders, createServiceOrder, updateServiceStatus, type ServiceOrder } from '../../api/serviceOrder'
import { serviceStatusMap, serviceTypeOptions, useStores } from '../../composables/useStores'

const { stores, storeId } = useStores()
const loading = ref(false)
const list = ref<ServiceOrder[]>([])
const dialogVisible = ref(false)
const saving = ref(false)
const form = reactive({
  serviceType: 'repair',
  customerName: '',
  customerPhone: '',
  deviceInfo: '',
  faultDesc: '',
  engineerName: '',
  estimatedAmount: 0,
  remark: '',
})

async function load() {
  loading.value = true
  try {
    const data = await listServiceOrders(storeId.value)
    list.value = data.list
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, {
    serviceType: 'repair', customerName: '', customerPhone: '',
    deviceInfo: '', faultDesc: '', engineerName: '', estimatedAmount: 0, remark: '',
  })
  dialogVisible.value = true
}

async function submit() {
  if (!storeId.value) {
    ElMessage.warning('请选择门店')
    return
  }
  saving.value = true
  try {
    await createServiceOrder({ storeId: storeId.value, ...form })
    ElMessage.success('已创建')
    dialogVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function setStatus(row: ServiceOrder, status: string) {
  try {
    await updateServiceStatus(row.id, status)
    ElMessage.success('状态已更新')
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
      <el-button type="primary" @click="openCreate">新建服务工单</el-button>
    </div>
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="orderNo" label="工单号" width="200" />
      <el-table-column prop="serviceType" label="类型" width="100" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">{{ serviceStatusMap[row.status] || row.status }}</template>
      </el-table-column>
      <el-table-column prop="customerName" label="客户" width="100" />
      <el-table-column prop="deviceInfo" label="设备" min-width="140" />
      <el-table-column prop="engineerName" label="工程师" width="100" />
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button v-if="row.status === 'pending'" link type="primary" @click="setStatus(row, 'in_progress')">开始</el-button>
          <el-button v-if="row.status === 'in_progress'" link type="success" @click="setStatus(row, 'completed')">完成</el-button>
          <el-button v-if="['pending','in_progress'].includes(row.status)" link type="danger" @click="setStatus(row, 'cancelled')">取消</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="dialogVisible" title="新建服务工单" width="560px">
    <el-form label-width="90px">
      <el-form-item label="服务类型">
        <el-select v-model="form.serviceType" style="width: 100%">
          <el-option v-for="o in serviceTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="客户"><el-input v-model="form.customerName" /></el-form-item>
      <el-form-item label="电话"><el-input v-model="form.customerPhone" /></el-form-item>
      <el-form-item label="设备"><el-input v-model="form.deviceInfo" placeholder="如 捷安特 TCR 2024" /></el-form-item>
      <el-form-item label="故障描述"><el-input v-model="form.faultDesc" type="textarea" /></el-form-item>
      <el-form-item label="工程师"><el-input v-model="form.engineerName" /></el-form-item>
      <el-form-item label="预估费用"><el-input-number v-model="form.estimatedAmount" :min="0" :precision="2" /></el-form-item>
      <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="submit">创建</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; }
</style>
