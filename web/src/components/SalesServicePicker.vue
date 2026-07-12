<script setup lang="ts">
import { computed, ref } from 'vue'
import { Delete, Search, Tools } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { listServiceItems, type ServiceItem } from '../api/serviceCatalog'
import type { SalesServiceLine } from '../api/salesOrder'

const lines = defineModel<SalesServiceLine[]>({ required: true })
const keyword = ref('')
const loading = ref(false)
const searched = ref(false)
const catalog = ref<ServiceItem[]>([])

const payableTotal = computed(() =>
  lines.value.reduce((sum, l) => sum + (l.unitPrice || 0) * l.quantity, 0),
)
const originalTotal = computed(() =>
  lines.value.reduce((sum, l) => sum + (l.originalPrice || l.unitPrice || 0) * l.quantity, 0),
)

async function search() {
  const q = keyword.value.trim()
  if (!q) {
    catalog.value = []
    searched.value = false
    ElMessage.info('请输入关键词搜索服务')
    return
  }
  loading.value = true
  searched.value = true
  try {
    const data = await listServiceItems({
      keyword: q,
      page: 1,
      pageSize: 48,
      status: 1,
    })
    catalog.value = data.list || []
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

function clearSearch() {
  keyword.value = ''
  catalog.value = []
  searched.value = false
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
</script>

<template>
  <div class="service-picker">
    <div class="toolbar">
      <el-input
        v-model="keyword"
        clearable
        placeholder="搜索服务名称、编码"
        class="search-input"
        @keyup.enter="search"
        @clear="clearSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-button type="primary" :loading="loading" @click="search">搜索</el-button>
    </div>
    <div class="mode-hint">输入关键词搜索服务目录，点击卡片加入订单（确认后生成服务工单）</div>

    <div v-loading="loading" class="result-wrap">
      <div v-if="!searched && !loading" class="grid-empty">请输入关键词搜索服务项目</div>
      <div v-else-if="searched && !loading && catalog.length === 0" class="grid-empty">未找到匹配服务</div>
      <div v-else class="service-grid">
        <button
          v-for="it in catalog"
          :key="it.id"
          type="button"
          class="service-card"
          @click="addItem(it)"
        >
          <div class="card-pic">
            <el-image v-if="it.pic" :src="it.pic" fit="cover" class="card-img" lazy>
              <template #error>
                <div class="pic-fallback"><el-icon><Tools /></el-icon></div>
              </template>
            </el-image>
            <div v-else class="pic-fallback"><el-icon><Tools /></el-icon></div>
          </div>
          <div class="card-body">
            <div class="card-title">{{ it.name }}</div>
            <div class="card-sub">
              {{ it.categoryName || '服务' }}
              <template v-if="it.durationMin"> · {{ it.durationMin }} 分钟</template>
            </div>
            <div class="card-footer">
              <span class="card-price">¥{{ Number(it.price || 0).toFixed(2) }}</span>
              <el-tag size="small" type="warning" effect="plain">服务</el-tag>
            </div>
          </div>
        </button>
      </div>
    </div>

    <el-table :data="lines" stripe class="mt-12" style="width: 100%">
      <el-table-column label="图标" width="72" align="center">
        <template #default="{ row }">
          <div class="line-icon">
            <el-image v-if="row.pic" :src="row.pic" fit="cover" class="line-pic">
              <template #error>
                <div class="line-pic-fallback"><el-icon><Tools /></el-icon></div>
              </template>
            </el-image>
            <div v-else class="line-pic-fallback"><el-icon><Tools /></el-icon></div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="服务名称" min-width="150">
        <template #default="{ row }">
          <div>{{ row.serviceName }}</div>
          <div class="sub">{{ row.serviceCode || '-' }}</div>
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
.toolbar { display: flex; gap: 8px; align-items: center; margin-bottom: 8px; }
.search-input { flex: 1; min-width: 220px; }
.mode-hint { font-size: 12px; color: #909399; margin-bottom: 10px; }
.result-wrap {
  min-height: 160px;
  max-height: 280px;
  overflow: auto;
  border: 1px solid #ebeef5;
  border-radius: 10px;
  background: #fff;
  padding: 12px;
}
.grid-empty {
  display: flex; align-items: center; justify-content: center;
  min-height: 140px; color: #909399; font-size: 13px;
}
.service-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(148px, 1fr));
  gap: 12px;
}
.service-card {
  border: 1px solid #ebeef5;
  border-radius: 10px;
  background: #fff;
  padding: 0;
  cursor: pointer;
  text-align: left;
  overflow: hidden;
  transition: border-color 0.15s, box-shadow 0.15s, transform 0.12s;
}
.service-card:hover {
  border-color: #e6a23c;
  box-shadow: 0 6px 16px rgba(230, 162, 60, 0.15);
  transform: translateY(-2px);
}
.card-pic { aspect-ratio: 1; background: #fff7e6; }
.card-img { width: 100%; height: 100%; }
.pic-fallback {
  width: 100%; height: 100%; min-height: 100px;
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
.mt-12 { margin-top: 12px; }
.line-icon { display: flex; justify-content: center; }
.line-pic { width: 44px; height: 44px; border-radius: 6px; }
.line-pic-fallback {
  width: 44px; height: 44px; border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  color: #e6a23c; background: #fff7e6; font-size: 20px;
}
.sub { margin-top: 2px; font-size: 12px; color: #909399; }
.totals {
  margin-top: 10px; display: flex; justify-content: flex-end; gap: 20px;
  font-size: 14px; color: #606266;
}
.totals span:last-child { font-weight: 600; color: #f56c6c; }
</style>
