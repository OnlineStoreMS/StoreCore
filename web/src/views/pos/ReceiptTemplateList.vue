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

const form = reactive({
  storeId: 0 as number,
  name: '',
  receiptType: 'small',
  headerTitle: '门店收银小票',
  headerSubtitle: '欢迎光临',
  headerExtra: '',
  footerThanks: '谢谢惠顾，欢迎再次光临',
  footerExtra: '商品如有质量问题，请凭小票在7日内联系门店处理',
  showSkuPic: true,
  isDefault: true,
  status: 1 as number,
})

async function load() {
  loading.value = true
  try {
    const data = await listReceiptTemplates(undefined, 1, 100)
    list.value = data.list
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = undefined
  Object.assign(form, {
    storeId: 0,
    name: '默认小票',
    receiptType: 'small',
    headerTitle: '门店收银小票',
    headerSubtitle: '欢迎光临',
    headerExtra: '',
    footerThanks: '谢谢惠顾，欢迎再次光临',
    footerExtra: '商品如有质量问题，请凭小票在7日内联系门店处理',
    showSkuPic: true,
    isDefault: true,
    status: 1,
  })
  dialogVisible.value = true
}

function openEdit(row: ReceiptTemplate) {
  editingId.value = row.id
  Object.assign(form, {
    storeId: row.storeId || 0,
    name: row.name,
    receiptType: row.receiptType || 'small',
    headerTitle: row.headerTitle || '',
    headerSubtitle: row.headerSubtitle || '',
    headerExtra: row.headerExtra || '',
    footerThanks: row.footerThanks || '',
    footerExtra: row.footerExtra || '',
    showSkuPic: row.showSkuPic,
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
      receiptType: form.receiptType,
      headerTitle: form.headerTitle,
      headerSubtitle: form.headerSubtitle,
      headerExtra: form.headerExtra,
      footerThanks: form.footerThanks,
      footerExtra: form.footerExtra,
      showSkuPic: form.showSkuPic,
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
        <h2>小票模板</h2>
        <p class="desc">配置电子小票页头、页尾与是否展示 SKU 缩略图。设为默认后，收银台结算将自动使用。</p>
      </div>
      <el-button type="primary" @click="openCreate">新建模板</el-button>
    </div>

    <el-card>
      <el-table v-loading="loading" :data="list" stripe>
        <el-table-column prop="name" label="模板名称" min-width="140" />
        <el-table-column label="适用范围" width="140">
          <template #default="{ row }">{{ storeLabel(row.storeId) }}</template>
        </el-table-column>
        <el-table-column prop="headerTitle" label="页头标题" min-width="140" />
        <el-table-column prop="footerThanks" label="页尾致谢" min-width="160" show-overflow-tooltip />
        <el-table-column label="SKU 图" width="90">
          <template #default="{ row }">
            <el-tag :type="row.showSkuPic ? 'success' : 'info'" size="small">
              {{ row.showSkuPic ? '显示' : '隐藏' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="默认" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.isDefault" type="warning" size="small">默认</el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '编辑小票模板' : '新建小票模板'"
      width="640px"
      destroy-on-close
    >
      <el-form label-width="100px">
        <el-form-item label="模板名称" required>
          <el-input v-model="form.name" maxlength="64" />
        </el-form-item>
        <el-form-item label="适用门店">
          <el-select v-model="form.storeId" style="width: 100%">
            <el-option :value="0" label="全部门店" />
            <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="小票类型">
          <el-radio-group v-model="form.receiptType">
            <el-radio value="small">小票</el-radio>
            <el-radio value="large">大票</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-divider content-position="left">页头</el-divider>
        <el-form-item label="页头标题">
          <el-input v-model="form.headerTitle" placeholder="如：门店收银小票 / 品牌名" />
        </el-form-item>
        <el-form-item label="页头副标题">
          <el-input v-model="form.headerSubtitle" placeholder="如：欢迎光临" />
        </el-form-item>
        <el-form-item label="页头附加">
          <el-input
            v-model="form.headerExtra"
            type="textarea"
            :rows="2"
            placeholder="可选，支持多行，如营业时间、促销语"
          />
        </el-form-item>
        <el-divider content-position="left">页尾</el-divider>
        <el-form-item label="致谢语">
          <el-input v-model="form.footerThanks" placeholder="如：谢谢惠顾" />
        </el-form-item>
        <el-form-item label="页尾附加">
          <el-input
            v-model="form.footerExtra"
            type="textarea"
            :rows="3"
            placeholder="可选，支持多行，如退换货说明、公众号引导"
          />
        </el-form-item>
        <el-divider />
        <el-form-item label="SKU 缩略图">
          <el-switch v-model="form.showSkuPic" active-text="显示" inactive-text="隐藏" />
        </el-form-item>
        <el-form-item label="设为默认">
          <el-switch v-model="form.isDefault" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">停用</el-radio>
          </el-radio-group>
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
  margin-bottom: 16px;
}
.page-header h2 {
  margin: 0 0 6px;
  font-size: 20px;
}
.desc {
  margin: 0;
  color: #909399;
  font-size: 13px;
}
</style>
