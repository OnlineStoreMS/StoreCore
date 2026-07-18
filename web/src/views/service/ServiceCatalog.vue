<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Edit, Plus, Document } from '@element-plus/icons-vue'
import type { TableInstance } from 'element-plus'
import { formatDurationMin } from '../../utils/formatDuration'
import {
  createServiceCategory,
  createServiceItem,
  deleteServiceCategory,
  deleteServiceItem,
  listServiceCategoryTree,
  listServiceItems,
  previewServicePriceList,
  updateServiceCategory,
  updateServiceItem,
  type ServiceCategory,
  type ServiceItem,
} from '../../api/serviceCatalog'
import { listReceiptTemplates, type ReceiptTemplate } from '../../api/receiptTemplate'
import { useStores } from '../../composables/useStores'
import PosReceiptPanel from '../../components/PosReceiptPanel.vue'

const { stores, storeId, reload: loadStores } = useStores()
const catLoading = ref(false)
const itemLoading = ref(false)
const categories = ref<ServiceCategory[]>([])
const items = ref<ServiceItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const statusFilter = ref<number | ''>('')
const activeCategoryId = ref(0)
/** 跨分类/分页记住勾选（id → 服务） */
const selectedMap = ref(new Map<number, ServiceItem>())
const selectedCount = computed(() => selectedMap.value.size)
const tableRef = ref<TableInstance>()
let restoringSelection = false
const priceListVisible = ref(false)
const priceListLoading = ref(false)
const priceListHtml = ref('')
const priceListTemplates = ref<ReceiptTemplate[]>([])
const priceListForm = reactive({
  storeId: 0 as number,
  templateId: 0 as number,
})

const catDialog = ref(false)
const itemDialog = ref(false)
const editingCatId = ref<number>()
const editingItemId = ref<number>()
const saving = ref(false)

const catForm = reactive({
  parentId: 0,
  name: '',
  sort: 0,
  status: 1,
})

const itemForm = reactive({
  categoryId: 0,
  code: '',
  name: '',
  description: '',
  price: 0,
  durationMin: 0,
  sort: 0,
  status: 1,
})

const flatCategories = computed(() => {
  const out: { id: number; name: string; level: number }[] = []
  function walk(list: ServiceCategory[], level: number) {
    for (const c of list) {
      out.push({ id: c.id, name: c.name, level })
      if (c.children?.length) walk(c.children, level + 1)
    }
  }
  walk(categories.value, 0)
  return out
})

const categoryTotal = computed(() => flatCategories.value.length)

const activeCategoryName = computed(() => {
  if (!activeCategoryId.value) return '全部服务'
  return findCat(categories.value, activeCategoryId.value)?.name || '全部服务'
})

const enabledCount = computed(() => items.value.filter((i) => i.status === 1).length)
const disabledCount = computed(() => items.value.filter((i) => i.status !== 1).length)

function findCat(list: ServiceCategory[], id: number): ServiceCategory | undefined {
  for (const c of list) {
    if (c.id === id) return c
    if (c.children?.length) {
      const hit = findCat(c.children, id)
      if (hit) return hit
    }
  }
  return undefined
}

async function loadCategories() {
  catLoading.value = true
  try {
    categories.value = await listServiceCategoryTree()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载分类失败')
  } finally {
    catLoading.value = false
  }
}

async function loadItems(reset = true) {
  if (reset) page.value = 1
  itemLoading.value = true
  try {
    const data = await listServiceItems({
      categoryId: activeCategoryId.value || undefined,
      keyword: keyword.value.trim() || undefined,
      status: statusFilter.value === '' ? undefined : Number(statusFilter.value),
      page: page.value,
      pageSize: pageSize.value,
    })
    items.value = data.list
    total.value = data.total
    await restoreTableSelection()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载服务失败')
  } finally {
    itemLoading.value = false
  }
}

async function restoreTableSelection() {
  await nextTick()
  if (!tableRef.value) return
  restoringSelection = true
  tableRef.value.clearSelection()
  for (const row of items.value) {
    if (selectedMap.value.has(row.id)) {
      tableRef.value.toggleRowSelection(row, true)
    }
  }
  await nextTick()
  restoringSelection = false
}

function selectCategory(id: number) {
  activeCategoryId.value = id
  void loadItems(true)
}

function onTreeNodeClick(data: ServiceCategory) {
  selectCategory(data.id)
}

function openCreateCat(parentId = 0) {
  editingCatId.value = undefined
  Object.assign(catForm, { parentId, name: '', sort: 0, status: 1 })
  catDialog.value = true
}

