<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listPosOrders, markPosPaid, type PosOrder } from '../../api/pos'

const loading = ref(false)
const list = ref<PosOrder[]>([])
const total = ref(0)
const page = ref(1)

async function load() {
  loading.value = true
  try {
    const data = await listPosOrders(undefined, page.value, 20)
    list.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}

async function pay(row: PosOrder) {
  await markPosPaid(row.id)
  await load()
}

onMounted(load)
</script>

<template>
  <el-card>
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="orderNo" label="单号" width="200" />
      <el-table-column prop="paymentMethod" label="支付方式" width="120" />
      <el-table-column prop="payStatus" label="支付状态" width="100" />
      <el-table-column prop="totalAmount" label="金额" width="100" />
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column label="操作" width="120">
        <template #default="{ row }">
          <el-button v-if="row.payStatus !== 'paid'" link type="primary" @click="pay(row)">确认收款</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>
