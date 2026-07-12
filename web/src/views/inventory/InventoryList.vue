<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Picture } from '@element-plus/icons-vue'
import {
  listInventories,
  listInventoriesByStore,
  adjustInventory,
  type InventoryRow,
} from '../../api/inventory'
import {
  displaySpecValues,
  listBrands,
  listCategories,
  listGroups,
  resolvePic,
  type BrandItem,
  type CategoryItem,
  type GroupItem,
} from '../../api/catalog'
import PosProductCatalog from '../../components/PosProductCatalog.vue'
import type { ProductSkuSearchItem } from '../../api/productSku'
import { useStores } from '../../composables/useStores'

const router = useRouter()
const { stores, storeId } = useStores()
const loading = ref(false)
const list = ref<InventoryRow[]>([])
const keyword = ref('')
const brandId = ref<number | undefined>()
const categoryId = ref<number | undefined>()
const groupId = ref<number | undefined>()
const brands = ref<BrandItem[]>([])
const categories = ref<{ id: number; name: string }[]>([])
const groups = ref<GroupItem[]>([])
const dialogVisible = ref(false)
const saving = ref(false)
const adjustForm = reactive({
  skuId: 0,
  skuCode: '',
  productName: '',
  productPic: '',
  specLabel: '',
  pic: '',
  quantity: 0,
  safetyStock: 0,
  systemQty: 0,
})

function flattenCategories(tree: CategoryItem[], level = 0): { id: number; name: string }[] {
  const out: { id: number; name: string }[] = []
  for (const cat of tree) {
    if (cat.showStatus === 0) continue
    out.push({ id: cat.id, name: `${'　'.repeat(level)}${cat.name}` })
    if (cat.children?.length) {
      out.push(...flattenCategories(cat.children, level + 1))
    }
  }
  return out
}

async function loadFilters() {
  try {
    const [b, c, g] = await Promise.all([listBrands(), listCategories(), listGroups()])
    brands.value = b
    categories.value = flattenCategories(c)
    groups.value = g
  } catch (e) {
    ElMessage.error((e as Error).message || '加载筛选项失败')
  }
}

async function load() {
  loading.value = true
  try {
    const data = await listInventories(storeId.value, keyword.value, 1, 100, {
      brandId: brandId.value,
      categoryId: categoryId.value,
      groupId: groupId.value,
    })
    list.value = data.list
  } finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(adjustForm, {
    skuId: 0,
    skuCode: '',
    productName: '',
    productPic: '',
    specLabel: '',
    pic: '',
    quantity: 0,
    safetyStock: 0,
    systemQty: 0,
  })
}

function openStocktake(row?: InventoryRow) {
  if (row) {
    Object.assign(adjustForm, {
      skuId: row.skuId,
      skuCode: row.skuCode,
      productName: row.productName,
      productPic: '',
      specLabel: row.specLabel || '',
      pic: row.pic || '',
      quantity: row.quantity,
      safetyStock: row.safetyStock,
      systemQty: row.quantity,
    })
  } else {
    resetForm()
  }
  dialogVisible.value = true
}

async function pickSku(sku: ProductSkuSearchItem) {
  const pic = resolvePic(sku.pic, sku.productPic)
  let qty = sku.storeQty ?? 0
  let safety = 0
  if (storeId.value) {
    try {
      const rows = await listInventoriesByStore(storeId.value)
      const row = rows.find((r) => r.skuId === sku.skuId)
      if (row) {
        qty = row.quantity
        safety = row.safetyStock
      } else {
        qty = 0
        safety = 0
      }
    } catch {
      // 选品时已带 storeQty，拉取失败则沿用
    }
  }
  Object.assign(adjustForm, {
    skuId: sku.skuId,
    skuCode: sku.skuCode,
    productName: sku.productName,
    productPic: resolvePic(sku.productPic),
    specLabel: displaySpecValues(sku.specLabel, sku.specs),
    pic,
    quantity: qty,
    safetyStock: safety,
    systemQty: qty,
  })
}

function clearPicked() {
  resetForm()
}

async function submitStocktake() {
  if (!storeId.value || !adjustForm.skuId) {
    ElMessage.warning('请先选择要盘点的商品规格')
    return
  }
  saving.value = true
  try {
    await adjustInventory({
      storeId: storeId.value,
      skuId: adjustForm.skuId,
      skuCode: adjustForm.skuCode,
      productName: adjustForm.productName,
      specLabel: adjustForm.specLabel,
      pic: adjustForm.pic,
      quantity: adjustForm.quantity,
      safetyStock: adjustForm.safetyStock,
    })
    ElMessage.success('盘点已保存，门店库存已更新')
    dialogVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await loadFilters()
  await load()
})
</script>

