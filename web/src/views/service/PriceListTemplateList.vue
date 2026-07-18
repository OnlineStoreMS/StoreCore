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
  name: '默认服务价目表',
  receiptType: 'price_list',
  headerTitle: '服务价目表',
  headerSubtitle: '到店服务报价参考',
  headerExtra: '以下价格供顾客参考，具体以到店确认为准。',
  footerThanks: '价格如有变动以到店确认为准，欢迎咨询门店',
  footerExtra: '',
  showSkuPic: false,
  showStorePhone: true,
  showStoreAddress: true,
  showBusinessHours: true,
  showBrandLogo: true,
  showCoverPic: false,
  showGuideText: false,
  showMapLabel: false,
  showDescription: true,
  showDuration: true,
  isDefault: true,
  status: 1 as number,
})

const form = reactive(defaultForm())

async function load() {
  loading.value = true
  try {
    const data = await listReceiptTemplates(undefined, 1, 100, 'price_list')
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
    receiptType: 'price_list',
    headerTitle: row.headerTitle || '',
    headerSubtitle: row.headerSubtitle || '',
    headerExtra: row.headerExtra || '',
    footerThanks: row.footerThanks || '',
    footerExtra: row.footerExtra || '',
    showSkuPic: !!row.showSkuPic,
    showStorePhone: row.showStorePhone !== false,
    showStoreAddress: row.showStoreAddress !== false,
    showBusinessHours: row.showBusinessHours !== false,
    showBrandLogo: row.showBrandLogo !== false,
    showCoverPic: !!row.showCoverPic,
    showGuideText: !!row.showGuideText,
    showMapLabel: !!row.showMapLabel,
    showDescription: row.showDescription !== false,
    showDuration: row.showDuration !== false,
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
      receiptType: 'price_list',
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
      showDescription: form.showDescription,
      showDuration: form.showDuration,
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
        <h2>服务价目表模板</h2>
        <p class="desc">
          配置门店顾客可见的服务报价单：标题、门店信息（品牌 Logo、电话、地址、营业时间）、是否展示服务说明/时长/图片等。
          在「服务目录」勾选服务后即可按模板生成价目表预览与打印。
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
        <el-table-column prop="headerTitle" label="价目表标题" min-width="120" />
        <el-table-column prop="headerSubtitle" label="副标题" min-width="120" />
        <el-table-column label="显示项" min-width="260">
          <template #default="{ row }">
            <div class="flag-tags">
              <el-tag v-if="row.showBrandLogo !== false" size="small" type="success" effect="plain">品牌Logo</el-tag>
              <el-tag v-if="row.showStorePhone !== false" size="small" effect="plain">电话</el-tag>
              <el-tag v-if="row.showStoreAddress !== false" size="small" effect="plain">地址</el-tag>
              <el-tag v-if="row.showBusinessHours !== false" size="small" effect="plain">营业时间</el-tag>
              <el-tag v-if="row.showDescription !== false" size="small" effect="plain">服务说明</el-tag>
              <el-tag v-if="row.showDuration !== false" size="small" effect="plain">时长</el-tag>
              <el-tag v-if="row.showSkuPic" size="small" effect="plain">服务图</el-tag>
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
      :title="editingId ? '编辑价目表模板' : '新建价目表模板'"
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
        <el-form-item label="价目表标题">
          <el-input v-model="form.headerTitle" placeholder="如：服务价目表" />
        </el-form-item>
        <el-form-item label="副标题">
          <el-input v-model="form.headerSubtitle" placeholder="如：到店服务报价参考" />
        </el-form-item>
        <el-form-item label="页头说明">
          <el-input v-model="form.headerExtra" type="textarea" :rows="2" placeholder="显示在门店信息下方，可写活动说明等" />
        </el-form-item>
        <el-form-item label="页尾文案">
          <el-input v-model="form.footerThanks" type="textarea" :rows="2" placeholder="价格声明、咨询提示等" />
        </el-form-item>
        <el-form-item label="页尾补充">
          <el-input v-model="form.footerExtra" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="门店信息">
          <div class="switch-grid">
            <el-checkbox v-model="form.showBrandLogo">品牌 Logo</el-checkbox>
            <el-checkbox v-model="form.showStorePhone">门店电话</el-checkbox>
            <el-checkbox v-model="form.showStoreAddress">门店地址</el-checkbox>
            <el-checkbox v-model="form.showBusinessHours">营业时间</el-checkbox>
          </div>
          <div class="field-hint">品牌 Logo / 电话 / 地址 / 营业时间取自「门店档案」</div>
        </el-form-item>
        <el-form-item label="服务展示">
          <div class="switch-grid">
            <el-checkbox v-model="form.showDescription">服务说明</el-checkbox>
            <el-checkbox v-model="form.showDuration">参考时长</el-checkbox>
            <el-checkbox v-model="form.showSkuPic">服务图片</el-checkbox>
          </div>
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
