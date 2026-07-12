<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Delete, Document, FullScreen, Minus, Picture, Plus, ShoppingCart, Tools,
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { listStores, type Store } from '../../api/store'
import { createPosOrder, type OrderLine } from '../../api/pos'
import { getServiceOrder } from '../../api/serviceOrder'
import { resolvePic } from '../../api/catalog'
import type { ProductSkuSearchItem } from '../../api/productSku'
import PosProductCatalog from '../../components/PosProductCatalog.vue'
import PosServiceCatalog, { type PosServicePick } from '../../components/PosServiceCatalog.vue'
import PosReceiptPanel from '../../components/PosReceiptPanel.vue'

interface CartLine extends OrderLine {
  key: string
  originalPrice: number
  discount: number
  unitPrice: number
  /** 防止折扣/单价互相触发循环 */
  syncing?: boolean
}

const route = useRoute()
const router = useRouter()
const stores = ref<Store[]>([])
const storeId = ref<number>()
const paymentMethod = ref('cash')
const cart = ref<CartLine[]>([])
const submitting = ref(false)
const previewing = ref(false)
const receiptHtml = ref('')
const receiptOrderNo = ref('')
const receiptTitle = ref('电子小票')
const isFullscreen = ref(false)
const posRoot = ref<HTMLElement>()
const catalogTab = ref<'product' | 'service'>('product')
const linkedServiceOrderId = ref(0)
const linkedServiceOrderNo = ref('')
const linkedCustomerName = ref('')
const linkedCustomerPhone = ref('')

function round2(n: number) {
  return Math.round(n * 100) / 100
}

const originalAmount = computed(() =>
  round2(cart.value.reduce((sum, line) => sum + line.originalPrice * line.quantity, 0)),
)

const totalAmount = computed(() =>
  round2(cart.value.reduce((sum, line) => sum + line.unitPrice * line.quantity, 0)),
)

const discountAmount = computed(() => round2(Math.max(0, originalAmount.value - totalAmount.value)))

const totalQty = computed(() =>
  cart.value.reduce((sum, line) => sum + line.quantity, 0),
)

const paymentOptions = [
  { value: 'cash', label: '现金' },
  { value: 'static_qr', label: '静态二维码' },
  { value: 'wechat', label: '微信支付（预留）' },
  { value: 'alipay', label: '支付宝（预留）' },
  { value: 'card', label: '银行卡' },
  { value: 'mixed', label: '组合支付（预留）' },
]

function addSku(sku: ProductSkuSearchItem) {
  const key = `product-${sku.skuId}`
  const existing = cart.value.find((l) => l.key === key)
  if (existing) {
    existing.quantity += 1
    return
  }
  const price = sku.price || 0
  cart.value.unshift({
    key,
    itemType: 'product',
    skuId: sku.skuId,
    productName: sku.productName,
    skuCode: sku.skuCode,
    specLabel: sku.specLabel,
    quantity: 1,
    originalPrice: price,
    discount: 10,
    unitPrice: price,
    pic: resolvePic(sku.pic, sku.productPic),
  })
}

function addService(item: PosServicePick) {
  const key = `service-${item.serviceItemId}`
  const existing = cart.value.find((l) => l.key === key)
  if (existing) {
    existing.quantity += 1
    return
  }
  const price = item.price || 0
  cart.value.unshift({
    key,
    itemType: 'service',
    serviceItemId: item.serviceItemId,
    productName: item.name,
    skuCode: item.code,
    specLabel: item.durationMin ? `约 ${item.durationMin} 分钟` : (item.categoryName || '服务'),
    quantity: 1,
    originalPrice: price,
    discount: 10,
    unitPrice: price,
    pic: item.pic,
  })
}

function changeQty(line: CartLine, delta: number) {
  line.quantity = Math.max(1, line.quantity + delta)
}