<template>
  <el-card>
    <div class="toolbar">
      <el-select v-model="storeId" style="width: 160px" @change="load">
        <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
      </el-select>
      <el-select v-model="brandId" clearable placeholder="品牌" style="width: 140px" @change="load">
        <el-option v-for="b in brands" :key="b.id" :label="b.name" :value="b.id" />
      </el-select>
      <el-select v-model="categoryId" clearable placeholder="分类" style="width: 160px" @change="load">
        <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
      </el-select>
      <el-select v-model="groupId" clearable placeholder="分组" style="width: 140px" @change="load">
        <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
      </el-select>
      <el-input v-model="keyword" placeholder="搜索 SKU/商品" clearable style="width: 180px" @keyup.enter="load" />
      <el-button @click="load">查询</el-button>
      <el-button type="primary" @click="openStocktake()">库存盘点</el-button>
      <el-button type="warning" plain @click="router.push('/stock-transfers')">调货入库工单</el-button>
    </div>
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column label="规格" min-width="140">
        <template #default="{ row }">{{ displaySpecValues(row.specLabel) }}</template>
      </el-table-column>
      <el-table-column label="规格图片" width="88" align="center">
        <template #default="{ row }">
          <el-image
            v-if="resolvePic(row.pic)"
            :src="resolvePic(row.pic)"
            :preview-src-list="[resolvePic(row.pic)]"
            fit="cover"
            class="sku-thumb"
            preview-teleported
          >
            <template #error>
              <div class="thumb-empty"><el-icon><Picture /></el-icon></div>
            </template>
          </el-image>
          <div v-else class="thumb-empty"><el-icon><Picture /></el-icon></div>
        </template>
      </el-table-column>
      <el-table-column prop="skuCode" label="SKU" width="140" />
      <el-table-column prop="productName" label="商品" min-width="160" />
      <el-table-column prop="quantity" label="可用" width="80" />
      <el-table-column prop="safetyStock" label="安全库存" width="100" />
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button link type="primary" @click="openStocktake(row)">盘点</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog
    v-model="dialogVisible"
    title="库存盘点"
    width="960px"
    top="4vh"
    destroy-on-close
    class="stocktake-dialog"
  >
    <template v-if="!adjustForm.skuId">
      <div class="picker-tip">按分类浏览，或搜索商品/SKU；先选商品再选规格，可预览图片。实盘数量将同步当前门店库存。</div>
      <PosProductCatalog
        :store-id="storeId"
        :require-store-stock="false"
        stocktake-mode
        @select="pickSku"
      />
    </template>

    <div v-else class="stocktake-form">
      <div class="preview-row">
        <div class="preview-card">
          <div class="preview-label">商品图</div>
          <el-image
            v-if="resolvePic(adjustForm.productPic, adjustForm.pic)"
            :src="resolvePic(adjustForm.productPic, adjustForm.pic)"
            :preview-src-list="[resolvePic(adjustForm.productPic, adjustForm.pic)]"
            fit="cover"
            class="preview-img"
            preview-teleported
          >
            <template #error>
              <div class="preview-empty"><el-icon><Picture /></el-icon></div>
            </template>
          </el-image>
          <div v-else class="preview-empty"><el-icon><Picture /></el-icon></div>
        </div>
        <div class="preview-card">
          <div class="preview-label">规格图</div>
          <el-image
            v-if="resolvePic(adjustForm.pic)"
            :src="resolvePic(adjustForm.pic)"
            :preview-src-list="[resolvePic(adjustForm.pic)]"
            fit="cover"
            class="preview-img"
            preview-teleported
          >
            <template #error>
              <div class="preview-empty"><el-icon><Picture /></el-icon></div>
            </template>
          </el-image>
          <div v-else class="preview-empty"><el-icon><Picture /></el-icon></div>
        </div>
        <div class="preview-meta">
          <div class="meta-name">{{ adjustForm.productName }}</div>
          <div class="meta-line">规格：{{ displaySpecValues(adjustForm.specLabel) }}</div>
          <div class="meta-line">SKU：{{ adjustForm.skuCode || adjustForm.skuId }}</div>
          <div class="meta-line accent">系统门店库存：{{ adjustForm.systemQty }}</div>
          <el-button link type="primary" class="repick" @click="clearPicked">重新选择商品</el-button>
        </div>
      </div>

      <el-form label-width="110px" class="mt-16">
        <el-form-item label="实盘数量" required>
          <el-input-number v-model="adjustForm.quantity" :min="0" />
          <span class="field-hint">以盘点实数为准，写入门店库存</span>
        </el-form-item>
        <el-form-item label="安全库存">
          <el-input-number v-model="adjustForm.safetyStock" :min="0" />
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" :disabled="!adjustForm.skuId" @click="submitStocktake">
        保存盘点
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.sku-thumb { width: 48px; height: 48px; border-radius: 6px; }
.thumb-empty {
  width: 48px; height: 48px; border-radius: 6px; margin: 0 auto;
  display: flex; align-items: center; justify-content: center;
  background: #f5f7fa; color: #c0c4cc; font-size: 18px;
}
.picker-tip { color: #909399; font-size: 12px; margin-bottom: 10px; }
.stocktake-form { min-height: 200px; }
.preview-row {
  display: flex; gap: 16px; align-items: flex-start;
  padding: 12px; background: #f8fafc; border-radius: 10px;
}
.preview-card { width: 96px; flex-shrink: 0; }
.preview-label { font-size: 12px; color: #909399; margin-bottom: 6px; }
.preview-img {
  width: 96px; height: 96px; border-radius: 8px; display: block;
  background: #fff; border: 1px solid #ebeef5;
}
.preview-empty {
  width: 96px; height: 96px; border-radius: 8px;
  display: flex; align-items: center; justify-content: center;
  background: #fff; border: 1px solid #ebeef5; color: #c0c4cc; font-size: 28px;
}
.preview-meta { flex: 1; min-width: 0; padding-top: 4px; }
.meta-name { font-size: 16px; font-weight: 600; color: #303133; margin-bottom: 8px; }
.meta-line { font-size: 13px; color: #606266; line-height: 1.7; }
.meta-line.accent { color: #409eff; font-weight: 500; }
.repick { margin-top: 8px; padding: 0; }
.mt-16 { margin-top: 16px; }
.field-hint { margin-left: 12px; font-size: 12px; color: #909399; }
</style>
