<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useStores } from '../../composables/useStores'
import {
  createReceiptTemplate,
  deleteReceiptTemplate,
  listReceiptTemplates,
  updateReceiptTemplate,
  type ReceiptTemplate,
} from '../../api/receiptTemplate'

const { stores, reload: loadStores } = useStores()
const list = ref<ReceiptTemplate[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const editingId = ref<number>()
const saving = ref(false)

const defaultForm = () => ({
  storeId: 0 as number,
  name: '默认服务工单模板',
  receiptType: 'service',
  headerTitle: '服务工单明细',
  headerSubtitle: '正式单据',
  headerExtra: '',
  footerThanks: '客户签字确认：____________　　经办人：____________　　日期：____________',
  footerExtra: '以上金额仅供参考确认，服务完成后到店结算',
  showSkuPic: true,
  showStorePhone: true,
  showStoreAddress: true,
  showBusinessHours: true,
  showBrandLogo: true,
  showCoverPic: false,
  showGuideText: false,
  showMapLabel: false,
  isDefault: true,
  status: 1 as number,
})

const form = reactive(defaultForm())

async function load() {
  loading.value = true
  try {
    const data = await listReceiptTemplates(undefined, 1, 100, 'service')
    list.value = data.list
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = undefined
  Object.assign(form, defaultForm())
  dialogVisible.value = true
}

function openEdit(row: ReceiptTemplate) {
  editingId.value = row.id
  Object.assign(form, {
    storeId: row.storeId || 0,
    name: row.name,
    receiptType: 'service',
    headerTitle: row.headerTitle || '',
    headerSubtitle: row.headerSubtitle || '',
    headerExtra: row.headerExtra || '',
    footerThanks: row.footerThanks || '',
    footerExtra: row.footerExtra || '',
    showSkuPic: row.showSkuPic,
    showStorePhone: row.showStorePhone !== false,
    showStoreAddress: row.showStoreAddress !== false,
    showBusinessHours: row.showBusinessHours !== false,
    showBrandLogo: row.showBrandLogo !== false,
    showCoverPic: !!row.showCoverPic,
    showGuideText: !!row.showGuideText,
    showMapLabel: !!row.showMapLabel,
    isDefault: row.isDefault,
    status: row.status,
  })
  dialogVisible.value = true
}

async function save() {
  if (!form.name.trim()) {
    ElMessage.warning('请填写模板名称')
    return
  }
  saving.value = true
  try {
    const payload = {
      storeId: form.storeId || 0,
      name: form.name.trim(),
      receiptType: 'service',
      headerTitle: form.headerTitle,
      headerSubtitle: form.headerSubtitle,
      headerExtra: form.headerExtra,
      footerThanks: form.footerThanks,
      footerExtra: form.footerExtra,
      showSkuPic: form.showSkuPic,
      showStorePhone: form.showStorePhone,
      showStoreAddress: form.showStoreAddress,
      showBusinessHours: form.showBusinessHours,
      showBrandLogo: form.showBrandLogo,
      showCoverPic: form.showCoverPic,
      showGuideText: form.showGuideText,
      showMapLabel: form.showMapLabel,
      isDefault: form.isDefault,
      status: form.status,
    }
    if (editingId.value) {
      await updateReceiptTemplate(editingId.value, payload)
      ElMessage.success('已更新')
    } else {
      await createReceiptTemplate(payload)
      ElMessage.success('已创建')
    }
    dialogVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function remove(row: ReceiptTemplate) {
  await ElMessageBox.confirm(`确认删除模板「${row.name}」？`, '删除确认', { type: 'warning' })
  await deleteReceiptTemplate(row.id)
  ElMessage.success('已删除')
  await load()
}

function storeLabel(storeId: number) {
  if (!storeId) return '全部门店'
  return stores.value.find((s) => s.id === storeId)?.name || `#${storeId}`
}

onMounted(async () => {
  await loadStores()
  await load()
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h2>服务工单模板</h2>
        <p class="desc">
          配置服务工单票据的页头标题、副标题、页尾文案，以及是否显示品牌 Logo、门店电话、地址、营业时间、商品图等。品牌 Logo 取自门店档案。设为默认后，服务工单预览与下载将自动使用。
        </p>
      </div>
      <el-button type="primary" @click="openCreate">新建模板</el-button>
    </div>

    <el-card>
      <el-table v-loading="loading" :data="list" stripe style="width: 100%">
        <el-table-column prop="name" label="模板名称" min-width="140" />
        <el-table-column label="适用范围" min-width="120">
          <template #default="{ row }">{{ storeLabel(row.storeId) }}</template>
        </el-table-column>
        <el-table-column prop="headerTitle" label="单据标题" min-width="120" />
        <el-table-column prop="headerSubtitle" label="副标题/角标" min-width="120" />
        <el-table-column prop="footerThanks" label="页尾文案" min-width="180" show-overflow-tooltip />
        <el-table-column label="显示项" min-width="220">
          <template #default="{ row }">
            <div class="flag-tags">
              <el-tag v-if="row.showBrandLogo !== false" size="small" type="success" effect="plain">品牌Logo</el-tag>
              <el-tag v-if="row.showSkuPic" size="small" effect="plain">商品图</el-tag>
              <el-tag v-if="row.showStorePhone !== false" size="small" effect="plain">电话</el-tag>
              <el-tag v-if="row.showStoreAddress !== false" size="small" effect="plain">地址</el-tag>
              <el-tag v-if="row.showBusinessHours !== false" size="small" effect="plain">营业时间</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="默认" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.isDefault" type="warning" size="small">默认</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '编辑服务工单模板' : '新建服务工单模板'"
      width="680px"
      destroy-on-close
    >
      <el-form label-width="120px">
        <el-form-item label="模板名称" required>
          <el-input v-model="form.name" maxlength="64" />
        </el-form-item>
        <el-form-item label="适用门店">
          <el-select v-model="form.storeId" style="width: 100%">
            <el-option :value="0" label="全部门店" />
            <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="单据标题">
          <el-input v-model="form.headerTitle" placeholder="如：服务工单明细" />
        </el-form-item>
        <el-form-item label="副标题/角标">
          <el-input v-model="form.headerSubtitle" placeholder="如：正式单据" />
        </el-form-item>
        <el-form-item label="页头补充">
          <el-input v-model="form.headerExtra" type="textarea" :rows="2" placeholder="可选，显示在门店信息下方" />
        </el-form-item>
        <el-form-item label="正式页尾">
          <el-input v-model="form.footerThanks" type="textarea" :rows="2" placeholder="正式服务工单页尾（签字栏等）" />
        </el-form-item>
        <el-form-item label="预结算页尾">
          <el-input v-model="form.footerExtra" type="textarea" :rows="2" placeholder="预结算单页尾说明" />
        </el-form-item>
        <el-form-item label="显示开关">
          <div class="switch-grid">
            <el-checkbox v-model="form.showBrandLogo">品牌 Logo</el-checkbox>
            <el-checkbox v-model="form.showSkuPic">商品/SKU 图</el-checkbox>
            <el-checkbox v-model="form.showStorePhone">门店电话</el-checkbox>
            <el-checkbox v-model="form.showStoreAddress">门店地址</el-checkbox>
            <el-checkbox v-model="form.showBusinessHours">营业时间</el-checkbox>
          </div>
          <div class="field-hint">品牌 Logo 从门店档案自动带出，请先在「门店管理」上传</div>
        </el-form-item>
        <el-form-item label="设为默认">
          <el-switch v-model="form.isDefault" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
}
.page-header h2 { margin: 0; font-size: 20px; }
.desc { margin: 6px 0 0; color: #909399; font-size: 13px; max-width: 720px; line-height: 1.5; }
.flag-tags { display: flex; flex-wrap: wrap; gap: 4px; }
.switch-grid { display: flex; flex-wrap: wrap; gap: 12px 20px; }
.field-hint { width: 100%; margin-top: 8px; font-size: 12px; color: #909399; }
</style>