function onDiscountChange(line: CartLine) {
  if (line.syncing) return
  line.syncing = true
  let d = Number(line.discount)
  if (!Number.isFinite(d) || d <= 0) d = 10
  if (d > 10) d = 10
  line.discount = round2(d)
  line.unitPrice = round2(line.originalPrice * line.discount / 10)
  line.syncing = false
}

function onUnitPriceChange(line: CartLine) {
  if (line.syncing) return
  line.syncing = true
  let p = Number(line.unitPrice)
  if (!Number.isFinite(p) || p < 0) p = 0
  line.unitPrice = round2(p)
  if (line.originalPrice > 0) {
    let d = round2(line.unitPrice / line.originalPrice * 10)
    if (d > 10) d = 10
    if (d <= 0) d = 0.01
    line.discount = d
  } else {
    line.discount = 10
    line.originalPrice = line.unitPrice
  }
  line.syncing = false
}

function removeLine(index: number) {
  cart.value.splice(index, 1)
}

function clearCart() {
  cart.value = []
}

function buildItemsPayload() {
  return cart.value.map((line) => ({
    itemType: line.itemType || 'product',
    skuId: line.skuId || 0,
    serviceItemId: line.serviceItemId || 0,
    productName: line.productName,
    skuCode: line.skuCode,
    specLabel: line.specLabel,
    quantity: line.quantity,
    originalPrice: line.originalPrice,
    discount: line.discount,
    unitPrice: line.unitPrice,
    pic: line.pic,
  }))
}

async function toggleFullscreen() {
  const el = posRoot.value
  if (!el) return
  try {
    if (!document.fullscreenElement) {
      await el.requestFullscreen()
    } else {
      await document.exitFullscreen()
    }
  } catch {
    ElMessage.warning('当前浏览器不支持全屏')
  }
}

function onFullscreenChange() {
  isFullscreen.value = !!document.fullscreenElement
}

async function createPreview() {
  if (!storeId.value) {
    ElMessage.warning('请选择门店')
    return
  }
  if (cart.value.length === 0) {
    ElMessage.warning('请添加商品或服务')
    return
  }
  previewing.value = true
  try {
    const order = await createPosOrder({
      storeId: storeId.value,
      isPreview: true,
      receiptType: 'small',
      customerName: linkedCustomerName.value || undefined,
      customerPhone: linkedCustomerPhone.value || undefined,
      items: buildItemsPayload(),
    })
    receiptHtml.value = order.receiptHtml || ''
    receiptOrderNo.value = order.orderNo || ''
    receiptTitle.value = '预结算单'
    ElMessage.success(`已生成预结算单：${order.orderNo}`)
  } finally {
    previewing.value = false
  }
}

function clearLinkedServiceOrder() {
  linkedServiceOrderId.value = 0
  linkedServiceOrderNo.value = ''
  linkedCustomerName.value = ''
  linkedCustomerPhone.value = ''
  router.replace({ path: '/pos', query: {} })
}

async function loadServiceOrderToCart(id: number) {
  const so = await getServiceOrder(id)
  if (!['awaiting_payment', 'in_progress'].includes(so.status)) {
    ElMessage.warning('仅待付款或进行中的服务工单可结算')
    return
  }
  if (so.payStatus === 'paid' || so.posOrderId) {
    ElMessage.warning('该服务工单已结算')
    return
  }
  storeId.value = so.storeId
  linkedServiceOrderId.value = so.id
  linkedServiceOrderNo.value = so.orderNo
  linkedCustomerName.value = so.customerName || ''
  linkedCustomerPhone.value = so.customerPhone || ''
  catalogTab.value = 'service'
  cart.value = (so.items || []).map((it) => {
    const price = Number(it.unitPrice) || 0
    return {
      key: `service-${it.serviceItemId}`,
      itemType: 'service' as const,
      serviceItemId: it.serviceItemId,
      productName: it.serviceName,
      skuCode: it.serviceCode,
      specLabel: it.durationMin ? `约 ${it.durationMin} 分钟` : '服务工单',
      quantity: it.quantity || 1,
      originalPrice: price,
      discount: 10,
      unitPrice: price,
      pic: it.pic,
    }
  })
  ElMessage.success(`已载入服务工单 ${so.orderNo}，可调整后结算`)
}

