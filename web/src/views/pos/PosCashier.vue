<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  Delete, FullScreen, Minus, Picture, Plus, ShoppingCart,
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { listStores, type Store } from '../../api/store'
import { createPosOrder, type OrderLine } from '../../api/pos'
import { resolvePic } from '../../api/catalog'
import type { ProductSkuSearchItem } from '../../api/productSku'
import PosProductCatalog from '../../components/PosProductCatalog.vue'
import PosReceiptPanel from '../../components/PosReceiptPanel.vue'

interface CartLine extends OrderLine {
  pic?: string
}

const stores = ref<Store[]>([])
const storeId = ref<number>()
const paymentMethod = ref('cash')
const cart = ref<CartLine[]>([])
const submitting = ref(false)
const receiptHtml = ref('')
const receiptOrderNo = ref('')
const isFullscreen = ref(false)
const posRoot = ref<HTMLElement>()

const totalAmount = computed(() =>
  cart.value.reduce((sum, line) => sum + line.unitPrice * line.quantity, 0),
)

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
  const existing = cart.value.find((l) => l.skuId === sku.skuId)
  if (existing) {
    existing.quantity += 1
    return
  }
  cart.value.unshift({
    skuId: sku.skuId,
    productName: sku.productName,
    skuCode: sku.skuCode,
    specLabel: sku.specLabel,
    quantity: 1,
    unitPrice: sku.price || 0,
    pic: resolvePic(sku.pic, sku.productPic),
  })
}

function changeQty(line: CartLine, delta: number) {
  line.quantity = Math.max(1, line.quantity + delta)
}

function removeLine(index: number) {
  cart.value.splice(index, 1)
}

function clearCart() {
  cart.value = []
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

async function checkout() {
  if (!storeId.value) {
    ElMessage.warning('请选择门店')
    return
  }
  if (cart.value.length === 0) {
    ElMessage.warning('请添加商品')
    return
  }
  submitting.value = true
  try {
    const order = await createPosOrder({
      storeId: storeId.value,
      paymentMethod: paymentMethod.value,
      receiptType: 'small',
      items: cart.value.map(({ skuId, productName, skuCode, specLabel, quantity, unitPrice, pic }) => ({
        skuId,
        productName,
        skuCode,
        specLabel,
        quantity,
        unitPrice,
        pic,
      })),
    })
    receiptHtml.value = order.receiptHtml || ''
    receiptOrderNo.value = order.orderNo || ''
    ElMessage.success(`结算成功：${order.orderNo}`)
    cart.value = []
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  document.addEventListener('fullscreenchange', onFullscreenChange)
  const data = await listStores('', 1, 100)
  stores.value = data.list
  if (data.list.length) storeId.value = data.list[0].id
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
        <el-select v-model="storeId" placeholder="选择门店" class="store-select">
          <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
      </div>
      <div class="pos-header-right">
        <el-tag type="info" effect="plain">即时零售</el-tag>
        <el-button :icon="FullScreen" @click="toggleFullscreen">
          {{ isFullscreen ? '退出全屏' : '全屏' }}
        </el-button>
      </div>
    </header>

    <div class="pos-body">
      <div class="pos-catalog-panel">
        <PosProductCatalog @select="addSku" />
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
          <p>点击左侧商品加入购物车</p>
        </div>

        <div v-else class="cart-lines">
          <div v-for="(line, index) in cart" :key="line.skuId" class="cart-line">
            <div class="line-pic">
              <el-image v-if="line.pic" :src="line.pic" fit="cover" class="line-img">
                <template #error>
                  <div class="line-pic-fallback"><el-icon><Picture /></el-icon></div>
                </template>
              </el-image>
              <div v-else class="line-pic-fallback"><el-icon><Picture /></el-icon></div>
            </div>
            <div class="line-main">
              <div class="line-name">{{ line.productName }}</div>
              <div class="line-spec">{{ line.specLabel }}</div>
              <div class="line-bottom">
                <span class="line-price">¥{{ line.unitPrice.toFixed(2) }}</span>
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
          <div class="summary-row">
            <span>合计</span>
            <strong class="summary-amount">¥{{ totalAmount.toFixed(2) }}</strong>
          </div>
          <div class="summary-sub">共 {{ totalQty }} 件商品</div>

          <el-form label-width="72px" class="payment-form">
            <el-form-item label="支付方式">
              <el-select v-model="paymentMethod" style="width: 100%">
                <el-option v-for="o in paymentOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </el-form-item>
          </el-form>

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

        <PosReceiptPanel :html="receiptHtml" :order-no="receiptOrderNo" compact />
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
.pos-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: #303133;
}
.store-select {
  width: 200px;
}
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
  width: 360px;
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
  color: #303133;
}
.cart-badge {
  margin-left: 4px;
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
.empty-icon {
  font-size: 48px;
  margin-bottom: 12px;
  opacity: 0.4;
}
.cart-lines {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
}
.cart-line {
  display: flex;
  gap: 10px;
  padding: 10px 4px;
  border-bottom: 1px solid #f5f7fa;
  align-items: flex-start;
}
.line-pic {
  width: 52px;
  height: 52px;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
  background: #f5f7fa;
}
.line-img {
  width: 100%;
  height: 100%;
}
.line-pic-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #c0c4cc;
}
.line-main {
  flex: 1;
  min-width: 0;
}
.line-name {
  font-size: 13px;
  font-weight: 500;
  color: #303133;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.line-spec {
  margin-top: 2px;
  font-size: 11px;
  color: #909399;
}
.line-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 6px;
}
.line-price {
  font-size: 14px;
  font-weight: 700;
  color: #f56c6c;
}
.qty-control {
  display: flex;
  align-items: center;
  gap: 6px;
}
.qty-num {
  min-width: 20px;
  text-align: center;
  font-size: 14px;
  font-weight: 600;
}
.line-remove {
  flex-shrink: 0;
  margin-top: 2px;
}
.cart-checkout {
  padding: 12px 16px 16px;
  border-top: 1px solid #ebeef5;
  background: #fafbfc;
}
.summary-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  font-size: 15px;
}
.summary-amount {
  font-size: 26px;
  color: #f56c6c;
}
.summary-sub {
  margin-top: 2px;
  font-size: 12px;
  color: #909399;
  text-align: right;
}
.payment-form {
  margin-top: 12px;
}
.checkout-btn {
  width: 100%;
  margin-top: 4px;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
}
</style>
