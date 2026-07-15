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
  deleteSalesOrder,
  markSalesPaid,
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
  salesPayStatusMap,
  salesServiceStatusMap,
  salesStatusMap,
} from '../../composables/useStores'

const route = useRoute()
const router = useRouter()
const id = Number(route.params.id)
const loading = ref(false)
const order = ref<SalesOrder | null>(null)

const canEdit = computed(() => order.value && ['draft', 'preview'].includes(order.value.status))
const canDelete = computed(() => order.value && ['draft', 'preview', 'cancelled'].includes(order.value.status))
const isPickupLike = computed(() => order.value && ['pickup', 'install'].includes(order.value.fulfillmentType))
const isShipLike = computed(() => order.value && ['delivery', 'express'].includes(order.value.fulfillmentType))
const canMarkPaid = computed(() =>
  order.value
  && order.value.payStatus !== 'paid'
  && ['confirmed', 'ready', 'shipping'].includes(order.value.status),
)

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

async function doConfirm() {
  if (!order.value) return
  if (['pickup', 'install'].includes(order.value.fulfillmentType) && !order.value.appointmentAt) {
    ElMessage.warning('请先编辑订单并填写预约时间后再确认')
    return
  }
  if (order.value.fulfillmentType === 'install' && !(order.value.serviceItems?.length)) {
    ElMessage.warning('到店安装请先选择服务项目')
    return
  }
  if (['delivery', 'express'].includes(order.value.fulfillmentType) && !order.value.shippingAddress?.trim()) {
    ElMessage.warning('请先编辑订单并填写收货地址后再确认')
    return
  }
  await act(() => confirmSalesOrder(id), '已确认（系统已自动判断库存补货方式）')
}

async function doMarkPaid() {
  await act(() => markSalesPaid(id), '已付款（需采购时将自动生成采购草稿）')
}

async function doComplete() {
  if (!order.value) return
  if (order.value.serviceOrderId || order.value.fulfillmentType === 'install') {
    if (order.value.serviceStatus !== 'completed') {
      ElMessage.warning('服务工单未完成，无法标记已提货')
      return
    }
  }
  if (order.value.needProcurement || order.value.purchaseOrderId || (order.value.purchaseStatus && order.value.purchaseStatus !== 'none')) {
    if (order.value.purchaseStatus !== 'received') {
      ElMessage.warning('采购订单未到货，无法标记已提货')
      return
    }
  }
  const msg = isPickupLike.value ? '已标记已提货' : '已完成'
  await act(() => completeSalesOrder(id), msg)
}

async function createPO() {
  try {
    const po = await createPurchaseFromSales(id, { supplierName: '', items: [] })
    ElMessage.success('已关联采购单草稿')
    router.push(`/purchase-orders/${(po as { id: number }).id}`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function doPreview() {
  const isDraftLike = order.value && ['draft', 'preview'].includes(order.value.status)
  await act(
    () => refreshSalesReceipt(id, !!isDraftLike),
    isDraftLike ? '已刷新预结算单' : '已刷新销售单',
  )
}

async function remove() {
  try {
    await ElMessageBox.confirm(`确认删除销售单「${order.value?.orderNo}」？删除后不可恢复。`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
    await deleteSalesOrder(id)
    ElMessage.success('已删除')
    router.push('/sales-orders')
  } catch (e) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error((e as Error).message || '删除失败')
  }
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
        <el-descriptions-item label="付款状态">
          <el-tag :type="order.payStatus === 'paid' ? 'success' : 'warning'" size="small">
            {{ salesPayStatusMap[order.payStatus] || order.payStatus }}
          </el-tag>
          <span v-if="order.paidAt" class="muted pad-l">{{ fmtTime(order.paidAt) }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="采购状态">
          {{ purchaseStatusMap[order.purchaseStatus || 'none'] }}
          <el-button
            v-if="order.purchaseOrderId"
            link
            type="primary"
            @click="router.push(`/purchase-orders/${order.purchaseOrderId}`)"
          >查看采购单</el-button>
        </el-descriptions-item>
        <el-descriptions-item label="系统判断">
          <el-tag v-if="order.needProcurement" type="warning" size="small">需采购</el-tag>
          <el-tag v-if="order.stockTransferOrderId" type="info" size="small" class="pad-l">已创建调货</el-tag>
          <span v-if="!order.needProcurement && !order.stockTransferOrderId" class="muted">门店库存可覆盖</span>
        </el-descriptions-item>
        <el-descriptions-item v-if="order.stockTransferOrderId" label="调货入库单">
          <el-button link type="primary" @click="router.push('/stock-transfers')">
            #{{ order.stockTransferOrderId }}（调货入库）
          </el-button>
        </el-descriptions-item>
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
        <el-descriptions-item label="履约状态">
          {{ fulfillStatusMap[order.fulfillStatus || 'none'] }}
          <span class="hint-inline">（提货/配送进度，非付款）</span>
        </el-descriptions-item>
        <el-descriptions-item label="顾客">{{ order.customerName }} {{ order.customerPhone }}</el-descriptions-item>
        <el-descriptions-item label="金额">
          <span v-if="(order.discountAmount || 0) > 0">
            原价 ¥{{ (order.originalAmount || 0).toFixed(2) }} /
            优惠 ¥{{ (order.discountAmount || 0).toFixed(2) }} /
          </span>
          应付 ¥{{ order.totalAmount?.toFixed(2) }}
        </el-descriptions-item>
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
        <h4 class="section-title">服务目录</h4>
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
        <el-button type="warning" plain @click="doPreview">
          {{ ['draft', 'preview'].includes(order.status) ? '刷新预结算单' : '刷新销售单' }}
        </el-button>
        <el-button v-if="order.status === 'draft' || order.status === 'preview'" type="primary" @click="doConfirm">确认订单</el-button>
        <el-button v-if="canMarkPaid" type="primary" @click="doMarkPaid">确认付款</el-button>
        <el-button
          v-if="order.status === 'confirmed' && isPickupLike"
          type="success"
          @click="act(() => markSalesReady(id), '已标记可提货（履约状态）')"
        >标记可提货</el-button>
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
          @click="doComplete"
        >{{ isPickupLike ? '标记已提货' : '完成' }}</el-button>
        <el-button
          v-if="order.payStatus === 'paid' && order.needProcurement && !order.purchaseOrderId && ['confirmed','ready','shipping'].includes(order.status)"
          type="warning"
          @click="createPO"
        >生成采购草稿</el-button>
        <el-button
          v-if="!['completed','cancelled'].includes(order.status)"
          type="danger"
          @click="act(() => cancelSalesOrder(id), '已取消')"
        >取消</el-button>
        <el-button v-if="canDelete" type="danger" plain @click="remove">删除</el-button>
      </div>

      <div v-if="order.receiptHtml" class="receipt-wrap">
        <PosReceiptPanel
          :html="order.receiptHtml"
          :order-no="order.orderNo"
          title="销售单"
          variant="sales-doc"
          compact
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
.pad-l { margin-left: 8px; }
.hint-inline { margin-left: 6px; color: #909399; font-size: 12px; }
</style>