async function checkout() {
  if (!storeId.value) {
    ElMessage.warning('请选择门店')
    return
  }
  if (cart.value.length === 0) {
    ElMessage.warning('请添加商品或服务')
    return
  }
  submitting.value = true
  try {
    const order = await createPosOrder({
      storeId: storeId.value,
      paymentMethod: paymentMethod.value,
      receiptType: 'small',
      customerName: linkedCustomerName.value || undefined,
      customerPhone: linkedCustomerPhone.value || undefined,
      serviceOrderId: linkedServiceOrderId.value || undefined,
      items: buildItemsPayload(),
    })
    receiptHtml.value = order.receiptHtml || ''
    receiptOrderNo.value = order.orderNo || ''
    receiptTitle.value = '电子小票'
    ElMessage.success(`结算成功：${order.orderNo}`)
    const soId = linkedServiceOrderId.value
    cart.value = []
    clearLinkedServiceOrder()
    if (soId && order.payStatus === 'paid') {
      router.push(`/service-orders/${soId}`)
    }
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  document.addEventListener('fullscreenchange', onFullscreenChange)
  const data = await listStores('', 1, 100)
  stores.value = data.list
  if (data.list.length) storeId.value = data.list[0].id
  const soId = Number(route.query.serviceOrderId || 0)
  if (soId) {
    try {
      await loadServiceOrderToCart(soId)
    } catch (e) {
      ElMessage.error((e as Error).message || '载入服务工单失败')
    }
  }
})

onUnmounted(() => {
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  if (document.fullscreenElement) {
    void document.exitFullscreen()
  }
})
</script>

