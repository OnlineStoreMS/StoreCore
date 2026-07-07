<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import OrderLineEditor, { type OrderLine } from '../../components/OrderLineEditor.vue'
import { createSalesOrder, getSalesOrder, updateSalesOrder } from '../../api/salesOrder'
import { fulfillmentOptions, useStores } from '../../composables/useStores'

const route = useRoute()
const router = useRouter()
const isEdit = computed(() => route.name === 'SalesOrderEdit')
const orderId = computed(() => Number(route.params.id))
const { stores, storeId } = useStores()

const loading = ref(false)
const saving = ref(false)
const form = ref({
  fulfillmentType: 'pickup',
  customerName: '',
  customerPhone: '',
  shippingAddress: '',
  needProcurement: false,
  remark: '',
})
const lines = ref<OrderLine[]>([])

async function load() {
  if (!isEdit.value) {
    lines.value = []
    return
  }
  loading.value = true
  try {
    const so = await getSalesOrder(orderId.value)
    if (so.status !== 'draft') {
      ElMessage.warning('仅草稿可编辑')
      router.replace(`/sales-orders/${orderId.value}`)
      return
    }
    storeId.value = so.storeId
    form.value = {
      fulfillmentType: so.fulfillmentType,
      customerName: so.customerName || '',
      customerPhone: so.customerPhone || '',
      shippingAddress: so.shippingAddress || '',
      needProcurement: so.needProcurement,
      remark: so.remark || '',
    }
    lines.value = (so.items || []).map((it) => ({
      skuId: it.skuId,
      productName: it.productName,
      skuCode: it.skuCode,
      specLabel: it.specLabel,
      quantity: it.quantity,
      unitPrice: it.unitPrice,
    }))
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!storeId.value) {
    ElMessage.warning('请选择门店')
    return
  }
  if (!lines.value.length || !lines.value.every((l) => l.productName && l.quantity > 0)) {
    ElMessage.warning('请添加商品明细')
    return
  }
  saving.value = true
  try {
    const payload = { storeId: storeId.value!, ...form.value, items: lines.value }
    if (isEdit.value) {
      await updateSalesOrder(orderId.value, payload)
      ElMessage.success('已保存')
      router.push(`/sales-orders/${orderId.value}`)
    } else {
      const so = await createSalesOrder(payload)
      ElMessage.success('已创建')
      router.push(`/sales-orders/${so.id}`)
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading">
    <el-page-header :icon="ArrowLeft" @back="router.back()">
      <template #content>{{ isEdit ? '编辑销售订单' : '新建销售订单' }}</template>
    </el-page-header>
    <el-card class="mt-16">
      <el-form label-width="100px">
        <el-form-item label="门店" required>
          <el-select v-model="storeId" style="width: 240px">
            <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="履约方式">
          <el-select v-model="form.fulfillmentType" style="width: 200px">
            <el-option v-for="o in fulfillmentOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="顾客姓名"><el-input v-model="form.customerName" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="电话"><el-input v-model="form.customerPhone" /></el-form-item>
          </el-col>
        </el-row>
        <el-form-item v-if="form.fulfillmentType !== 'pickup'" label="收货地址">
          <el-input v-model="form.shippingAddress" type="textarea" />
        </el-form-item>
        <el-form-item label="需采购">
          <el-switch v-model="form.needProcurement" />
          <span class="hint">勾选表示需向供应商订货后再交付顾客</span>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" /></el-form-item>
        <el-divider>商品明细</el-divider>
        <OrderLineEditor v-model="lines" />
      </el-form>
      <div class="actions">
        <el-button @click="router.back()">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.mt-16 { margin-top: 16px; }
.actions { margin-top: 16px; display: flex; justify-content: flex-end; gap: 8px; }
.hint { margin-left: 8px; color: #909399; font-size: 12px; }
</style>
