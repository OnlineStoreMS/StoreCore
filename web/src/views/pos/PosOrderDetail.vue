<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getPosOrder, markPosPaid, type PosOrder } from '../../api/pos'
import PosReceiptPanel from '../../components/PosReceiptPanel.vue'
import { useStores } from '../../composables/useStores'

const route = useRoute()
const router = useRouter()
const { stores } = useStores()
const loading = ref(false)
const order = ref<PosOrder | null>(null)

const paymentMap: Record<string, string> = {
  cash: '现金',
  static_qr: '静态二维码',
  wechat: '微信',
  alipay: '支付宝',
  card: '银行卡',
  mixed: '组合支付',
}

const storeName = computed(() => {
  if (!order.value) return '-'
  return stores.value.find((s) => s.id === order.value!.storeId)?.name || `#${order.value.storeId}`
})

async function load() {
  const id = Number(route.params.id)
  if (!id) return
  loading.value = true
  try {
    order.value = await getPosOrder(id)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function pay() {
  if (!order.value) return
  await markPosPaid(order.value.id)
  ElMessage.success('已确认收款')
  await load()
}

function formatTime(v?: string) {
  if (!v) return '-'
  return v.replace('T', ' ').slice(0, 19)
}

onMounted(load)
</script>

<template>
  <div v-loading="loading">
    <div class="toolbar">
      <el-button @click="router.push('/pos/orders')">返回列表</el-button>
      <el-button v-if="order && order.payStatus !== 'paid'" type="primary" @click="pay">确认收款</el-button>
      <el-button @click="router.push('/pos')">去收银台</el-button>
    </div>

    <el-row v-if="order" :gutter="16">
      <el-col :span="14">
        <el-card>
          <template #header>
            <div class="card-head">
              <span>收银订单 {{ order.orderNo }}</span>
              <el-tag :type="order.payStatus === 'paid' ? 'success' : 'warning'" size="small">
                {{ order.payStatus === 'paid' ? '已支付' : '未支付' }}
              </el-tag>
            </div>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="门店">{{ storeName }}</el-descriptions-item>
            <el-descriptions-item label="支付方式">
              {{ paymentMap[order.paymentMethod] || order.paymentMethod }}
            </el-descriptions-item>
            <el-descriptions-item label="订单状态">{{ order.status }}</el-descriptions-item>
            <el-descriptions-item label="合计金额">
              <strong class="amount">¥{{ Number(order.totalAmount).toFixed(2) }}</strong>
            </el-descriptions-item>
            <el-descriptions-item label="实收">¥{{ Number(order.paidAmount).toFixed(2) }}</el-descriptions-item>
            <el-descriptions-item label="时间">{{ formatTime(order.paidAt || order.createdAt) }}</el-descriptions-item>
            <el-descriptions-item v-if="order.customerName" label="顾客">{{ order.customerName }}</el-descriptions-item>
            <el-descriptions-item v-if="order.customerPhone" label="电话">{{ order.customerPhone }}</el-descriptions-item>
          </el-descriptions>

          <h4 class="section-title">明细</h4>
          <el-table :data="order.items || []" stripe>
            <el-table-column label="类型" width="80">
              <template #default="{ row }">
                <el-tag size="small" :type="row.itemType === 'service' ? 'warning' : 'primary'" effect="plain">
                  {{ row.itemType === 'service' ? '服务' : '商品' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="图片" width="70">
              <template #default="{ row }">
                <el-image v-if="row.pic" :src="row.pic" style="width: 40px; height: 40px" fit="cover" />
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column prop="productName" label="名称" min-width="160" />
            <el-table-column prop="specLabel" label="规格/说明" width="140" />
            <el-table-column prop="skuCode" label="编码" width="110" />
            <el-table-column label="单价" width="90">
              <template #default="{ row }">¥{{ Number(row.unitPrice).toFixed(2) }}</template>
            </el-table-column>
            <el-table-column prop="quantity" label="数量" width="70" />
            <el-table-column label="小计" width="100">
              <template #default="{ row }">
                ¥{{ Number(row.totalAmount ?? row.unitPrice * row.quantity).toFixed(2) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="10">
        <el-card>
          <template #header>电子小票</template>
          <PosReceiptPanel v-if="order.receiptHtml" :html="order.receiptHtml" :order-no="order.orderNo" />
          <el-empty v-else description="尚未生成小票（未支付）" />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; }
.card-head { display: flex; justify-content: space-between; align-items: center; }
.amount { color: #f56c6c; font-size: 16px; }
.section-title { margin: 20px 0 12px; font-size: 15px; }
</style>