<template>
  <div ref="posRoot" class="pos-page" :class="{ fullscreen: isFullscreen }">
    <header class="pos-header">
      <div class="pos-header-left">
        <h1 class="pos-title">收银台</h1>
        <el-select v-model="storeId" placeholder="选择门店" class="store-select" :disabled="!!linkedServiceOrderId">
          <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
        <el-radio-group v-model="catalogTab" size="default">
          <el-radio-button value="product">商品</el-radio-button>
          <el-radio-button value="service">服务</el-radio-button>
        </el-radio-group>
      </div>
      <div class="pos-header-right">
        <el-tag v-if="linkedServiceOrderId" type="warning" effect="plain" class="link-tag">
          结算工单 {{ linkedServiceOrderNo }}
          <el-button link type="primary" @click="router.push(`/service-orders/${linkedServiceOrderId}`)">查看</el-button>
          <el-button link @click="clearLinkedServiceOrder">取消关联</el-button>
        </el-tag>
        <el-tag v-else type="info" effect="plain">商品 + 服务</el-tag>
        <el-button :icon="FullScreen" @click="toggleFullscreen">
          {{ isFullscreen ? '退出全屏' : '全屏' }}
        </el-button>
      </div>
    </header>

    <div class="pos-body">
      <div class="pos-catalog-panel">
        <PosProductCatalog
          v-show="catalogTab === 'product'"
          :store-id="storeId"
          :require-store-stock="true"
          @select="addSku"
        />
        <PosServiceCatalog v-show="catalogTab === 'service'" @select="addService" />
      </div>

      <aside class="pos-cart-panel">
        <div class="cart-header">
          <div class="cart-title">
            <el-icon><ShoppingCart /></el-icon>
            <span>购物车</span>
            <el-badge v-if="totalQty" :value="totalQty" class="cart-badge" />
          </div>
          <el-button v-if="cart.length" link type="danger" @click="clearCart">清空</el-button>
        </div>

        <div v-if="cart.length === 0" class="cart-empty">
          <el-icon class="empty-icon"><ShoppingCart /></el-icon>
          <p>添加商品或服务后结算</p>
        </div>

        <div v-else class="cart-lines">
          <div v-for="(line, index) in cart" :key="line.key" class="cart-line">
            <div class="line-pic" :class="{ service: line.itemType === 'service' }">
              <el-image v-if="line.pic" :src="line.pic" fit="cover" class="line-img">
                <template #error>
                  <div class="line-pic-fallback">
                    <el-icon><component :is="line.itemType === 'service' ? Tools : Picture" /></el-icon>
                  </div>
                </template>
              </el-image>
              <div v-else class="line-pic-fallback">
                <el-icon><component :is="line.itemType === 'service' ? Tools : Picture" /></el-icon>
              </div>
            </div>
            <div class="line-main">
              <div class="line-name">
                <el-tag
                  size="small"
                  :type="line.itemType === 'service' ? 'warning' : 'primary'"
                  effect="plain"
                  class="type-tag"
                >
                  {{ line.itemType === 'service' ? '服务' : '商品' }}
                </el-tag>
                {{ line.productName }}
              </div>
              <div class="line-spec">{{ line.specLabel }}</div>
              <div class="line-orig">原价 ¥{{ line.originalPrice.toFixed(2) }}</div>
              <div class="line-discount-row">
                <span class="field-label">折扣</span>
                <el-input-number
                  v-model="line.discount"
                  :min="0.01"
                  :max="10"
                  :step="0.1"
                  :precision="2"
                  size="small"
                  controls-position="right"
                  class="discount-input"
                  @change="onDiscountChange(line)"
                />
                <span class="field-unit">折</span>
                <span class="field-label">实付</span>
                <el-input-number
                  v-model="line.unitPrice"
                  :min="0"
                  :step="1"
                  :precision="2"
                  size="small"
                  controls-position="right"
                  class="price-input"
                  @change="onUnitPriceChange(line)"
                />
              </div>
              <div class="line-bottom">
                <span class="line-price">
                  ¥{{ line.unitPrice.toFixed(2) }}
                  <span class="line-sub">× {{ line.quantity }} = ¥{{ (line.unitPrice * line.quantity).toFixed(2) }}</span>
                </span>
                <div class="qty-control">
                  <el-button size="small" circle :icon="Minus" @click="changeQty(line, -1)" />
                  <span class="qty-num">{{ line.quantity }}</span>
                  <el-button size="small" circle :icon="Plus" @click="changeQty(line, 1)" />
                </div>
              </div>
            </div>
            <el-button link type="danger" :icon="Delete" class="line-remove" @click="removeLine(index)" />
          </div>
        </div>

        <div class="cart-checkout">
          <div v-if="discountAmount > 0" class="summary-row muted">
            <span>原价合计</span>
            <span class="strike">¥{{ originalAmount.toFixed(2) }}</span>
          </div>
          <div v-if="discountAmount > 0" class="summary-row muted">
            <span>优惠</span>
            <span>-¥{{ discountAmount.toFixed(2) }}</span>
          </div>
          <div class="summary-row">
            <span>应付合计</span>
            <strong class="summary-amount">¥{{ totalAmount.toFixed(2) }}</strong>
          </div>
          <div class="summary-sub">共 {{ totalQty }} 项</div>

          <el-form label-width="72px" class="payment-form">
            <el-form-item label="支付方式">
              <el-select v-model="paymentMethod" style="width: 100%">
                <el-option v-for="o in paymentOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </el-form-item>
          </el-form>

          <div class="action-btns">
            <el-button
              :icon="Document"
              size="large"
              class="preview-btn"
              :loading="previewing"
              :disabled="cart.length === 0"
              @click="createPreview"
            >
              预结算单
            </el-button>
            <el-button
              type="primary"
              size="large"
              class="checkout-btn"
              :loading="submitting"
              :disabled="cart.length === 0"
              @click="checkout"
            >
              结算 ¥{{ totalAmount.toFixed(2) }}
            </el-button>
          </div>
        </div>

        <PosReceiptPanel
          :html="receiptHtml"
          :order-no="receiptOrderNo"
          :title="receiptTitle"
          :auto-open="receiptTitle === '预结算单'"
          compact
        />
      </aside>
    </div>
  </div>
