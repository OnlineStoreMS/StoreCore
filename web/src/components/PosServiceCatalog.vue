<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Tools } from '@element-plus/icons-vue'
import { formatDurationApprox } from '../utils/formatDuration'
import {
  listServiceCategoryTree,
  listServiceItems,
  type ServiceCategory,
  type ServiceItem,
} from '../api/serviceCatalog'

export interface PosServicePick {
  serviceItemId: number
  name: string
  code?: string
  price: number
  durationMin?: number
  pic?: string
  categoryName?: string
}

const emit = defineEmits<{
  select: [item: PosServicePick]
}>()

const categories = ref<ServiceCategory[]>([])
const items = ref<ServiceItem[]>([])
const activeCategoryId = ref(0)
const keyword = ref('')
const loading = ref(false)
const page = ref(1)
const pageSize = 24
const total = ref(0)

const sidebarCategories = computed(() => {
  const out: { id: number; name: string; count?: number; level: number }[] = [
    { id: 0, name: '全部服务', level: 0 },
  ]
  function walk(list: ServiceCategory[], level: number) {
    for (const c of list) {
      if (c.status === 0) continue
      out.push({ id: c.id, name: c.name, count: c.itemCount, level })
      if (c.children?.length) walk(c.children, level + 1)
    }
  }
  walk(categories.value, 0)
  return out
})

async function loadCategories() {
  try {
    categories.value = await listServiceCategoryTree()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载服务分类失败')
  }
}

async function loadItems(reset = true) {
  if (reset) page.value = 1
  loading.value = true
  try {
    const data = await listServiceItems({
      categoryId: activeCategoryId.value || undefined,
      keyword: keyword.value.trim() || undefined,
      status: 1,
      page: page.value,
      pageSize,
    })
    items.value = data.list
    total.value = data.total
  } catch (e) {
    ElMessage.error((e as Error).message || '加载服务失败')
  } finally {
    loading.value = false
  }
}

function selectCategory(id: number) {
  activeCategoryId.value = id
  keyword.value = ''
  void loadItems(true)
}

function runSearch() {
  void loadItems(true)
}

function pick(item: ServiceItem) {
  emit('select', {
    serviceItemId: item.id,
    name: item.name,
    code: item.code,
    price: item.price || 0,
    durationMin: item.durationMin,
    pic: item.pic,
    categoryName: item.categoryName,
  })
}

onMounted(async () => {
  await loadCategories()
  await loadItems(true)
})
</script>

<template>
  <div class="pos-service-catalog">
    <aside class="category-sidebar">
      <div class="sidebar-title">服务分类</div>
      <button
        v-for="cat in sidebarCategories"
        :key="cat.id"
        type="button"
        class="category-item"
        :class="{ active: activeCategoryId === cat.id }"
        :style="{ paddingLeft: `${12 + cat.level * 12}px` }"
        @click="selectCategory(cat.id)"
      >
        <span class="cat-name">{{ cat.name }}</span>
        <span v-if="cat.count" class="cat-count">{{ cat.count }}</span>
      </button>
    </aside>

    <section class="catalog-main">
      <div class="toolbar">
        <el-input
          v-model="keyword"
          placeholder="搜索服务名称、编码"
          clearable
          class="search-input"
          @keyup.enter="runSearch"
          @clear="runSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" :loading="loading" @click="runSearch">搜索</el-button>
      </div>
      <div class="mode-hint">点击服务加入购物车，可与商品一起结算</div>

      <div v-loading="loading" class="product-grid-wrap">
        <div v-if="!loading && items.length === 0" class="grid-empty">暂无服务项目，请先在「服务目录」中维护</div>
        <div v-else class="product-grid">
          <button
            v-for="item in items"
            :key="item.id"
            type="button"
            class="product-card"
            @click="pick(item)"
          >
            <div class="card-pic service">
              <el-image v-if="item.pic" :src="item.pic" fit="cover" class="card-img" lazy>
                <template #error>
                  <div class="pic-fallback"><el-icon><Tools /></el-icon></div>
                </template>
              </el-image>
              <div v-else class="pic-fallback"><el-icon><Tools /></el-icon></div>
            </div>
            <div class="card-body">
              <div class="card-title">{{ item.name }}</div>
              <div class="card-sub">
                {{ item.categoryName || '服务' }}
                <template v-if="item.durationMin"> · {{ formatDurationApprox(item.durationMin) }}</template>
              </div>
              <div class="card-footer">
                <span class="card-price">¥{{ Number(item.price).toFixed(2) }}</span>
                <el-tag size="small" type="warning" effect="plain">服务</el-tag>
              </div>
            </div>
          </button>
        </div>
      </div>

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
</template>

<style scoped>
.pos-service-catalog {
  display: flex;
  min-height: 0;
  flex: 1;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid #ebeef5;
}
.category-sidebar {
  width: 148px;
  flex-shrink: 0;
  background: #1f2937;
  color: #e5e7eb;
  overflow-y: auto;
}
.sidebar-title {
  padding: 14px 12px 10px;
  font-size: 12px;
  font-weight: 600;
  color: #9ca3af;
}
.category-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  width: 100%;
  border: none;
  background: transparent;
  color: inherit;
  text-align: left;
  padding: 10px 12px;
  cursor: pointer;
  font-size: 13px;
}
.category-item:hover { background: rgba(255,255,255,0.06); }
.category-item.active { background: #e6a23c; color: #fff; font-weight: 600; }
.cat-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cat-count { font-size: 11px; opacity: 0.75; flex-shrink: 0; }
.catalog-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 14px 16px 12px;
}
.toolbar { display: flex; gap: 8px; margin-bottom: 10px; }
.search-input { flex: 1; }
.mode-hint { font-size: 12px; color: #909399; margin-bottom: 10px; }
.product-grid-wrap { flex: 1; min-height: 280px; overflow-y: auto; }
.grid-empty {
  display: flex; align-items: center; justify-content: center;
  height: 240px; color: #909399;
}
.product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(148px, 1fr));
  gap: 12px;
}
.product-card {
  border: 1px solid #ebeef5;
  border-radius: 10px;
  background: #fff;
  padding: 0;
  cursor: pointer;
  text-align: left;
  overflow: hidden;
  transition: border-color 0.15s, box-shadow 0.15s, transform 0.12s;
}
.product-card:hover {
  border-color: #e6a23c;
  box-shadow: 0 6px 16px rgba(230, 162, 60, 0.15);
  transform: translateY(-2px);
}
.card-pic { aspect-ratio: 1; background: #fafafa; }
.card-pic.service { background: #fff7e6; }
.card-img { width: 100%; height: 100%; }
.pic-fallback {
  width: 100%; height: 100%; min-height: 120px;
  display: flex; align-items: center; justify-content: center;
  color: #e6a23c; font-size: 32px; background: #fff7e6;
}
.card-body { padding: 8px 10px 10px; }
.card-title {
  font-size: 13px; font-weight: 500; color: #303133; line-height: 1.35;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
  min-height: 2.7em;
}
.card-sub { margin-top: 2px; font-size: 11px; color: #909399; }
.card-footer {
  display: flex; align-items: center; justify-content: space-between;
  margin-top: 6px; gap: 4px;
}
.card-price { font-size: 15px; font-weight: 700; color: #f56c6c; }
.pager { display: flex; justify-content: center; padding-top: 12px; }
</style>
