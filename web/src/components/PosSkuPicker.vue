<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { searchProductSkus, formatSkuOptionLabel, type ProductSkuSearchItem } from '../api/productSku'

const emit = defineEmits<{
  select: [sku: ProductSkuSearchItem]
}>()

const keyword = ref('')
const loading = ref(false)
const options = ref<ProductSkuSearchItem[]>([])

async function search() {
  const q = keyword.value.trim()
  if (!q) return
  loading.value = true
  try {
    const data = await searchProductSkus({ keyword: q, page: 1, pageSize: 20 })
    options.value = data.list
  } catch (e) {
    ElMessage.error((e as Error).message || '搜索失败')
  } finally {
    loading.value = false
  }
}

function pick(sku: ProductSkuSearchItem) {
  emit('select', sku)
  keyword.value = ''
  options.value = []
}
</script>

<template>
  <div class="sku-picker">
    <div class="search-row">
      <el-input
        v-model="keyword"
        placeholder="搜索 ProductCore 商品/SKU（编码、名称、规格）"
        clearable
        @keyup.enter="search"
      />
      <el-button type="primary" :loading="loading" @click="search">搜索</el-button>
    </div>
    <div v-if="options.length" class="results">
      <el-button
        v-for="item in options"
        :key="item.skuId"
        class="result-btn"
        @click="pick(item)"
      >
        {{ formatSkuOptionLabel(item) }} · ¥{{ item.price?.toFixed(2) ?? '0.00' }}
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.search-row { display: flex; gap: 8px; }
.results { margin-top: 12px; display: flex; flex-wrap: wrap; gap: 8px; }
.result-btn { margin: 0; }
</style>
