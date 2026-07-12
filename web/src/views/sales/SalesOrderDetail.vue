<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import PosReceiptPanel from '../../components/PosReceiptPanel.vue'
import {
  getSalesOrder,
  confirmSalesOrder,
  cancelSalesOrder,
  markSalesReady,
  shipSalesOrder,
  completeSalesOrder,
  scheduleSalesExpress,
  refreshSalesReceipt,
  createPurchaseFromSales,
  type SalesOrder,
} from '../../api/salesOrder'
import {
  deliveryTypeMap,
  fulfillStatusMap,
  fulfillmentMap,
  purchaseStatusMap,
  salesServiceStatusMap,
  salesStatusMap,
} from '../../composables/useStores'

const route = useRoute()
const router = useRouter()
const id = Number(route.params.id)
const loading = ref(false)
const order = ref<SalesOrder | null>(null)

const canEdit = computed(() => order.value && ['draft', 'preview'].includes(order.value.status))
const isPickupLike = computed(() => order.value && ['pickup', 'install'].includes(order.value.fulfillmentType))
const isShipLike = computed(() => order.value && ['delivery', 'express'].includes(order.value.fulfillmentType))

async function load() {
  loading.value = true
  try {
    order.value = await getSalesOrder(id)
  } finally {
    loading.value = false
  }
}