</template>

<style scoped>
.pos-page {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 56px - 32px);
  min-height: 640px;
  margin: -16px;
  background: #eef1f6;
}
.pos-page.fullscreen {
  margin: 0;
  height: 100vh;
  min-height: 100vh;
  background: #eef1f6;
}
.pos-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
}
.pos-header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.pos-header-right {
  display: flex;
  align-items: center;
  gap: 10px;
}
.link-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 420px;
}
.pos-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: #303133;
}
.store-select { width: 180px; }
.pos-body {
  flex: 1;
  min-height: 0;
  display: flex;
  gap: 12px;
  padding: 12px;
}
.pos-catalog-panel {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.pos-cart-panel {
  width: 460px;
  flex-shrink: 0;
  background: #fff;
  border-radius: 12px;
  border: 1px solid #ebeef5;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.cart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px 10px;
  border-bottom: 1px solid #f0f2f5;
}
.cart-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
}
.cart-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #909399;
  padding: 32px;
}
.empty-icon { font-size: 48px; margin-bottom: 12px; opacity: 0.4; }
.cart-lines { flex: 1; overflow-y: auto; padding: 8px 12px; }
.cart-line {
  display: flex;
  gap: 10px;
  padding: 10px 4px;
  border-bottom: 1px solid #f5f7fa;
  align-items: flex-start;
}
.line-pic {
  width: 52px; height: 52px; border-radius: 8px; overflow: hidden;
  flex-shrink: 0; background: #f5f7fa;
}
.line-pic.service { background: #fff7e6; }
.line-img { width: 100%; height: 100%; }
.line-pic-fallback {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center; color: #c0c4cc;
}
.line-main { flex: 1; min-width: 0; }
.line-name {
  font-size: 13px; font-weight: 500; color: #303133; line-height: 1.35;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
}
.type-tag { margin-right: 4px; vertical-align: middle; }
.line-spec { margin-top: 2px; font-size: 11px; color: #909399; }
.line-orig {
  margin-top: 4px;
  font-size: 11px;
  color: #909399;
  text-decoration: line-through;
}
.line-discount-row {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 6px;
  flex-wrap: nowrap;
}
.field-label { font-size: 11px; color: #909399; flex-shrink: 0; }
.field-unit { font-size: 11px; color: #606266; margin-right: 2px; flex-shrink: 0; }
.discount-input { width: 86px; flex-shrink: 0; }
.price-input { width: 98px; flex-shrink: 0; }
.discount-input :deep(.el-input__wrapper),
.price-input :deep(.el-input__wrapper) {
  padding-left: 6px;
  padding-right: 6px;
}
.line-bottom {
  display: flex; align-items: center; justify-content: space-between; margin-top: 6px;
}
.line-price { font-size: 14px; font-weight: 700; color: #f56c6c; }
.line-sub { font-size: 11px; font-weight: 400; color: #909399; margin-left: 4px; }
.qty-control { display: flex; align-items: center; gap: 6px; }
.qty-num { min-width: 20px; text-align: center; font-size: 14px; font-weight: 600; }
.line-remove { flex-shrink: 0; margin-top: 2px; }
.cart-checkout {
  padding: 12px 16px 16px;
  border-top: 1px solid #ebeef5;
  background: #fafbfc;
}
.summary-row {
  display: flex; justify-content: space-between; align-items: baseline; font-size: 15px;
}
.summary-row.muted { font-size: 13px; color: #909399; margin-bottom: 2px; }
.strike { text-decoration: line-through; }
.summary-amount { font-size: 26px; color: #f56c6c; }
.summary-sub { margin-top: 2px; font-size: 12px; color: #909399; text-align: right; }
.payment-form { margin-top: 12px; }
.action-btns { display: flex; gap: 8px; margin-top: 4px; }
.preview-btn { flex: 1; height: 48px; }
.checkout-btn {
  flex: 1.4; height: 48px; font-size: 16px; font-weight: 600;
}
</style>
