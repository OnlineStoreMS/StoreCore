<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Picture, Search } from '@element-plus/icons-vue'
import {
  getProductSkus,
  listCatalogProducts,
  listCategories,
  flattenCategoryTree,
  formatSpecLabel,
  resolvePic,
  type CatalogProduct,
  type CatalogSku,
  type CategoryItem,
} from '../api/catalog'
import { listInventoriesByStore } from '../api/inventory'
import { searchProductSkus, type ProductSkuSearchItem } from '../api/productSku'
import PosSkuSelectDialog from './PosSkuSelectDialog.vue'

const props = defineProps<{
  storeId?: number
  /** 收银台默认 true：门店无货不可选；销售单/盘点传 false */
  requireStoreStock?: boolean
  /** 库存盘点：不展示统一商品库库存，只同步门店库存 */
  stocktakeMode?: boolean
}>()

const emit = defineEmits<{
  select: [sku: ProductSkuSearchItem]
}>()

const blockEmptyStore = computed(() => !props.stocktakeMode && props.requireStoreStock !== false)
const stocktakeMode = computed(() => !!props.stocktakeMode)

const categories = ref<CategoryItem[]>([])
const activeCategoryId = ref(0)
const keyword = ref('')
const searchMode = ref(false)

const products = ref<CatalogProduct[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = 24
const total = ref(0)

/** 门店库存 skuId → quantity；缺失视为 0 */
const storeQtyMap = ref<Record<number, number>>({})
/** 商品是否在门店有可用规格库存 */
const productAvailable = ref<Record<number, boolean>>({})
/** 商品维度门店库存合计（盘点模式卡片展示） */
const productStoreQty = ref<Record<number, number>>({})
const productSkuCache = ref<Record<number, CatalogSku[]>>({})

const skuDialogVisible = ref(false)
const skuDialogProduct = ref<CatalogProduct | null>(null)
const skuDialogSkus = ref<(CatalogSku & { storeQty: number })[]>([])
const skuDialogLoading = ref(false)

const sidebarCategories = computed(() => {
  const items: { id: number; name: string; count?: number; level: number }[] = [
    { id: 0, name: '全部商品', level: 0 },
  ]
  items.push(...flattenCategoryTree(categories.value, 0))
  return items
})

const effectiveCategoryId = computed(() => activeCategoryId.value)

const gridItems = computed(() =>
  products.value.map((p) => {
    const available = productAvailable.value[p.id]
    // 收银台：仅门店有货可选；未确认/无货/无该商品一律置灰
    const disabled = blockEmptyStore.value && available !== true
    const storeQtySum = productStoreQty.value[p.id]
    return {
      key: `product-${p.id}`,
      pic: resolvePic(p.pic),
      title: p.name,
      subtitle: p.skuCount && p.skuCount > 1 ? `${p.skuCount} 规格可选` : p.categoryName || '',
      price: p.price,
      stock: p.stock,
      storeQty: storeQtySum,
      disabled,
      product: p,
    }
  }),
)

async function loadStoreStock() {
  storeQtyMap.value = {}
  if (!props.storeId) return
  try {
    const rows = await listInventoriesByStore(props.storeId)
    const map: Record<number, number> = {}
    for (const row of rows) {
      map[row.skuId] = row.quantity
    }
    storeQtyMap.value = map
  } catch (e) {
    ElMessage.error((e as Error).message || '加载门店库存失败')
  }
}

function storeQtyOf(skuId: number) {
  return storeQtyMap.value[skuId] ?? 0
}

async function resolveProductAvailability(list: CatalogProduct[]) {
  // 销售单等：不限制可选，也不强刷门店合计（避免额外 SKU 请求）
  if (!blockEmptyStore.value && !stocktakeMode.value) {
    const nextAvail = { ...productAvailable.value }
    for (const p of list) nextAvail[p.id] = true
    productAvailable.value = nextAvail
    return
  }

  // 收银台且门店无库存记录：全部置灰
  if (blockEmptyStore.value && (!props.storeId || Object.keys(storeQtyMap.value).length === 0)) {
    const nextAvail = { ...productAvailable.value }
    const nextStore = { ...productStoreQty.value }
    for (const p of list) {
      nextAvail[p.id] = false
      nextStore[p.id] = 0
    }
    productAvailable.value = nextAvail
    productStoreQty.value = nextStore
    return
  }

  const nextAvail = { ...productAvailable.value }
  const nextStore = { ...productStoreQty.value }
  await Promise.all(
    list.map(async (p) => {
      try {
        let skus = productSkuCache.value[p.id]
        if (!skus) {
          const detail = await getProductSkus(p.id)
          skus = detail.skus
          productSkuCache.value[p.id] = skus
        }
        const qtySum = skus.reduce((sum, s) => sum + storeQtyOf(s.id), 0)
        nextStore[p.id] = qtySum
        nextAvail[p.id] = blockEmptyStore.value ? skus.some((s) => storeQtyOf(s.id) > 0) : true
      } catch {
        nextAvail[p.id] = !blockEmptyStore.value
        nextStore[p.id] = 0
      }
    }),
  )
  productAvailable.value = nextAvail
  productStoreQty.value = nextStore
}

async function loadCategories() {
  try {
    categories.value = await listCategories()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载分类失败')
  }
}

async function loadProducts(resetPage = true) {
  if (resetPage) page.value = 1
  loading.value = true
  try {
    const q = keyword.value.trim()
    // 盘点搜索：走超级搜索，支持商品名 / SKU 编码 / 规格值
    if (stocktakeMode.value && q) {
      const data = await searchProductSkus({ keyword: q, page: page.value, pageSize })
      const byProduct = new Map<number, CatalogProduct>()
      for (const item of data.list) {
        if (!byProduct.has(item.productId)) {
          byProduct.set(item.productId, {
            id: item.productId,
            name: item.productName,
            pic: resolvePic(item.productPic, item.pic),
            price: item.price,
            stock: item.stock,
            skuCount: 1,
            categoryId: 0,
            categoryName: item.categoryName,
            materialCode: item.materialCode,
            publishStatus: 1,
          })
        } else {
          const p = byProduct.get(item.productId)!
          p.skuCount = (p.skuCount || 1) + 1
        }
      }
      const list = Array.from(byProduct.values())
      products.value = list
      total.value = data.total
      await resolveProductAvailability(list)
      return
    }

    const data = await listCatalogProducts({
      categoryId: searchMode.value ? undefined : effectiveCategoryId.value || undefined,
      keyword: q || undefined,
      page: page.value,
      pageSize,
    })
    products.value = data.list
    total.value = data.total
    await resolveProductAvailability(data.list)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载商品失败')
  } finally {
    loading.value = false
  }
}

async function runSearch() {
  const q = keyword.value.trim()
  searchMode.value = !!q
  await loadProducts(true)
}

function selectCategory(id: number) {
  activeCategoryId.value = id
  keyword.value = ''
  searchMode.value = false
  void loadProducts(true)
}

function onGridItemClick(item: (typeof gridItems.value)[number]) {
  if (item.disabled) {
    ElMessage.warning('门店无货或库存为 0，需仓库调货')
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
    let skus = productSkuCache.value[product.id]
    if (!skus) {
      const detail = await getProductSkus(product.id)
      skus = detail.skus
      productSkuCache.value[product.id] = skus
    }
    const withStore = skus.map((s) => ({ ...s, storeQty: storeQtyOf(s.id) }))
    skuDialogSkus.value = withStore
    if (blockEmptyStore.value) {
      productAvailable.value = {
        ...productAvailable.value,
        [product.id]: withStore.some((s) => s.storeQty > 0),
      }
    }

    if (withStore.length === 1) {
      if (blockEmptyStore.value && withStore[0].storeQty <= 0) {
        ElMessage.warning('门店无货或库存为 0，需仓库调货')
        return
      }
      skuDialogVisible.value = false
      emit('select', {
        productId: product.id,
        productName: product.name,
        productPic: resolvePic(product.pic),
        skuId: withStore[0].id,
        skuCode: withStore[0].skuCode,
        specs: withStore[0].specs || {},
        specLabel: formatSpecLabel(withStore[0].specs),
        price: withStore[0].price || product.price,
        stock: withStore[0].stock,
        pic: resolvePic(withStore[0].pic, product.pic),
        storeQty: withStore[0].storeQty,
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

function onSkuSelect(sku: ProductSkuSearchItem) {
  if (blockEmptyStore.value && storeQtyOf(sku.skuId) <= 0) {
    ElMessage.warning('门店无货或库存为 0，需仓库调货')
    return
  }
  emit('select', {
    ...sku,
    storeQty: sku.storeQty ?? storeQtyOf(sku.skuId),
  })
}

watch(
  () => props.storeId,
  async () => {
    productAvailable.value = {}
    productStoreQty.value = {}
    productSkuCache.value = {}
    await loadStoreStock()
    if (products.value.length) {
      await resolveProductAvailability(products.value)
    }
  },
)

onMounted(async () => {
  await loadStoreStock()
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
        :class="{ active: activeCategoryId === cat.id && !searchMode }"
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
          :placeholder="stocktakeMode ? '搜索商品名称、SKU、货号' : '搜索商品名称、货号、资料编码'"
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

      <div class="mode-hint">
        {{
          stocktakeMode
            ? '库存盘点：按分类浏览或搜索商品/SKU · 先选商品再选规格 · 数量同步当前门店库存（不参考统一商品库）'
            : blockEmptyStore
              ? '即时零售按门店库存售卖 · 门店库存为 0 或无该商品置灰不可选 · 卡片数字为仓库库存（仅参考）'
              : '按分类筛选商品 · 点击选择规格加入销售单 · 可同步参考仓库/门店库存'
        }}
      </div>

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
            :class="{ disabled: item.disabled }"
            @click="onGridItemClick(item)"
          >
            <div class="card-pic">
              <el-image v-if="item.pic" :src="item.pic" fit="cover" class="card-img" lazy>
                <template #error>
                  <div class="pic-fallback"><el-icon><Picture /></el-icon></div>
                </template>
              </el-image>
              <div v-else class="pic-fallback"><el-icon><Picture /></el-icon></div>
              <div v-if="item.disabled" class="card-badge">门店无货</div>
            </div>
            <div class="card-body">
              <div class="card-title">{{ item.title }}</div>
              <div v-if="item.subtitle" class="card-sub">{{ item.subtitle }}</div>
              <div class="card-footer">
                <span class="card-price">¥{{ item.price.toFixed(2) }}</span>
                <span v-if="stocktakeMode" class="card-stock">
                  门店 {{ item.storeQty ?? 0 }}
                </span>
                <span v-else class="card-stock">仓库 {{ item.stock ?? 0 }}</span>
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
          @current-change="onPageChange"
        />
      </div>
    </section>

    <PosSkuSelectDialog
      v-model:visible="skuDialogVisible"
      :product="skuDialogProduct"
      :skus="skuDialogSkus"
      :loading="skuDialogLoading"
      :require-store-stock="blockEmptyStore"
      :stocktake-mode="stocktakeMode"
      @select="onSkuSelect"
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
  width: 148px;
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
  padding: 10px 12px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.15s;
}
.category-item:hover { background: rgba(255, 255, 255, 0.06); }
.category-item.active {
  background: #409eff;
  color: #fff;
  font-weight: 600;
}
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
  position: relative;
}
.product-card:hover:not(.disabled) {
  border-color: #409eff;
  box-shadow: 0 6px 16px rgba(64, 158, 255, 0.12);
  transform: translateY(-2px);
}
.product-card.disabled {
  opacity: 0.55;
  cursor: not-allowed;
  filter: grayscale(0.35);
}
.card-pic { aspect-ratio: 1; background: #fafafa; position: relative; }
.card-img { width: 100%; height: 100%; }
.card-badge {
  position: absolute;
  left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.62);
  color: #fff;
  font-size: 11px;
  text-align: center;
  padding: 4px 0;
}
.pic-fallback {
  width: 100%; height: 100%; min-height: 120px;
  display: flex; align-items: center; justify-content: center;
  color: #c0c4cc; font-size: 32px; background: #f5f7fa;
}
.card-body { padding: 8px 10px 10px; }
.card-title {
  font-size: 13px; font-weight: 500; color: #303133; line-height: 1.35;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
  overflow: hidden; min-height: 2.7em;
}
.card-sub {
  margin-top: 2px; font-size: 11px; color: #909399;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.card-footer {
  display: flex; align-items: baseline; justify-content: space-between;
  margin-top: 6px; gap: 4px;
}
.card-price { font-size: 15px; font-weight: 700; color: #f56c6c; }
.card-stock { font-size: 11px; color: #909399; }
.pager { display: flex; justify-content: center; padding-top: 12px; }
</style>
