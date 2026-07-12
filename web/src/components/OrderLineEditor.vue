<script setup lang="ts">
import { computed } from 'vue'
import { Delete, Picture } from '@element-plus/icons-vue'
import PosProductCatalog from './PosProductCatalog.vue'
import { formatSpecLabel, resolvePic } from '../api/catalog'
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

const payableTotal = computed(() =>
  lines.value.reduce((sum, l) => sum + l.unitPrice * l.quantity, 0),
)
const originalTotal = computed(() =>
  lines.value.reduce((sum, l) => sum + (l.originalPrice || l.unitPrice) * l.quantity, 0),
)

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
    <div class="catalog-wrap">
      <PosProductCatalog
        :store-id="props.storeId"
        :require-store-stock="false"
        @select="addSku"
      />
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
          <div class="name">{{ row.productName }}</div>
          <div class="sub">{{ row.specLabel || '-' }} · {{ row.skuCode || '-' }}</div>
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
  </div>
</template>

<style scoped>
.catalog-wrap {
  height: 420px;
  min-height: 360px;
}
.catalog-wrap :deep(.pos-catalog) {
  height: 100%;
}
.mt-12 { margin-top: 12px; }
.line-pic { width: 44px; height: 44px; border-radius: 6px; }
.line-pic-fallback {
  width: 44px; height: 44px; border-radius: 6px; margin: 0 auto;
  display: flex; align-items: center; justify-content: center;
  color: #c0c4cc; background: #f5f7fa;
}
.name { font-size: 13px; font-weight: 500; color: #303133; }
.sub { margin-top: 2px; font-size: 12px; color: #909399; }
.totals {
  margin-top: 10px; display: flex; justify-content: flex-end; gap: 20px;
  font-size: 14px; color: #606266;
}
.totals span:last-child { font-weight: 600; color: #f56c6c; }
</style>
