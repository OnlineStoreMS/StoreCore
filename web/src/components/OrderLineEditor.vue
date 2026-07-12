<script setup lang="ts">
import { computed, ref } from 'vue'
import { Delete, Picture, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import PosSkuSelectDialog from './PosSkuSelectDialog.vue'
import {
  formatSpecLabel,
  getProductSkus,
  listCatalogProducts,
  resolvePic,
  type CatalogProduct,
  type CatalogSku,
} from '../api/catalog'
import { listInventoryBySkus } from '../api/inventory'
import type { ProductSkuSearchItem } from '../api/productSku'

export interface OrderLine {
  skuId: number
  productName: string
  skuCode?: string
  specLabel?: string
  pic?: string
  quantity: number
  originalPrice: number
  discount: number
  unitPrice: number
}

const props = defineProps<{ storeId?: number }>()
const lines = defineModel<OrderLine[]>({ required: true })

const keyword = ref('')
const searching = ref(false)
const products = ref<CatalogProduct[]>([])
const skuDialogVisible = ref(false)
const skuLoading = ref(false)
const activeProduct = ref<CatalogProduct | null>(null)
const activeSkus = ref<(CatalogSku & { storeQty?: number })[]>([])

const payableTotal = computed(() =>
  lines.value.reduce((sum, l) => sum + l.unitPrice * l.quantity, 0),
)
const originalTotal = computed(() =>
  lines.value.reduce((sum, l) => sum + (l.originalPrice || l.unitPrice) * l.quantity, 0),
)

async function searchProducts() {
  searching.value = true
  try {
    const data = await listCatalogProducts({
      keyword: keyword.value.trim() || undefined,
      page: 1,
      pageSize: 24,
    })
    products.value = data.list || []
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    searching.value = false
  }
}

async function openProduct(p: CatalogProduct) {
  activeProduct.value = p
  skuDialogVisible.value = true
  skuLoading.value = true
  try {
    const detail = await getProductSkus(p.id)
    const skus = detail.skus || []
    const qtyMap: Record<number, number> = {}
    if (props.storeId && skus.length) {
      const rows = await listInventoryBySkus(props.storeId, skus.map((s) => s.id))
      for (const row of rows || []) qtyMap[row.skuId] = row.quantity
    }
    activeSkus.value = skus.map((s) => ({ ...s, storeQty: qtyMap[s.id] ?? 0 }))
  } catch (e) {
    ElMessage.error((e as Error).message)
    skuDialogVisible.value = false
  } finally {
    skuLoading.value = false
  }
}

function addSku(sku: ProductSkuSearchItem) {
  const price = sku.price || 0
  const existing = lines.value.find((l) => l.skuId > 0 && l.skuId === sku.skuId)
  if (existing) {
    existing.quantity += 1
    return
  }
  lines.value.push({
    skuId: sku.skuId,
    productName: sku.productName,
    skuCode: sku.skuCode,
    specLabel: sku.specLabel || formatSpecLabel(sku.specs),
    pic: resolvePic(sku.pic, sku.productPic),
    quantity: 1,
    originalPrice: price,
    discount: 10,
    unitPrice: price,
  })
}

function addManualLine() {
  lines.value.push({
    skuId: 0,
    productName: '',
    skuCode: '',
    specLabel: '',
    pic: '',
    quantity: 1,
    originalPrice: 0,
    discount: 10,
    unitPrice: 0,
  })
}

function onDiscountChange(row: OrderLine) {
  let d = Number(row.discount)
  if (!Number.isFinite(d) || d <= 0) d = 10
  if (d > 10) d = 10
  row.discount = Math.round(d * 100) / 100
  const orig = row.originalPrice > 0 ? row.originalPrice : row.unitPrice
  row.originalPrice = orig
  row.unitPrice = Math.round(orig * (row.discount / 10) * 100) / 100
}

function onUnitPriceChange(row: OrderLine) {
  const orig = row.originalPrice > 0 ? row.originalPrice : row.unitPrice
  row.originalPrice = orig
  if (orig > 0) {
    row.discount = Math.round((row.unitPrice / orig) * 10 * 100) / 100
  } else {
    row.discount = 10
  }
}

function removeLine(index: number) {
  lines.value.splice(index, 1)
}
</script>

<template>
  <div class="order-lines">
    <div class="picker">
      <div class="picker-bar">
        <el-input
          v-model="keyword"
          clearable
          placeholder="搜索商品名称 / 编码"
          class="search-input"
          @keyup.enter="searchProducts"
        >
          <template #append>
            <el-button :icon="Search" :loading="searching" @click="searchProducts">搜索</el-button>
          </template>
        </el-input>
        <el-button :icon="Plus" @click="addManualLine">手动添加</el-button>
      </div>
      <div v-loading="searching" class="product-grid">
        <button
          v-for="p in products"
          :key="p.id"
          type="button"
          class="product-card"
          @click="openProduct(p)"
        >
          <div class="pic-wrap">
            <el-image
              v-if="resolvePic(p.pic)"
              :src="resolvePic(p.pic)"
              :preview-src-list="[resolvePic(p.pic)]"
              preview-teleported
              fit="cover"
              class="pic"
              @click.stop
            >
              <template #error>
                <div class="pic-fallback"><el-icon><Picture /></el-icon></div>
              </template>
            </el-image>
            <div v-else class="pic-fallback"><el-icon><Picture /></el-icon></div>
          </div>
          <div class="info">
            <div class="name">{{ p.name }}</div>
            <div class="meta">¥{{ (p.price || 0).toFixed(2) }} · 仓 {{ p.stock ?? 0 }}</div>
            <div class="hint">点击选规格</div>
          </div>
        </button>
        <div v-if="!searching && products.length === 0" class="empty-hint">
          搜索商品后选择规格，或点击「手动添加」录入
        </div>
      </div>
    </div>

    <el-table :data="lines" stripe class="mt-12" style="width: 100%">
      <el-table-column label="预览" width="72" align="center">
        <template #default="{ row }">
          <el-image
            v-if="row.pic"
            :src="row.pic"
            :preview-src-list="[row.pic]"
            preview-teleported
            fit="cover"
            class="line-pic"
          >
            <template #error>
              <div class="line-pic-fallback"><el-icon><Picture /></el-icon></div>
            </template>
          </el-image>
          <div v-else class="line-pic-fallback"><el-icon><Picture /></el-icon></div>
        </template>
      </el-table-column>
      <el-table-column label="商品" min-width="160">
        <template #default="{ row }">
          <el-input v-model="row.productName" placeholder="商品名称" />
        </template>
      </el-table-column>
      <el-table-column label="规格" min-width="110">
        <template #default="{ row }">
          <el-input v-model="row.specLabel" placeholder="规格" />
        </template>
      </el-table-column>
      <el-table-column label="SKU" min-width="100">
        <template #default="{ row }">
          <el-input v-model="row.skuCode" placeholder="SKU" />
        </template>
      </el-table-column>
      <el-table-column label="原价" width="108">
        <template #default="{ row }">
          <el-input-number
            v-model="row.originalPrice"
            :min="0"
            :precision="2"
            size="small"
            controls-position="right"
            @change="onDiscountChange(row)"
          />
        </template>
      </el-table-column>
      <el-table-column label="折扣" width="96">
        <template #default="{ row }">
          <el-input-number
            v-model="row.discount"
            :min="0.1"
            :max="10"
            :precision="2"
            :step="0.5"
            size="small"
            controls-position="right"
            @change="onDiscountChange(row)"
          />
        </template>
      </el-table-column>
      <el-table-column label="实付价" width="108">
        <template #default="{ row }">
          <el-input-number
            v-model="row.unitPrice"
            :min="0"
            :precision="2"
            size="small"
            controls-position="right"
            @change="onUnitPriceChange(row)"
          />
        </template>
      </el-table-column>
      <el-table-column label="数量" width="96">
        <template #default="{ row }">
          <el-input-number v-model="row.quantity" :min="1" size="small" />
        </template>
      </el-table-column>
      <el-table-column label="小计" width="90" align="right">
        <template #default="{ row }">
          ¥{{ (row.unitPrice * row.quantity).toFixed(2) }}
        </template>
      </el-table-column>
      <el-table-column width="52" fixed="right">
        <template #default="{ $index }">
          <el-button link type="danger" :icon="Delete" @click="removeLine($index)" />
        </template>
      </el-table-column>
    </el-table>
    <div class="totals">
      <span>原价合计 ¥{{ originalTotal.toFixed(2) }}</span>
      <span>应付合计 ¥{{ payableTotal.toFixed(2) }}</span>
    </div>

    <PosSkuSelectDialog
      v-model:visible="skuDialogVisible"
      :product="activeProduct"
      :skus="activeSkus"
      :loading="skuLoading"
      :require-store-stock="false"
      @select="addSku"
    />
  </div>
</template>

<style scoped>
.picker { border: 1px solid #ebeef5; border-radius: 8px; padding: 12px; background: #fafafa; }
.picker-bar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.search-input { flex: 1; min-width: 220px; }
.product-grid {
  margin-top: 12px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(132px, 1fr));
  gap: 10px;
  min-height: 48px;
  max-height: 280px;
  overflow: auto;
}
.product-card {
  border: 1px solid #ebeef5; border-radius: 8px; background: #fff;
  padding: 0; cursor: pointer; text-align: left; overflow: hidden;
}
.product-card:hover { border-color: #409eff; }
.pic-wrap { aspect-ratio: 1; background: #f5f7fa; }
.pic { width: 100%; height: 100%; }
.pic-fallback, .line-pic-fallback {
  width: 100%; height: 100%; display: flex; align-items: center; justify-content: center;
  color: #c0c4cc; background: #f5f7fa;
}
.line-pic { width: 44px; height: 44px; border-radius: 6px; }
.line-pic-fallback { width: 44px; height: 44px; border-radius: 6px; margin: 0 auto; }
.info { padding: 8px; }
.name {
  font-size: 13px; font-weight: 500; color: #303133; line-height: 1.3;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
}
.meta { margin-top: 4px; font-size: 12px; color: #909399; }
.hint { margin-top: 2px; font-size: 11px; color: #a0cfff; }
.empty-hint { grid-column: 1 / -1; text-align: center; color: #909399; padding: 16px 0; font-size: 13px; }
.mt-12 { margin-top: 12px; }
.totals {
  margin-top: 10px; display: flex; justify-content: flex-end; gap: 20px;
  font-size: 14px; color: #606266;
}
.totals span:last-child { font-weight: 600; color: #f56c6c; }
</style>
