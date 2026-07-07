<script setup lang="ts">
import { Delete, Plus } from '@element-plus/icons-vue'
import PosSkuPicker from './PosSkuPicker.vue'
import type { ProductSkuSearchItem } from '../api/productSku'

export interface OrderLine {
  skuId: number
  productName: string
  skuCode?: string
  specLabel?: string
  quantity: number
  unitPrice: number
}

const lines = defineModel<OrderLine[]>({ required: true })

function addSku(sku: ProductSkuSearchItem) {
  const existing = lines.value.find((l) => l.skuId === sku.skuId)
  if (existing) {
    existing.quantity += 1
    return
  }
  lines.value.push({
    skuId: sku.skuId,
    productName: sku.productName,
    skuCode: sku.skuCode,
    specLabel: sku.specLabel,
    quantity: 1,
    unitPrice: sku.price || 0,
  })
}

function addEmptyLine() {
  lines.value.push({ skuId: 0, productName: '', quantity: 1, unitPrice: 0 })
}

function removeLine(index: number) {
  lines.value.splice(index, 1)
}
</script>

<template>
  <div class="order-lines">
    <PosSkuPicker @select="addSku" />
    <el-table :data="lines" stripe class="mt-12">
      <el-table-column label="商品" min-width="180">
        <template #default="{ row }">
          <el-input v-model="row.productName" placeholder="商品名" />
        </template>
      </el-table-column>
      <el-table-column label="SKU" width="120">
        <template #default="{ row }">
          <el-input v-model="row.skuCode" placeholder="SKU" />
        </template>
      </el-table-column>
      <el-table-column label="规格" width="120">
        <template #default="{ row }">
          <el-input v-model="row.specLabel" />
        </template>
      </el-table-column>
      <el-table-column label="单价" width="110">
        <template #default="{ row }">
          <el-input-number v-model="row.unitPrice" :min="0" :precision="2" size="small" />
        </template>
      </el-table-column>
      <el-table-column label="数量" width="100">
        <template #default="{ row }">
          <el-input-number v-model="row.quantity" :min="1" size="small" />
        </template>
      </el-table-column>
      <el-table-column label="小计" width="90">
        <template #default="{ row }">
          ¥{{ (row.unitPrice * row.quantity).toFixed(2) }}
        </template>
      </el-table-column>
      <el-table-column width="60">
        <template #default="{ $index }">
          <el-button link type="danger" :icon="Delete" @click="removeLine($index)" />
        </template>
      </el-table-column>
    </el-table>
    <el-button class="mt-8" :icon="Plus" @click="addEmptyLine">手动添加行</el-button>
  </div>
</template>

<style scoped>
.mt-8 { margin-top: 8px; }
.mt-12 { margin-top: 12px; }
</style>