async function act(fn: () => Promise<unknown>, msg: string) {
  try {
    await fn()
    ElMessage.success(msg)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function createPO() {
  try {
    const po = await createPurchaseFromSales(id, { supplierName: '', items: [] })
    ElMessage.success('已生成采购单')
    router.push(`/purchase-orders/${(po as { id: number }).id}`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function doPreview() {
  await act(() => refreshSalesReceipt(id, true), '已刷新预结算单')
}

async function doScheduleExpress() {
  try {
    const { value } = await ElMessageBox.prompt('预约取件/寄出时间（可空）', '预约快递', {
      inputPlaceholder: '2026-07-12 15:00',
      confirmButtonText: '预约',
      cancelButtonText: '取消',
    }).catch(() => ({ value: null as string | null }))
    if (value === null) return
    await act(
      () => scheduleSalesExpress(id, { scheduledAt: value || undefined }),
      '已预约快递（发货中心对接预留）',
    )
  } catch (e) {
    if ((e as Error).message) ElMessage.error((e as Error).message)
  }
}

function fmtTime(v?: string) {
  if (!v) return '-'
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return v
  return d.toLocaleString()
}

onMounted(load)
</script>

<template>
  <div v-loading="loading">
    <el-page-header :icon="ArrowLeft" @back="router.push('/sales-orders')">
      <template #content>销售订单详情</template>
    </el-page-header>
    <el-card v-if="order" class="mt-16">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="单号">{{ order.orderNo }}</el-descriptions-item>
        <el-descriptions-item label="订单状态">{{ salesStatusMap[order.status] || order.status }}</el-descriptions-item>
        <el-descriptions-item label="履约方式">{{ fulfillmentMap[order.fulfillmentType] || order.fulfillmentType }}</el-descriptions-item>
        <el-descriptions-item label="采购状态">{{ purchaseStatusMap[order.purchaseStatus || 'none'] }}</el-descriptions-item>
        <el-descriptions-item v-if="order.fulfillmentType === 'install'" label="服务状态">
          {{ salesServiceStatusMap[order.serviceStatus || 'none'] }}
          <el-button
            v-if="order.serviceOrderId"
            link
            type="primary"
            @click="router.push(`/service-orders/${order.serviceOrderId}`)"
          >
            {{ order.serviceOrderNo || '查看工单' }}
          </el-button>
        </el-descriptions-item>
        <el-descriptions-item label="履约状态">{{ fulfillStatusMap[order.fulfillStatus || 'none'] }}</el-descriptions-item>
        <el-descriptions-item label="顾客">{{ order.customerName }} {{ order.customerPhone }}</el-descriptions-item>
        <el-descriptions-item label="金额">
          <span v-if="(order.discountAmount || 0) > 0">
            原价 ¥{{ (order.originalAmount || 0).toFixed(2) }} /
            优惠 ¥{{ (order.discountAmount || 0).toFixed(2) }} /
          </span>
          应付 ¥{{ order.totalAmount?.toFixed(2) }}
        </el-descriptions-item>
        <el-descriptions-item label="需采购">{{ order.needProcurement ? '是' : '否' }}</el-descriptions-item>
        <el-descriptions-item v-if="order.appointmentAt" label="预约时间">{{ fmtTime(order.appointmentAt) }}</el-descriptions-item>
        <el-descriptions-item v-if="order.pickupPersonName" label="取件人">
          {{ order.pickupPersonName }} {{ order.pickupPersonPhone }}
          <span v-if="order.pickupCode"> · 取件码 {{ order.pickupCode }}</span>
        </el-descriptions-item>
        <el-descriptions-item v-if="order.deliveryType" label="配送类型">
          {{ deliveryTypeMap[order.deliveryType] || order.deliveryType }}
        </el-descriptions-item>
        <el-descriptions-item v-if="order.expectedDeliveryAt" label="期望配送">{{ fmtTime(order.expectedDeliveryAt) }}</el-descriptions-item>
        <el-descriptions-item v-if="order.receiverName || order.shippingAddress" label="收货" :span="2">
          {{ order.receiverName }} {{ order.receiverPhone }}
          <div v-if="order.shippingAddress">{{ order.shippingAddress }}</div>
        </el-descriptions-item>
        <el-descriptions-item v-if="order.expressScheduledAt || order.expressCompany" label="快递">
          {{ order.expressCompany || '-' }}
          <span v-if="order.expressScheduledAt"> · 预约 {{ fmtTime(order.expressScheduledAt) }}</span>
          <span v-if="order.expressNo"> · 运单 {{ order.expressNo }}</span>
        </el-descriptions-item>
        <el-descriptions-item v-if="order.remark" label="备注" :span="2">{{ order.remark }}</el-descriptions-item>
      </el-descriptions>

      <el-table :data="order.items || []" stripe class="mt-16" style="width: 100%">
        <el-table-column label="预览" width="72" align="center">
          <template #default="{ row }">
            <el-image
              v-if="row.pic"
              :src="row.pic"
              :preview-src-list="[row.pic]"
              preview-teleported
              fit="cover"
              style="width: 44px; height: 44px; border-radius: 6px"
            />
            <span v-else class="muted">无图</span>
          </template>
        </el-table-column>
        <el-table-column prop="productName" label="商品" min-width="160" />
        <el-table-column prop="specLabel" label="规格" min-width="120" />
        <el-table-column prop="skuCode" label="SKU" min-width="110" />
        <el-table-column prop="quantity" label="数量" width="70" />
        <el-table-column label="原价" width="90">
          <template #default="{ row }">¥{{ (row.originalPrice ?? row.unitPrice)?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="折扣" width="70">
          <template #default="{ row }">{{ row.discount ?? 10 }}</template>
        </el-table-column>
        <el-table-column label="实付" width="90">
          <template #default="{ row }">¥{{ row.unitPrice?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="小计" width="90">
          <template #default="{ row }">¥{{ (row.totalAmount ?? row.unitPrice * row.quantity)?.toFixed(2) }}</template>
        </el-table-column>
      </el-table>

      <template v-if="order.serviceItems?.length">
        <h4 class="section-title">安装服务</h4>
        <el-table :data="order.serviceItems" stripe style="width: 100%">
          <el-table-column label="预览" width="72" align="center">
            <template #default="{ row }">
              <el-image
                v-if="row.pic"
                :src="row.pic"
                :preview-src-list="[row.pic]"
                preview-teleported
                fit="cover"
                style="width: 44px; height: 44px; border-radius: 6px"
              />
              <span v-else class="muted">无图</span>
            </template>
          </el-table-column>
          <el-table-column prop="serviceName" label="服务" min-width="140" />
          <el-table-column prop="quantity" label="数量" width="80" />
          <el-table-column label="原价" width="90">
            <template #default="{ row }">¥{{ ((row.originalPrice ?? row.unitPrice) || 0).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="折扣" width="70">
            <template #default="{ row }">{{ row.discount ?? 10 }}</template>
          </el-table-column>
          <el-table-column label="优惠价" width="90">
            <template #default="{ row }">¥{{ (row.unitPrice || 0).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="小计" width="90">
            <template #default="{ row }">¥{{ (row.totalAmount ?? (row.unitPrice || 0) * row.quantity).toFixed(2) }}</template>
          </el-table-column>
        </el-table>
      </template>

      <div class="actions">
        <el-button v-if="canEdit" @click="router.push(`/sales-orders/${id}/edit`)">编辑</el-button>
        <el-button type="warning" plain @click="doPreview">销售单预览</el-button>
        <el-button v-if="order.status === 'draft' || order.status === 'preview'" type="primary" @click="act(() => confirmSalesOrder(id), '已确认')">确认订单</el-button>
        <el-button
          v-if="order.status === 'confirmed' && isPickupLike"
          type="success"
          @click="act(() => markSalesReady(id), '已标记待提货')"
        >标记待提货</el-button>
        <el-button
          v-if="order.status === 'confirmed' && order.fulfillmentType === 'express'"
          type="warning"
          @click="doScheduleExpress"
        >预约快递</el-button>
        <el-button
          v-if="['confirmed','ready'].includes(order.status) && isShipLike"
          type="success"
          @click="act(() => shipSalesOrder(id), '已发货/配送中')"
        >发货/配送</el-button>
        <el-button
          v-if="(isPickupLike && order.status === 'ready') || (isShipLike && order.status === 'shipping')"
          type="primary"
          @click="act(() => completeSalesOrder(id), '已完成')"
        >完成</el-button>
        <el-button
          v-if="order.needProcurement && ['confirmed','ready','shipping'].includes(order.status)"
          type="warning"
          @click="createPO"
        >生成采购单</el-button>
        <el-button
          v-if="!['completed','cancelled'].includes(order.status)"
          type="danger"
          @click="act(() => cancelSalesOrder(id), '已取消')"
        >取消</el-button>
      </div>

      <div v-if="order.receiptHtml" class="receipt-wrap">
        <h4 class="section-title">销售单预览 / 下载</h4>
        <PosReceiptPanel
          :html="order.receiptHtml"
          :order-no="order.orderNo"
          title="销售单"
          variant="sales-doc"
        />
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.mt-16 { margin-top: 16px; }
.actions { margin-top: 16px; display: flex; flex-wrap: wrap; gap: 8px; }
.section-title { margin: 20px 0 10px; font-size: 15px; color: #303133; }
.receipt-wrap { margin-top: 8px; }
.muted { color: #c0c4cc; font-size: 12px; }
</style>
