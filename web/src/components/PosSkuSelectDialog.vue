<script setup lang="ts">
import { computed } from 'vue'
import { Close, Picture } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import {
  formatSpecLabel,
  resolvePic,
  type CatalogProduct,
  type CatalogSku,
} from '../api/catalog'
import type { ProductSkuSearchItem } from '../api/productSku'

type SkuWithStore = CatalogSku & { storeQty?: number }

const props = defineProps<{
  visible: boolean
  product: CatalogProduct | null
  skus: SkuWithStore[]
  loading?: boolean
  /** 销售单等场景允许无门店库存仍可选 */
  requireStoreStock?: boolean
  /** 库存盘点：只展示门店库存，不展示统一商品库库存 */
  stocktakeMode?: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  select: [sku: ProductSkuSearchItem]
}>()

const productPic = computed(() => resolvePic(props.product?.pic))
const blockEmptyStore = computed(() => !props.stocktakeMode && props.requireStoreStock !== false)
const stocktakeMode = computed(() => !!props.stocktakeMode)

function close() {
  emit('update:visible', false)
}

function pick(sku: SkuWithStore) {
  if (!props.product) return
  if (blockEmptyStore.value && (sku.storeQty ?? 0) <= 0) {
    ElMessage.warning('门店无货或库存为 0，需仓库调货')
    return
  }
  emit('select', {
    productId: props.product.id,
    productName: props.product.name,
    productPic: productPic.value,
    skuId: sku.id,
    skuCode: sku.skuCode,
    specs: sku.specs || {},
    specLabel: formatSpecLabel(sku.specs),
    price: sku.price || props.product.price,
    stock: sku.stock,
    pic: resolvePic(sku.pic, props.product.pic),
    storeQty: sku.storeQty ?? 0,
  })
  close()
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="product ? `选择规格 — ${product.name}` : '选择规格'"
    width="720px"
    class="pos-sku-dialog"
    destroy-on-close
    append-to-body
    @update:model-value="emit('update:visible', $event)"
  >
    <div v-loading="loading" class="dialog-body">
      <div v-if="product" class="product-brief">
        <div class="product-thumb">
          <el-image v-if="productPic" :src="productPic" fit="cover" class="thumb-img">
            <template #error>
              <div class="thumb-placeholder"><el-icon><Picture /></el-icon></div>
            </template>
          </el-image>
          <div v-else class="thumb-placeholder"><el-icon><Picture /></el-icon></div>
        </div>
        <div>
          <div class="product-name">{{ product.name }}</div>
          <div class="product-meta">
            共 {{ skus.length }} 个 SKU
            <template v-if="stocktakeMode"> · 显示当前门店库存</template>
            <template v-else-if="blockEmptyStore"> · 灰色表示门店无货</template>
            <template v-else> · 灰色表示门店无货（仍可选）</template>
          </div>
        </div>
      </div>

      <div v-if="!loading && skus.length === 0" class="empty">暂无可用 SKU</div>

      <div v-else class="sku-grid">
        <button
          v-for="sku in skus"
          :key="sku.id"
          type="button"
          class="sku-card"
          :class="{ disabled: blockEmptyStore && (sku.storeQty ?? 0) <= 0 }"
          @click="pick(sku)"
        >
          <div class="sku-pic-wrap">
            <el-image
              :src="resolvePic(sku.pic, product?.pic)"
              fit="cover"
              class="sku-pic"
            >
              <template #error>
                <div class="sku-pic-fallback"><el-icon><Picture /></el-icon></div>
              </template>
            </el-image>
            <div v-if="blockEmptyStore && (sku.storeQty ?? 0) <= 0" class="sku-badge">门店无货</div>
            <div v-else-if="!stocktakeMode && !blockEmptyStore && (sku.storeQty ?? 0) <= 0" class="sku-badge soft">门店无货·可采购</div>
          </div>
          <div class="sku-info">
            <div class="sku-spec">{{ formatSpecLabel(sku.specs) }}</div>
            <div v-if="sku.skuCode" class="sku-code">{{ sku.skuCode }}</div>
            <div v-if="!stocktakeMode" class="sku-price">¥{{ (sku.price || product?.price || 0).toFixed(2) }}</div>
            <div class="sku-stock">
              <template v-if="stocktakeMode">门店库存 {{ sku.storeQty ?? 0 }}</template>
              <template v-else>仓库 {{ sku.stock ?? 0 }} · 门店 {{ sku.storeQty ?? 0 }}</template>
            </div>
          </div>
        </button>
      </div>
    </div>

    <template #footer>
      <el-button :icon="Close" @click="close">取消</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.dialog-body { min-height: 200px; }
.product-brief {
  display: flex; gap: 12px; align-items: center;
  margin-bottom: 16px; padding-bottom: 16px; border-bottom: 1px solid #ebeef5;
}
.product-thumb {
  width: 64px; height: 64px; border-radius: 8px; overflow: hidden;
  background: #f5f7fa; flex-shrink: 0;
}
.thumb-img { width: 100%; height: 100%; }
.thumb-placeholder,
.sku-pic-fallback {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
  color: #c0c4cc; font-size: 24px; background: #f5f7fa;
}
.product-name { font-size: 16px; font-weight: 600; color: #303133; }
.product-meta { margin-top: 4px; font-size: 13px; color: #909399; }
.empty { text-align: center; color: #909399; padding: 48px 0; }
.sku-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}
.sku-card {
  border: 1px solid #ebeef5;
  border-radius: 10px;
  background: #fff;
  padding: 0;
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
  overflow: hidden;
}
.sku-card:hover:not(.disabled) {
  border-color: #409eff;
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.15);
  transform: translateY(-2px);
}
.sku-card.disabled {
  opacity: 0.55;
  cursor: not-allowed;
  filter: grayscale(0.3);
}
.sku-pic-wrap { aspect-ratio: 1; background: #fafafa; position: relative; }
.sku-pic { width: 100%; height: 100%; }
.sku-badge {
  position: absolute; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.62); color: #fff;
  font-size: 11px; text-align: center; padding: 3px 0;
}
.sku-badge.soft { background: rgba(230, 162, 60, 0.9); }
.sku-info { padding: 8px 10px 10px; }
.sku-spec {
  font-size: 13px; font-weight: 500; color: #303133; line-height: 1.3;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
}
.sku-code { margin-top: 2px; font-size: 11px; color: #909399; }
.sku-price { margin-top: 6px; font-size: 15px; font-weight: 700; color: #f56c6c; }
.sku-stock { margin-top: 2px; font-size: 11px; color: #909399; }
</style>
