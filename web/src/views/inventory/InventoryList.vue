<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Picture } from '@element-plus/icons-vue'
import { listInventories, adjustInventory, type InventoryRow } from '../../api/inventory'
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
import PosSkuPicker from '../../components/PosSkuPicker.vue'
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
const adjustForm = reactive({
  skuId: 0,
  skuCode: '',
  productName: '',
  specLabel: '',
  pic: '',
  quantity: 0,
  safetyStock: 0,
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

function openAdjust(row?: InventoryRow) {
  if (row) {
    Object.assign(adjustForm, {
      skuId: row.skuId,
      skuCode: row.skuCode,
      productName: row.productName,
      specLabel: row.specLabel || '',
      pic: row.pic || '',
      quantity: row.quantity,
      safetyStock: row.safetyStock,
    })
  } else {
    Object.assign(adjustForm, {
      skuId: 0,
      skuCode: '',
      productName: '',
      specLabel: '',
      pic: '',
      quantity: 0,
      safetyStock: 0,
    })
  }
  dialogVisible.value = true
}

function pickSku(sku: ProductSkuSearchItem) {
  adjustForm.skuId = sku.skuId
  adjustForm.skuCode = sku.skuCode
  adjustForm.productName = sku.productName
  adjustForm.specLabel = displaySpecValues(sku.specLabel, sku.specs)
  adjustForm.pic = resolvePic(sku.pic, sku.productPic)
}

async function submitAdjust() {
  if (!storeId.value || !adjustForm.skuId) {
    ElMessage.warning('请选择 SKU')
    return
  }
  try {
    await adjustInventory({ storeId: storeId.value, ...adjustForm })
    ElMessage.success('库存已更新')
    dialogVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
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
      <el-button type="primary" @click="openAdjust()">盘点调整</el-button>
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
          <el-button link type="primary" @click="openAdjust(row)">调整</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="dialogVisible" title="库存调整" width="520px">
    <PosSkuPicker v-if="!adjustForm.skuId" @select="pickSku" />
    <el-form v-else label-width="90px" class="mt-8">
      <el-form-item label="商品">{{ adjustForm.productName }} ({{ adjustForm.skuCode }})</el-form-item>
      <el-form-item label="规格">{{ displaySpecValues(adjustForm.specLabel) }}</el-form-item>
      <el-form-item label="当前数量"><el-input-number v-model="adjustForm.quantity" :min="0" /></el-form-item>
      <el-form-item label="安全库存"><el-input-number v-model="adjustForm.safetyStock" :min="0" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submitAdjust">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.mt-8 { margin-top: 8px; }
.sku-thumb { width: 48px; height: 48px; border-radius: 6px; }
.thumb-empty {
  width: 48px; height: 48px; border-radius: 6px; margin: 0 auto;
  display: flex; align-items: center; justify-content: center;
  background: #f5f7fa; color: #c0c4cc; font-size: 18px;
}
</style>
