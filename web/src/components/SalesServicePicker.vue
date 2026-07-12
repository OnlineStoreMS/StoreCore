<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Delete, Picture, Plus, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { listServiceItems, type ServiceItem } from '../api/serviceCatalog'
import type { SalesServiceLine } from '../api/salesOrder'

const lines = defineModel<SalesServiceLine[]>({ required: true })
const keyword = ref('')
const loading = ref(false)
const catalog = ref<ServiceItem[]>([])

const payableTotal = computed(() =>
  lines.value.reduce((sum, l) => sum + (l.unitPrice || 0) * l.quantity, 0),
)
const originalTotal = computed(() =>
  lines.value.reduce((sum, l) => sum + (l.originalPrice || l.unitPrice || 0) * l.quantity, 0),
)

async function search() {
  loading.value = true
  try {
    const data = await listServiceItems({
      keyword: keyword.value.trim() || undefined,
      page: 1,
      pageSize: 50,
      status: 1,
    })
    catalog.value = data.list || []
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

function addItem(it: ServiceItem) {
  const existing = lines.value.find((l) => l.serviceItemId > 0 && l.serviceItemId === it.id)
  if (existing) {
    existing.quantity += 1
    return
  }
  const price = it.price || 0
  lines.value.push({
    serviceItemId: it.id,
    serviceName: it.name,
    serviceCode: it.code,
    quantity: 1,
    originalPrice: price,
    discount: 10,
    unitPrice: price,
    durationMin: it.durationMin || 0,
    pic: it.pic,
  })
}

function addManualLine() {
  lines.value.push({
    serviceItemId: 0,
    serviceName: '',
    serviceCode: '',
    quantity: 1,
    originalPrice: 0,
    discount: 10,
    unitPrice: 0,
    durationMin: 0,
    pic: '',
  })
}

function onDiscountChange(row: SalesServiceLine) {
  let d = Number(row.discount)
  if (!Number.isFinite(d) || d <= 0) d = 10
  if (d > 10) d = 10
  row.discount = Math.round(d * 100) / 100
  const orig = (row.originalPrice || 0) > 0 ? row.originalPrice! : row.unitPrice || 0
  row.originalPrice = orig
  row.unitPrice = Math.round(orig * (row.discount! / 10) * 100) / 100
}

function onUnitPriceChange(row: SalesServiceLine) {
  const orig = (row.originalPrice || 0) > 0 ? row.originalPrice! : row.unitPrice || 0
  row.originalPrice = orig
  if (orig > 0) {
    row.discount = Math.round(((row.unitPrice || 0) / orig) * 10 * 100) / 100
  } else {
    row.discount = 10
  }
}

function removeLine(index: number) {
  lines.value.splice(index, 1)
}

onMounted(search)
</script>

<template>
  <div class="service-picker">
    <div class="picker-bar">
      <el-input
        v-model="keyword"
        clearable
        placeholder="搜索服务项目"
        class="search-input"
        @keyup.enter="search"
      >
        <template #append>
          <el-button :icon="Search" :loading="loading" @click="search">搜索</el-button>
        </template>
      </el-input>
      <el-button :icon="Plus" @click="addManualLine">手动添加</el-button>
    </div>

    <div v-loading="loading" class="service-grid">
      <button
        v-for="it in catalog"
        :key="it.id"
        type="button"
        class="service-card"
        @click="addItem(it)"
      >
        <div class="pic-wrap">
          <el-image
            v-if="it.pic"
            :src="it.pic"
            :preview-src-list="[it.pic]"
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
          <div class="name">{{ it.name }}</div>
          <div class="meta">¥{{ (it.price || 0).toFixed(2) }}</div>
        </div>
      </button>
      <div v-if="!loading && catalog.length === 0" class="empty-hint">暂无服务，可手动添加</div>
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
      <el-table-column label="服务名称" min-width="150">
        <template #default="{ row }">
          <el-input v-model="row.serviceName" placeholder="服务名称" />
        </template>
      </el-table-column>
      <el-table-column label="编码" min-width="100">
        <template #default="{ row }">
          <el-input v-model="row.serviceCode" placeholder="编码" />
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
      <el-table-column label="优惠价" width="108">
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
          ¥{{ ((row.unitPrice || 0) * row.quantity).toFixed(2) }}
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
.picker-bar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 10px; }
.search-input { flex: 1; min-width: 220px; }
.service-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(132px, 1fr));
  gap: 10px;
  max-height: 220px;
  overflow: auto;
  padding: 10px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #fafafa;
}
.service-card {
  border: 1px solid #ebeef5; border-radius: 8px; background: #fff;
  padding: 0; cursor: pointer; text-align: left; overflow: hidden;
}
.service-card:hover { border-color: #409eff; }
.pic-wrap { aspect-ratio: 1.2; background: #f5f7fa; }
.pic { width: 100%; height: 100%; }
.pic-fallback, .line-pic-fallback {
  width: 100%; height: 100%; display: flex; align-items: center; justify-content: center;
  color: #c0c4cc; background: #f5f7fa;
}
.line-pic { width: 44px; height: 44px; border-radius: 6px; }
.line-pic-fallback { width: 44px; height: 44px; border-radius: 6px; margin: 0 auto; }
.info { padding: 8px; }
.name {
  font-size: 13px; font-weight: 500; color: #303133;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
}
.meta { margin-top: 4px; font-size: 12px; color: #f56c6c; font-weight: 600; }
.empty-hint { grid-column: 1 / -1; text-align: center; color: #909399; padding: 12px 0; }
.mt-12 { margin-top: 12px; }
.totals {
  margin-top: 10px; display: flex; justify-content: flex-end; gap: 20px;
  font-size: 14px; color: #606266;
}
.totals span:last-child { font-weight: 600; color: #f56c6c; }
</style>
