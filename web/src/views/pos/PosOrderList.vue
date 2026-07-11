<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deletePosOrder, listPosOrders, markPosPaid, type PosOrder } from '../../api/pos'
import { useStores } from '../../composables/useStores'

const router = useRouter()
const { stores, storeId } = useStores()
const loading = ref(false)
const list = ref<PosOrder[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const payStatusFilter = ref('')

const payStatusMap: Record<string, string> = {
  unpaid: '未支付',
  paid: '已支付',
}
const statusMap: Record<string, string> = {
  pending: '待完成',
  completed: '已完成',
  preview: '预结算',
}
const paymentMap: Record<string, string> = {
  cash: '现金',
  static_qr: '静态二维码',
  wechat: '微信',
  alipay: '支付宝',
  card: '银行卡',
  mixed: '组合支付',
  preview: '预结算',
}

async function load() {
  loading.value = true
  try {
    const data = await listPosOrders(storeId.value, page.value, pageSize)
    let rows = data.list
    if (payStatusFilter.value) {
      rows = rows.filter((r) => r.payStatus === payStatusFilter.value)
    }
    list.value = rows
    total.value = data.total
  } finally {
    loading.value = false
  }
}

async function pay(row: PosOrder) {
  await markPosPaid(row.id)
  ElMessage.success('已确认收款')
  await load()
}

async function remove(row: PosOrder) {
  await ElMessageBox.confirm(`确认删除收银订单「${row.orderNo}」？删除后不可恢复。`, '删除确认', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消',
  })
  await deletePosOrder(row.id)
  ElMessage.success('已删除')
  await load()
}

function formatTime(v?: string) {
  if (!v) return '-'
  return v.replace('T', ' ').slice(0, 19)
}

onMounted(load)
</script>

<template>
  <el-card>
    <div class="toolbar">
      <el-select v-model="storeId" placeholder="门店" clearable style="width: 180px" @change="() => { page = 1; load() }">
        <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
      </el-select>
      <el-select v-model="payStatusFilter" clearable placeholder="支付状态" style="width: 140px" @change="load">
        <el-option v-for="(label, value) in payStatusMap" :key="value" :label="label" :value="value" />
      </el-select>
      <el-button @click="load">刷新</el-button>
      <el-button type="primary" @click="router.push('/pos')">去收银台</el-button>
    </div>

    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column prop="orderNo" label="单号" min-width="200" />
      <el-table-column label="支付方式" width="120">
        <template #default="{ row }">{{ paymentMap[row.paymentMethod] || row.paymentMethod }}</template>
      </el-table-column>
      <el-table-column label="支付状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.payStatus === 'paid' ? 'success' : 'warning'" size="small">
            {{ payStatusMap[row.payStatus] || row.payStatus }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="订单状态" width="100">
        <template #default="{ row }">
          <el-tag v-if="row.status === 'preview'" type="info" size="small" effect="plain">预结算</el-tag>
          <span v-else>{{ statusMap[row.status] || row.status }}</span>
        </template>
      </el-table-column>
      <el-table-column label="关联服务工单" min-width="180">
        <template #default="{ row }">
          <el-button
            v-if="row.serviceOrderId"
            link
            type="primary"
            @click="router.push(`/service-orders/${row.serviceOrderId}`)"
          >
            {{ row.serviceOrderNo || `#${row.serviceOrderId}` }}
          </el-button>
          <span v-else class="muted">-</span>
        </template>
      </el-table-column>
      <el-table-column label="金额" width="140">
        <template #default="{ row }">
          <div>¥{{ Number(row.totalAmount).toFixed(2) }}</div>
          <div
            v-if="row.discountAmount && Number(row.discountAmount) > 0"
            class="disc-hint"
          >
            原价 ¥{{ Number(row.originalAmount || 0).toFixed(2) }}
          </div>
        </template>
      </el-table-column>
      <el-table-column label="明细数" width="80">
        <template #default="{ row }">{{ row.items?.length || 0 }}</template>
      </el-table-column>
      <el-table-column label="时间" width="170">
        <template #default="{ row }">{{ formatTime(row.paidAt || row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/pos/orders/${row.id}`)">详情</el-button>
          <el-button
            v-if="row.payStatus !== 'paid' && row.status !== 'preview'"
            link
            type="success"
            @click="pay(row)"
          >
            确认收款
          </el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        background
        @current-change="load"
      />
    </div>
  </el-card>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
.disc-hint { font-size: 11px; color: #909399; }
.muted { color: #c0c4cc; }
</style>
