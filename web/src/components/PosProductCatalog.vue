<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Picture, Search } from '@element-plus/icons-vue'
import {
  getProductSkus,
  listCatalogProducts,
  listCategories,
  formatSpecLabel,
  resolvePic,
  type CatalogProduct,
  type CatalogSku,
  type CategoryItem,
} from '../api/catalog'
import { searchProductSkus, type ProductSkuSearchItem } from '../api/productSku'
import PosSkuSelectDialog from './PosSkuSelectDialog.vue'

const emit = defineEmits<{
  select: [sku: ProductSkuSearchItem]
}>()

const categories = ref<CategoryItem[]>([])
const activeCategoryId = ref(0)
const keyword = ref('')
const searchMode = ref(false)

const products = ref<CatalogProduct[]>([])
const searchResults = ref<ProductSkuSearchItem[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = 24
const total = ref(0)

const skuDialogVisible = ref(false)
const skuDialogProduct = ref<CatalogProduct | null>(null)
const skuDialogSkus = ref<CatalogSku[]>([])
const skuDialogLoading = ref(false)

const sidebarCategories = computed(() => {
  const items: { id: number; name: string; count?: number; level: number }[] = [
    { id: 0, name: '全部商品', level: 0 },
  ]
  for (const cat of categories.value) {
    items.push({
      id: cat.id,
      name: cat.name,
      count: cat.productCount,
      level: 0,
    })
    for (const child of cat.children || []) {
      if (child.showStatus === 0) continue
      items.push({
        id: child.id,
        name: child.name,
        count: child.productCount,
        level: 1,
      })
    }
  }
  return items
})

const effectiveCategoryId = computed(() => activeCategoryId.value)

const gridItems = computed(() => {
  if (searchMode.value) {
    return searchResults.value.map((item) => ({
      kind: 'sku' as const,
      key: `sku-${item.skuId}`,
      pic: resolvePic(item.pic, item.productPic),
      title: item.productName,
      subtitle: item.specLabel || formatSpecLabel(item.specs),
      price: item.price,
      stock: item.stock,
      sku: item,
    }))
  }
  return products.value.map((p) => ({
    kind: 'product' as const,
    key: `product-${p.id}`,
    pic: resolvePic(p.pic),
    title: p.name,
    subtitle: p.skuCount && p.skuCount > 1 ? `${p.skuCount} 规格可选` : p.categoryName || '',
    price: p.price,
    stock: p.stock,
    product: p,
  }))
})

async function loadCategories() {
  try {
    categories.value = await listCategories()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载分类失败')
  }
}

async function loadProducts(resetPage = true) {
  if (searchMode.value) return
  if (resetPage) page.value = 1
  loading.value = true
  try {
    const data = await listCatalogProducts({
      categoryId: effectiveCategoryId.value || undefined,
      page: page.value,
      pageSize,
    })
    products.value = data.list
    total.value = data.total
  } catch (e) {
    ElMessage.error((e as Error).message || '加载商品失败')
  } finally {
    loading.value = false
  }
}

async function runSearch() {
  const q = keyword.value.trim()
  if (!q) {
    searchMode.value = false
    searchResults.value = []
    await loadProducts(true)
    return
  }
  searchMode.value = true
  loading.value = true
  try {
    const data = await searchProductSkus({ keyword: q, page: 1, pageSize: 48 })
    searchResults.value = data.list
    total.value = data.total
  } catch (e) {
    ElMessage.error((e as Error).message || '搜索失败')
  } finally {
    loading.value = false
  }
}

function selectCategory(id: number) {
  activeCategoryId.value = id
  keyword.value = ''
  searchMode.value = false
  void loadProducts(true)
}

function onGridItemClick(item: (typeof gridItems.value)[number]) {
  if (item.kind === 'sku') {
    emit('select', item.sku)
    return
  }
  void openProduct(item.product)
}

async function openProduct(product: CatalogProduct) {
  skuDialogProduct.value = product
  skuDialogSkus.value = []
  skuDialogLoading.value = true
  skuDialogVisible.value = true
  try {
    const detail = await getProductSkus(product.id)
    skuDialogSkus.value = detail.skus
    if (detail.skus.length === 1) {
      skuDialogVisible.value = false
      emit('select', {
        productId: product.id,
        productName: product.name,
        productPic: resolvePic(product.pic),
        skuId: detail.skus[0].id,
        skuCode: detail.skus[0].skuCode,
        specs: detail.skus[0].specs || {},
        specLabel: formatSpecLabel(detail.skus[0].specs),
        price: detail.skus[0].price || product.price,
        stock: detail.skus[0].stock,
        pic: resolvePic(detail.skus[0].pic, product.pic),
      })
    }
  } catch (e) {
    skuDialogVisible.value = false
    ElMessage.error((e as Error).message || '加载 SKU 失败')
  } finally {
    skuDialogLoading.value = false
  }
}

function onPageChange(p: number) {
  page.value = p
  void loadProducts(false)
}

onMounted(async () => {
  await loadCategories()
  await loadProducts(true)
})
</script>

<template>
  <div class="pos-catalog">
    <aside class="category-sidebar">
      <div class="sidebar-title">商品分类</div>
      <button
        v-for="cat in sidebarCategories"
        :key="cat.id"
        type="button"
        class="category-item"
        :class="{ active: activeCategoryId === cat.id && !searchMode, indent: cat.level > 0 }"
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
          placeholder="扫码 / 搜索商品名、编码、规格"
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

      <div v-if="searchMode" class="mode-hint">搜索结果 · 点击 SKU 加入购物车</div>
      <div v-else class="mode-hint">已上架商品 · 点击卡片选择规格</div>

      <div v-loading="loading" class="product-grid-wrap">
        <div v-if="!loading && gridItems.length === 0" class="grid-empty">
          {{ searchMode ? '未找到匹配商品' : '该分类暂无商品' }}
        </div>
        <div v-else class="product-grid">
          <button
            v-for="item in gridItems"
            :key="item.key"
            type="button"
            class="product-card"
            @click="onGridItemClick(item)"
          >
            <div class="card-pic">
              <el-image v-if="item.pic" :src="item.pic" fit="cover" class="card-img" lazy>
                <template #error>
                  <div class="pic-fallback"><el-icon><Picture /></el-icon></div>
                </template>
              </el-image>
              <div v-else class="pic-fallback"><el-icon><Picture /></el-icon></div>
            </div>
            <div class="card-body">
              <div class="card-title">{{ item.title }}</div>
              <div v-if="item.subtitle" class="card-sub">{{ item.subtitle }}</div>
              <div class="card-footer">
                <span class="card-price">¥{{ item.price.toFixed(2) }}</span>
                <span class="card-stock">库存 {{ item.stock ?? 0 }}</span>
              </div>
            </div>
          </button>
        </div>
      </div>

      <div v-if="!searchMode && total > pageSize" class="pager">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          background
          @current-change="onPageChange"
        />
      </div>
    </section>

    <PosSkuSelectDialog
      v-model:visible="skuDialogVisible"
      :product="skuDialogProduct"
      :skus="skuDialogSkus"
      :loading="skuDialogLoading"
      @select="emit('select', $event)"
    />
  </div>
</template>

<style scoped>
.pos-catalog {
  display: flex;
  gap: 0;
  min-height: 0;
  flex: 1;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid #ebeef5;
}
.category-sidebar {
  width: 132px;
  flex-shrink: 0;
  background: #1f2937;
  color: #e5e7eb;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}
.sidebar-title {
  padding: 14px 12px 10px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.05em;
  color: #9ca3af;
  text-transform: uppercase;
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
  padding: 12px 14px;
  cursor: pointer;
  font-size: 14px;
  transition: background 0.15s;
}
.category-item:hover {
  background: rgba(255, 255, 255, 0.06);
}
.category-item.active {
  background: #409eff;
  color: #fff;
  font-weight: 600;
}
.category-item.indent {
  padding-left: 22px;
  font-size: 13px;
}
.cat-count {
  font-size: 11px;
  opacity: 0.75;
}
.catalog-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 14px 16px 12px;
}
.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}
.search-input {
  flex: 1;
}
.mode-hint {
  font-size: 12px;
  color: #909399;
  margin-bottom: 10px;
}
.product-grid-wrap {
  flex: 1;
  min-height: 280px;
  overflow-y: auto;
}
.grid-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 240px;
  color: #909399;
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
  border-color: #409eff;
  box-shadow: 0 6px 16px rgba(64, 158, 255, 0.12);
  transform: translateY(-2px);
}
.card-pic {
  aspect-ratio: 1;
  background: #fafafa;
}
.card-img {
  width: 100%;
  height: 100%;
}
.pic-fallback {
  width: 100%;
  height: 100%;
  min-height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #c0c4cc;
  font-size: 32px;
  background: #f5f7fa;
}
.card-body {
  padding: 8px 10px 10px;
}
.card-title {
  font-size: 13px;
  font-weight: 500;
  color: #303133;
  line-height: 1.35;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: 2.7em;
}
.card-sub {
  margin-top: 2px;
  font-size: 11px;
  color: #909399;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.card-footer {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-top: 6px;
  gap: 4px;
}
.card-price {
  font-size: 15px;
  font-weight: 700;
  color: #f56c6c;
}
.card-stock {
  font-size: 11px;
  color: #909399;
}
.pager {
  display: flex;
  justify-content: center;
  padding-top: 12px;
}
</style>
