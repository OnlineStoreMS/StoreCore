<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listStores, type Store } from '../../api/store'
import { createPosOrder, type OrderLine } from '../../api/pos'
import PosSkuPicker from '../../components/PosSkuPicker.vue'
import type { ProductSkuSearchItem } from '../../api/productSku'

const stores = ref<Store[]>([])
const storeId = ref<number>()
const paymentMethod = ref('cash')
const cart = ref<OrderLine[]>([])
const submitting = ref(false)
const receiptHtml = ref('')

const totalAmount = computed(() =>
  cart.value.reduce((sum, line) => sum + line.unitPrice * line.quantity, 0),
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
  cart.value.push({
    skuId: sku.skuId,
    productName: sku.productName,
    skuCode: sku.skuCode,
    specLabel: sku.specLabel,
    quantity: 1,
    unitPrice: sku.price || 0,
  })
}

function removeLine(index: number) {
  cart.value.splice(index, 1)
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
      items: cart.value,
    })
    receiptHtml.value = order.receiptHtml || ''
    ElMessage.success(`结算成功：${order.orderNo}`)
    cart.value = []
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  const data = await listStores('', 1, 100)
  stores.value = data.list
  if (data.list.length) storeId.value = data.list[0].id
})
</script>

<template>
  <el-row :gutter="16">
    <el-col :span="16">
      <el-card>
        <template #header>
          <div class="card-header">
            <span>收银台 — 即时零售</span>
            <el-select v-model="storeId" placeholder="选择门店" style="width: 200px">
              <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
            </el-select>
          </div>
        </template>
        <PosSkuPicker @select="addSku" />
        <el-table :data="cart" stripe class="mt-16">
          <el-table-column prop="productName" label="商品" min-width="180" />
          <el-table-column prop="specLabel" label="规格" width="120" />
          <el-table-column label="单价" width="100">
            <template #default="{ row }">¥{{ row.unitPrice.toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="数量" width="120">
            <template #default="{ row }">
              <el-input-number v-model="row.quantity" :min="1" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="小计" width="100">
            <template #default="{ row }">¥{{ (row.unitPrice * row.quantity).toFixed(2) }}</template>
          </el-table-column>
          <el-table-column label="" width="80">
            <template #default="{ $index }">
              <el-button link type="danger" @click="removeLine($index)">移除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </el-col>
    <el-col :span="8">
      <el-card>
        <div class="summary">
          <div class="summary-row"><span>合计</span><strong>¥{{ totalAmount.toFixed(2) }}</strong></div>
          <el-form label-width="80px" class="mt-16">
            <el-form-item label="支付方式">
              <el-select v-model="paymentMethod" style="width: 100%">
                <el-option v-for="o in paymentOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </el-form-item>
          </el-form>
          <el-button type="primary" size="large" class="checkout-btn" :loading="submitting" @click="checkout">
            结算
          </el-button>
          <el-alert
            class="mt-16"
            type="info"
            :closable="false"
            title="后续扩展"
            description="创建订单后微信/支付宝扫码支付、大小票模板、云打印机对接。"
          />
        </div>
      </el-card>
      <el-card v-if="receiptHtml" class="mt-16">
        <template #header>电子小票</template>
        <div class="receipt" v-html="receiptHtml" />
      </el-card>
    </el-col>
  </el-row>
</template>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.mt-16 { margin-top: 16px; }
.summary-row { display: flex; justify-content: space-between; font-size: 18px; }
.checkout-btn { width: 100%; margin-top: 8px; }
.receipt { font-size: 13px; line-height: 1.6; }
</style>
