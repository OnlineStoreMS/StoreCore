<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createServiceCategory,
  createServiceItem,
  deleteServiceCategory,
  deleteServiceItem,
  listServiceCategoryTree,
  listServiceItems,
  updateServiceCategory,
  updateServiceItem,
  type ServiceCategory,
  type ServiceItem,
} from '../../api/serviceCatalog'

const loading = ref(false)
const categories = ref<ServiceCategory[]>([])
const items = ref<ServiceItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const keyword = ref('')
const activeCategoryId = ref(0)

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

const sidebarCats = computed(() => {
  const out: { id: number; name: string; count?: number; level: number }[] = [
    { id: 0, name: '全部服务', level: 0 },
  ]
  for (const row of flatCategories.value) {
    const found = findCat(categories.value, row.id)
    out.push({ id: row.id, name: row.name, count: found?.itemCount, level: row.level })
  }
  return out
})

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
  categories.value = await listServiceCategoryTree()
}

async function loadItems(reset = true) {
  if (reset) page.value = 1
  loading.value = true
  try {
    const data = await listServiceItems({
      categoryId: activeCategoryId.value || undefined,
      keyword: keyword.value.trim() || undefined,
      page: page.value,
      pageSize,
    })
    items.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function selectCategory(id: number) {
  activeCategoryId.value = id
  void loadItems(true)
}

function openCreateCat(parentId = 0) {
  editingCatId.value = undefined
  Object.assign(catForm, { parentId, name: '', sort: 0, status: 1 })
  catDialog.value = true
}

function openEditCat(row: { id: number; name: string; level: number }) {
  const cat = findCat(categories.value, row.id)
  if (!cat) return
  editingCatId.value = cat.id
  Object.assign(catForm, {
    parentId: cat.parentId,
    name: cat.name,
    sort: cat.sort,
    status: cat.status,
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

async function removeCat(row: { id: number; name: string }) {
  await ElMessageBox.confirm(`删除分类「${row.name}」？需先清空子分类与服务项目。`, '确认', { type: 'warning' })
  try {
    await deleteServiceCategory(row.id)
    ElMessage.success('已删除')
    if (activeCategoryId.value === row.id) activeCategoryId.value = 0
    await loadCategories()
    await loadItems(true)
  } catch (e) {
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
  await ElMessageBox.confirm(`删除服务「${row.name}」？`, '确认', { type: 'warning' })
  await deleteServiceItem(row.id)
  ElMessage.success('已删除')
  await loadCategories()
  await loadItems(false)
}

onMounted(async () => {
  await loadCategories()
  await loadItems(true)
})
</script>

<template>
  <div class="catalog-page">
    <div class="page-header">
      <div>
        <h2>服务目录</h2>
        <p class="desc">维护服务分类与服务项目，可在收银台与商品一起结算。</p>
      </div>
      <div class="header-actions">
        <el-button @click="openCreateCat(0)">新建分类</el-button>
        <el-button type="primary" @click="openCreateItem">新建服务</el-button>
      </div>
    </div>

    <div class="catalog-body">
      <aside class="cat-side">
        <div class="side-title">服务分类</div>
        <button
          v-for="cat in sidebarCats"
          :key="cat.id"
          type="button"
          class="cat-item"
          :class="{ active: activeCategoryId === cat.id, indent: cat.level > 0 }"
          @click="selectCategory(cat.id)"
        >
          <span>{{ cat.name }}</span>
          <span v-if="cat.count" class="count">{{ cat.count }}</span>
        </button>
        <div class="side-actions">
          <el-button
            v-for="cat in flatCategories"
            :key="'edit-' + cat.id"
            link
            size="small"
            @click="openEditCat(cat)"
          >
            编辑 {{ cat.name }}
          </el-button>
        </div>
      </aside>

      <section class="item-main">
        <div class="toolbar">
          <el-input
            v-model="keyword"
            placeholder="搜索服务名称/编码"
            clearable
            style="width: 240px"
            @keyup.enter="loadItems(true)"
          />
          <el-button @click="loadItems(true)">查询</el-button>
          <el-button
            v-if="activeCategoryId"
            link
            type="primary"
            @click="openCreateCat(activeCategoryId)"
          >
            在当前分类下新建子分类
          </el-button>
          <el-button
            v-if="activeCategoryId"
            link
            type="danger"
            @click="removeCat({ id: activeCategoryId, name: findCat(categories, activeCategoryId)?.name || '' })"
          >
            删除当前分类
          </el-button>
        </div>

        <el-table v-loading="loading" :data="items" stripe>
          <el-table-column prop="code" label="编码" width="120" />
          <el-table-column prop="name" label="服务名称" min-width="160" />
          <el-table-column prop="categoryName" label="分类" width="120" />
          <el-table-column label="价格" width="100">
            <template #default="{ row }">¥{{ Number(row.price).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="时长(分)" width="90" prop="durationMin" />
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

        <div v-if="total > pageSize" class="pager">
          <el-pagination
            v-model:current-page="page"
            :page-size="pageSize"
            :total="total"
            layout="prev, pager, next"
            background
            @current-change="loadItems(false)"
          />
        </div>
      </section>
    </div>

    <el-dialog v-model="catDialog" :title="editingCatId ? '编辑分类' : '新建分类'" width="480px">
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
          <el-radio-group v-model="catForm.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">停用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="catDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveCat">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="itemDialog" :title="editingItemId ? '编辑服务' : '新建服务'" width="560px">
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
        <el-form-item label="价格">
          <el-input-number v-model="itemForm.price" :min="0" :precision="2" :step="10" />
        </el-form-item>
        <el-form-item label="时长(分)">
          <el-input-number v-model="itemForm.durationMin" :min="0" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="itemForm.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="itemForm.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="itemForm.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">停用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="itemDialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveItem">保存</el-button>
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
.page-header h2 { margin: 0 0 6px; font-size: 20px; }
.desc { margin: 0; color: #909399; font-size: 13px; }
.header-actions { display: flex; gap: 8px; }
.catalog-body {
  display: flex;
  gap: 12px;
  min-height: 520px;
}
.cat-side {
  width: 200px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 10px;
  overflow: auto;
}
.side-title {
  padding: 12px 14px 8px;
  font-size: 12px;
  color: #909399;
  font-weight: 600;
}
.cat-item {
  display: flex;
  justify-content: space-between;
  width: 100%;
  border: none;
  background: transparent;
  text-align: left;
  padding: 10px 14px;
  cursor: pointer;
  font-size: 14px;
  color: #303133;
}
.cat-item:hover { background: #f5f7fa; }
.cat-item.active { background: #ecf5ff; color: #409eff; font-weight: 600; }
.cat-item.indent { padding-left: 28px; font-size: 13px; }
.count { font-size: 11px; color: #909399; }
.side-actions {
  border-top: 1px solid #f0f2f5;
  padding: 8px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
}
.item-main {
  flex: 1;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 10px;
  padding: 14px;
}
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; align-items: center; }
.pager { display: flex; justify-content: center; margin-top: 12px; }
</style>
