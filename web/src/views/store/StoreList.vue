<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createStore, deleteStore, listStores, updateStore, type Store } from '../../api/store'

const loading = ref(false)
const list = ref<Store[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const dialogVisible = ref(false)
const editing = ref<Store | null>(null)
const form = reactive({
  code: '',
  name: '',
  shortName: '',
  phone: '',
  address: '',
  businessHours: '',
  remark: '',
  status: 1,
})

async function load() {
  loading.value = true
  try {
    const data = await listStores(keyword.value, page.value, pageSize.value)
    list.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { code: '', name: '', shortName: '', phone: '', address: '', businessHours: '', remark: '', status: 1 })
  dialogVisible.value = true
}

function openEdit(row: Store) {
  editing.value = row
  Object.assign(form, row)
  dialogVisible.value = true
}

async function submit() {
  if (!form.code || !form.name) {
    ElMessage.warning('请填写门店编码和名称')
    return
  }
  if (editing.value) {
    await updateStore(editing.value.id, form)
    ElMessage.success('已更新')
  } else {
    await createStore(form)
    ElMessage.success('已创建')
  }
  dialogVisible.value = false
  await load()
}

async function remove(row: Store) {
  await ElMessageBox.confirm(`确定删除门店「${row.name}」？`, '确认')
  await deleteStore(row.id)
  ElMessage.success('已删除')
  await load()
}

onMounted(load)
</script>

<template>
  <el-card>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索门店" clearable style="width: 240px" @keyup.enter="load" />
      <el-button @click="load">查询</el-button>
      <el-button type="primary" @click="openCreate">新建门店</el-button>
    </div>
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="code" label="编码" width="120" />
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="phone" label="电话" width="140" />
      <el-table-column prop="address" label="地址" min-width="200" show-overflow-tooltip />
      <el-table-column prop="businessHours" label="营业时间" width="140" />
      <el-table-column label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="load"
      />
    </div>
  </el-card>

  <el-dialog v-model="dialogVisible" :title="editing ? '编辑门店' : '新建门店'" width="520px">
    <el-form label-width="90px">
      <el-form-item label="编码" required><el-input v-model="form.code" /></el-form-item>
      <el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="简称"><el-input v-model="form.shortName" /></el-form-item>
      <el-form-item label="电话"><el-input v-model="form.phone" /></el-form-item>
      <el-form-item label="地址"><el-input v-model="form.address" /></el-form-item>
      <el-form-item label="营业时间"><el-input v-model="form.businessHours" placeholder="如 09:00-21:00" /></el-form-item>
      <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; }
.pager { margin-top: 16px; display: flex; justify-content: flex-end; }
</style>