function openEditCat(data: ServiceCategory) {
  editingCatId.value = data.id
  Object.assign(catForm, {
    parentId: data.parentId,
    name: data.name,
    sort: data.sort,
    status: data.status,
  })
  catDialog.value = true
}

async function saveCat() {
  if (!catForm.name.trim()) {
    ElMessage.warning('请填写分类名称')
    return
  }
  saving.value = true
  try {
    if (editingCatId.value) {
      await updateServiceCategory(editingCatId.value, { ...catForm })
      ElMessage.success('分类已更新')
    } else {
      await createServiceCategory({ ...catForm })
      ElMessage.success('分类已创建')
    }
    catDialog.value = false
    await loadCategories()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function removeCat(data: ServiceCategory) {
  try {
    await ElMessageBox.confirm(
      `确定删除分类「${data.name}」？需先清空子分类与服务项目。`,
      '删除确认',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    )
    await deleteServiceCategory(data.id)
    ElMessage.success('分类已删除')
    if (activeCategoryId.value === data.id) activeCategoryId.value = 0
    await loadCategories()
    await loadItems(true)
  } catch (e) {
    if (e === 'cancel' || (e as { action?: string })?.action === 'cancel') return
    ElMessage.error((e as Error).message || '删除失败，请先清空子分类和服务项目')
  }
}

function openCreateItem() {
  editingItemId.value = undefined
  Object.assign(itemForm, {
    categoryId: activeCategoryId.value || flatCategories.value[0]?.id || 0,
    code: '',
    name: '',
    description: '',
    price: 0,
    durationMin: 0,
    sort: 0,
    status: 1,
  })
  itemDialog.value = true
}

function openEditItem(row: ServiceItem) {
  editingItemId.value = row.id
  Object.assign(itemForm, {
    categoryId: row.categoryId,
    code: row.code || '',
    name: row.name,
    description: row.description || '',
    price: row.price,
    durationMin: row.durationMin || 0,
    sort: row.sort,
    status: row.status,
  })
  itemDialog.value = true
}

async function saveItem() {
  if (!itemForm.categoryId || !itemForm.name.trim()) {
    ElMessage.warning('请选择分类并填写服务名称')
    return
  }
  saving.value = true
  try {
    if (editingItemId.value) {
      await updateServiceItem(editingItemId.value, { ...itemForm })
      ElMessage.success('服务已更新')
    } else {
      await createServiceItem({ ...itemForm })
      ElMessage.success('服务已创建')
    }
    itemDialog.value = false
    await loadCategories()
    await loadItems(false)
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function removeItem(row: ServiceItem) {
  try {
    await ElMessageBox.confirm(`确定删除服务「${row.name}」？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
    await deleteServiceItem(row.id)
    ElMessage.success('服务已删除')
    await loadCategories()
    await loadItems(false)
  } catch (e) {
    if (e === 'cancel' || (e as { action?: string })?.action === 'cancel') return
    ElMessage.error((e as Error).message || '删除失败')
  }
}

function onSelectionChange(rows: ServiceItem[]) {
  if (restoringSelection) return
  const pageIds = new Set(items.value.map((i) => i.id))
  // 同步本页：取消勾选的从汇总里移除
  for (const id of [...selectedMap.value.keys()]) {
    if (pageIds.has(id) && !rows.some((r) => r.id === id)) {
      selectedMap.value.delete(id)
    }
  }
  // 本页新勾选写入汇总（换页/换分类后仍保留）
  const next = new Map(selectedMap.value)
  for (const row of rows) {
    next.set(row.id, row)
  }
  selectedMap.value = next
}

function clearPriceListSelection() {
  selectedMap.value = new Map()
  tableRef.value?.clearSelection()
}

async function openPriceListDialog() {
  if (!selectedCount.value) {
    ElMessage.warning('请先勾选要生成价目表的服务（可跨分类、跨页勾选）')
    return
  }
  if (!stores.value.length) await loadStores()
  if (!priceListForm.storeId) {
    priceListForm.storeId = storeId.value || stores.value[0]?.id || 0
  }
  try {
    const data = await listReceiptTemplates(priceListForm.storeId || undefined, 1, 50, 'price_list')
    priceListTemplates.value = data.list || []
    const def = priceListTemplates.value.find((t) => t.isDefault)
    priceListForm.templateId = def?.id || priceListTemplates.value[0]?.id || 0
  } catch {
    priceListTemplates.value = []
    priceListForm.templateId = 0
  }
  priceListHtml.value = ''
  priceListVisible.value = true
}

async function generatePriceList() {
  if (!priceListForm.storeId) {
    ElMessage.warning('请选择门店')
    return
  }
  if (!selectedCount.value) {
    ElMessage.warning('请勾选服务项目')
    return
  }
  priceListLoading.value = true
  try {
    const res = await previewServicePriceList({
      storeId: priceListForm.storeId,
      templateId: priceListForm.templateId || undefined,
      serviceItemIds: [...selectedMap.value.keys()],
      groupByCategory: true,
    })
    priceListHtml.value = res.html
    ElMessage.success(`已生成价目表（${res.itemCount} 项）`)
  } catch (e) {
    ElMessage.error((e as Error).message || '生成失败')
  } finally {
    priceListLoading.value = false
  }
}

onMounted(async () => {
  await loadStores()
  await loadCategories()
  await loadItems(true)
})
</script>

<template>
  <div class="catalog-page">
    <el-row :gutter="16">
      <el-col :span="8">
        <el-card v-loading="catLoading" class="cat-card">
          <template #header>
            <div class="card-head">
              <span>服务分类</span>
              <el-button type="primary" :icon="Plus" size="small" @click="openCreateCat(0)">
                添加一级分类
              </el-button>
            </div>
          </template>

          <div class="cat-stats">
            共 <strong>{{ categoryTotal }}</strong> 个分类
          </div>

          <div
            class="all-row"
            :class="{ active: activeCategoryId === 0 }"
            @click="selectCategory(0)"
          >
            <span>全部服务</span>
            <el-tag size="small" type="info">全部</el-tag>
          </div>

          <el-tree
            :data="categories"
            :props="{ label: 'name', children: 'children' }"
            default-expand-all
            node-key="id"
            highlight-current
            :current-node-key="activeCategoryId || undefined"
            @node-click="onTreeNodeClick"
          >
            <template #default="{ node, data }">
              <div class="tree-node">
                <span class="tree-label">{{ node.label }}</span>
                <span class="node-meta" @click.stop>
                  <el-tag size="small" type="info">{{ data.itemCount || 0 }} 项</el-tag>
                  <el-tag v-if="data.status === 0" size="small" type="warning">停用</el-tag>
                  <el-button type="primary" link size="small" :icon="Plus" @click="openCreateCat(data.id)" />
                  <el-button type="primary" link size="small" :icon="Edit" @click="openEditCat(data)" />
                  <el-button type="danger" link size="small" :icon="Delete" @click="removeCat(data)" />
                </span>
              </div>
            </template>
          </el-tree>

          <el-empty v-if="!catLoading && categories.length === 0" description="暂无分类，请先添加" :image-size="64" />
        </el-card>
      </el-col>

      <el-col :span="16">
        <el-card class="item-card">
          <template #header>
            <div class="card-head">
              <span>服务列表 · {{ activeCategoryName }}</span>
              <div class="head-actions">
                <el-button
                  v-if="selectedCount"
                  link
                  type="info"
                  size="small"
                  @click="clearPriceListSelection"
                >
                  清空已选
                </el-button>
                <el-button
                  type="warning"
                  plain
                  :icon="Document"
                  size="small"
                  :disabled="!selectedCount"
                  @click="openPriceListDialog"
                >
                  生成价目表{{ selectedCount ? `（${selectedCount}）` : '' }}
                </el-button>
                <el-button type="primary" :icon="Plus" size="small" @click="openCreateItem">
                  新建服务
                </el-button>
              </div>
            </div>
          </template>

          <div class="stats-bar">
            <el-tag effect="plain">合计 {{ total }} 项</el-tag>
            <el-tag type="success" effect="plain">本页启用 {{ enabledCount }}</el-tag>
            <el-tag type="info" effect="plain">本页停用 {{ disabledCount }}</el-tag>
          </div>

          <div class="toolbar">
            <el-input
              v-model="keyword"
              placeholder="搜索服务名称/编码"
              clearable
              style="width: 220px"
              @keyup.enter="loadItems(true)"
              @clear="loadItems(true)"
            />
            <el-select v-model="statusFilter" clearable placeholder="状态" style="width: 120px" @change="loadItems(true)">
              <el-option :value="1" label="启用" />
              <el-option :value="0" label="停用" />
            </el-select>
            <el-button @click="loadItems(true)">查询</el-button>
          </div>

          <el-table
            ref="tableRef"
            v-loading="itemLoading"
            :data="items"
            stripe
            row-key="id"
            @selection-change="onSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="code" label="编码" width="110" />
            <el-table-column prop="name" label="服务名称" min-width="140" />
            <el-table-column prop="description" label="说明" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                <span :class="{ muted: !row.description }">{{ row.description || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="categoryName" label="分类" width="110" />
            <el-table-column label="价格" width="100">
              <template #default="{ row }">¥{{ Number(row.price).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="时长" width="110">
              <template #default="{ row }">
                {{ formatDurationMin(row.durationMin) }}
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
                  {{ row.status === 1 ? '启用' : '停用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openEditItem(row)">编辑</el-button>
                <el-button link type="danger" @click="removeItem(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="pager">
            <el-pagination
              v-model:current-page="page"
              v-model:page-size="pageSize"
              :total="total"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next"
              background
              @current-change="loadItems(false)"
              @size-change="loadItems(true)"
            />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="catDialog" :title="editingCatId ? '编辑分类' : '新建分类'" width="480px" destroy-on-close>
      <el-form label-width="90px">
        <el-form-item label="上级分类">
          <el-select v-model="catForm.parentId" style="width: 100%">
            <el-option :value="0" label="无（一级分类）" />
            <el-option
              v-for="c in flatCategories"
              :key="c.id"
              :value="c.id"
              :label="`${'　'.repeat(c.level)}${c.name}`"
              :disabled="c.id === editingCatId"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="catForm.name" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="catForm.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="catForm.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="catDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveCat">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="itemDialog" :title="editingItemId ? '编辑服务' : '新建服务'" width="560px" destroy-on-close>
      <el-form label-width="90px">
        <el-form-item label="所属分类" required>
          <el-select v-model="itemForm.categoryId" style="width: 100%">
            <el-option
              v-for="c in flatCategories"
              :key="c.id"
              :value="c.id"
              :label="`${'　'.repeat(c.level)}${c.name}`"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="编码">
          <el-input v-model="itemForm.code" placeholder="可选" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="itemForm.name" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="itemForm.description" type="textarea" :rows="3" placeholder="服务说明，将在列表中展示" />
        </el-form-item>
        <el-form-item label="价格">
          <el-input-number v-model="itemForm.price" :min="0" :precision="2" :step="10" />
        </el-form-item>
        <el-form-item label="时长(分)">
          <el-input-number v-model="itemForm.durationMin" :min="0" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="itemForm.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="itemForm.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="itemDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveItem">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="priceListVisible"
      title="生成服务价目表"
      width="560px"
      top="4vh"
      destroy-on-close
    >
      <el-form label-width="96px" class="price-form">
        <el-form-item label="门店" required>
          <el-select v-model="priceListForm.storeId" style="width: 100%">
            <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="价目表模板">
          <el-select v-model="priceListForm.templateId" clearable placeholder="使用默认模板" style="width: 100%">
            <el-option :value="0" label="系统默认（未配置模板时）" />
            <el-option
              v-for="t in priceListTemplates"
              :key="t.id"
              :label="`${t.name}${t.isDefault ? '（默认）' : ''}`"
              :value="t.id"
            />
          </el-select>
          <div class="field-hint">
            可在侧栏「服务目录 → 价目表模板」配置 Logo、营业时间、服务说明等展示项
          </div>
        </el-form-item>
        <el-form-item label="已选服务">
          <el-tag type="info">{{ selectedCount }} 项</el-tag>
          <span class="muted-inline">已记住跨分类/跨页勾选；价目表按分类分组，图标与收银台一致（暂不展示图片）</span>
        </el-form-item>
      </el-form>
      <div class="price-actions">
        <el-button type="primary" :loading="priceListLoading" @click="generatePriceList">生成预览</el-button>
      </div>
      <PosReceiptPanel
        v-if="priceListHtml"
        :html="priceListHtml"
        order-no="price-list"
        title="服务价目表"
        variant="sales-doc"
        aspect-ratio="3:4"
      />
      <el-empty v-else description="选择门店与模板后点击「生成预览」" :image-size="64" />
    </el-dialog>
  </div>
</template>

<style scoped>
.catalog-page { min-height: 560px; }
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}
.head-actions { display: flex; gap: 8px; align-items: center; }
.price-form { margin-bottom: 8px; }
.price-actions { margin-bottom: 12px; }
.field-hint { margin-top: 6px; font-size: 12px; color: #909399; line-height: 1.4; }
.muted-inline { margin-left: 8px; font-size: 13px; color: #909399; }
.cat-card, .item-card { min-height: 560px; }
.cat-stats {
  margin-bottom: 10px;
  font-size: 13px;
  color: #606266;
}
.all-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 10px;
  margin-bottom: 6px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}
.all-row:hover { background: #f5f7fa; }
.all-row.active {
  background: #ecf5ff;
  color: #409eff;
  font-weight: 600;
}
.tree-node {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-right: 4px;
  font-size: 14px;
  min-width: 0;
}
.tree-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 8px;
}
.node-meta {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}
.stats-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
  align-items: center;
}
.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.muted { color: #c0c4cc; }
:deep(.el-card__header) {
  display: flex;
  align-items: center;
}
:deep(.el-tree-node__content) {
  height: 36px;
}
</style>
