<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import OrderLineEditor, { type OrderLine } from '../../components/OrderLineEditor.vue'
import SalesServicePicker from '../../components/SalesServicePicker.vue'
import {
  createSalesOrder,
  getSalesOrder,
  updateSalesOrder,
  type SalesServiceLine,
} from '../../api/salesOrder'
import {
  deliveryTypeOptions,
  fulfillmentOptions,
  useStores,
} from '../../composables/useStores'

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
  appointmentAt: '' as string,
  pickupPersonName: '',
  pickupPersonPhone: '',
  pickupCode: '',
  deliveryType: 'store_delivery',
  expectedDeliveryAt: '' as string,
  receiverName: '',
  receiverPhone: '',
  shippingAddress: '',
  expressCompany: '',
  expressNo: '',
  expressScheduledAt: '' as string,
  needProcurement: false,
  remark: '',
})
const lines = ref<OrderLine[]>([])
const serviceLines = ref<SalesServiceLine[]>([])

const showAppointment = computed(() => ['pickup', 'install'].includes(form.value.fulfillmentType))
const showPickupPerson = computed(() => ['pickup', 'install'].includes(form.value.fulfillmentType))
const showDelivery = computed(() => form.value.fulfillmentType === 'delivery')
const showExpress = computed(() => form.value.fulfillmentType === 'express')
const showAddress = computed(() => ['delivery', 'express'].includes(form.value.fulfillmentType))
const showInstallServices = computed(() => form.value.fulfillmentType === 'install')

function toLocalInput(iso?: string) {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function toPayloadTime(v: string) {
  if (!v) return null
  // datetime-local → 本地可解析字符串
  return v.length === 16 ? `${v}:00` : v
}

async function load() {
  if (!isEdit.value) {
    lines.value = []
    serviceLines.value = []
    return
  }
  loading.value = true
  try {
    const so = await getSalesOrder(orderId.value)
    if (so.status !== 'draft' && so.status !== 'preview') {
      ElMessage.warning('仅草稿/预结算可编辑')
      router.replace(`/sales-orders/${orderId.value}`)
      return
    }
    storeId.value = so.storeId
    form.value = {
      fulfillmentType: so.fulfillmentType || 'pickup',
      customerName: so.customerName || '',
      customerPhone: so.customerPhone || '',
      appointmentAt: toLocalInput(so.appointmentAt),
      pickupPersonName: so.pickupPersonName || '',
      pickupPersonPhone: so.pickupPersonPhone || '',
      pickupCode: so.pickupCode || '',
      deliveryType: so.deliveryType || 'store_delivery',
      expectedDeliveryAt: toLocalInput(so.expectedDeliveryAt),
      receiverName: so.receiverName || '',
      receiverPhone: so.receiverPhone || '',
      shippingAddress: so.shippingAddress || '',
      expressCompany: so.expressCompany || '',
      expressNo: so.expressNo || '',
      expressScheduledAt: toLocalInput(so.expressScheduledAt),
      needProcurement: so.needProcurement,
      remark: so.remark || '',
    }
    lines.value = (so.items || []).map((it) => ({
      skuId: it.skuId,
      productName: it.productName,
      skuCode: it.skuCode,
      specLabel: it.specLabel,
      pic: it.pic,
      quantity: it.quantity,
      originalPrice: it.originalPrice ?? it.unitPrice,
      discount: it.discount ?? 10,
      unitPrice: it.unitPrice,
    }))
    serviceLines.value = (so.serviceItems || []).map((it) => ({
      serviceItemId: it.serviceItemId,
      serviceName: it.serviceName,
      serviceCode: it.serviceCode,
      quantity: it.quantity,
      originalPrice: it.originalPrice ?? it.unitPrice,
      discount: it.discount ?? 10,
      unitPrice: it.unitPrice,
      durationMin: it.durationMin,
      pic: it.pic,
    }))
  } finally {
    loading.value = false
  }
}

function buildPayload(isPreview = false) {
  return {
    storeId: storeId.value!,
    fulfillmentType: form.value.fulfillmentType,
    isPreview,
    customerName: form.value.customerName,
    customerPhone: form.value.customerPhone,
    appointmentAt: showAppointment.value ? toPayloadTime(form.value.appointmentAt) : null,
    pickupPersonName: showPickupPerson.value ? form.value.pickupPersonName : '',
    pickupPersonPhone: showPickupPerson.value ? form.value.pickupPersonPhone : '',
    pickupCode: form.value.fulfillmentType === 'pickup' ? form.value.pickupCode : '',
    deliveryType: showDelivery.value ? form.value.deliveryType : '',
    expectedDeliveryAt: showDelivery.value ? toPayloadTime(form.value.expectedDeliveryAt) : null,
    receiverName: showAddress.value ? form.value.receiverName : '',
    receiverPhone: showAddress.value ? form.value.receiverPhone : '',
    shippingAddress: showAddress.value ? form.value.shippingAddress : '',
    expressCompany: showExpress.value ? form.value.expressCompany : '',
    expressNo: showExpress.value ? form.value.expressNo : '',
    expressScheduledAt: showExpress.value ? toPayloadTime(form.value.expressScheduledAt) : null,
    needProcurement: form.value.needProcurement,
    remark: form.value.remark,
    items: lines.value,
    serviceItems: showInstallServices.value ? serviceLines.value : [],
  }
}

function validate(forConfirmLike = false) {
  if (!storeId.value) {
    ElMessage.warning('请选择门店')
    return false
  }
  if (!lines.value.length || !lines.value.every((l) => l.productName && l.quantity > 0 && l.skuId > 0)) {
    ElMessage.warning('请从商品目录选择商品（需选择到规格）')
    return false
  }
  if (showAppointment.value && forConfirmLike && !form.value.appointmentAt) {
    ElMessage.warning('请填写预约时间')
    return false
  }
  if (showInstallServices.value && forConfirmLike && !serviceLines.value.length) {
    ElMessage.warning('到店安装请选择服务项目')
    return false
  }
  if (showAddress.value && !form.value.shippingAddress.trim()) {
    ElMessage.warning('请填写收货地址')
    return false
  }
  return true
}

async function save(asPreview = false) {
  if (!validate(false)) return
  if (asPreview && showInstallServices.value && !serviceLines.value.length) {
    // 预结算允许暂无服务，但建议有
  }
  saving.value = true
  try {
    const payload = buildPayload(asPreview)
    if (isEdit.value) {
      const so = await updateSalesOrder(orderId.value, payload)
      ElMessage.success(asPreview ? '已生成预结算' : '已保存')
      router.push(`/sales-orders/${so.id}`)
    } else {
      const so = await createSalesOrder(payload)
      ElMessage.success(asPreview ? '已生成预结算' : '已创建')
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
      <el-form label-width="110px">
        <el-form-item label="门店" required>
          <el-select v-model="storeId" style="width: 240px">
            <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="履约方式">
          <el-radio-group v-model="form.fulfillmentType">
            <el-radio-button v-for="o in fulfillmentOptions" :key="o.value" :value="o.value">
              {{ o.label }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="顾客姓名"><el-input v-model="form.customerName" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="电话"><el-input v-model="form.customerPhone" /></el-form-item>
          </el-col>
        </el-row>

        <template v-if="showAppointment">
          <el-form-item :label="form.fulfillmentType === 'install' ? '安装预约' : '提货预约'" required>
            <el-date-picker
              v-model="form.appointmentAt"
              type="datetime"
              value-format="YYYY-MM-DDTHH:mm"
              placeholder="选择预约时间"
              style="width: 240px"
            />
          </el-form-item>
        </template>

        <template v-if="showPickupPerson">
          <el-row :gutter="16">
            <el-col :span="8">
              <el-form-item label="取件人">
                <el-input v-model="form.pickupPersonName" placeholder="默认顾客姓名" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="取件电话">
                <el-input v-model="form.pickupPersonPhone" placeholder="默认顾客电话" />
              </el-form-item>
            </el-col>
            <el-col v-if="form.fulfillmentType === 'pickup'" :span="8">
              <el-form-item label="取件码">
                <el-input v-model="form.pickupCode" placeholder="可选" />
              </el-form-item>
            </el-col>
          </el-row>
        </template>

        <template v-if="showDelivery">
          <el-form-item label="配送类型">
            <el-select v-model="form.deliveryType" style="width: 200px">
              <el-option v-for="o in deliveryTypeOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="期望配送">
            <el-date-picker
              v-model="form.expectedDeliveryAt"
              type="datetime"
              value-format="YYYY-MM-DDTHH:mm"
              style="width: 240px"
            />
          </el-form-item>
        </template>

        <template v-if="showAddress">
          <el-row :gutter="16">
            <el-col :span="8">
              <el-form-item label="收货人">
                <el-input v-model="form.receiverName" placeholder="默认顾客姓名" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="收货电话">
                <el-input v-model="form.receiverPhone" placeholder="默认顾客电话" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item label="收货地址" required>
            <el-input v-model="form.shippingAddress" type="textarea" :rows="2" />
          </el-form-item>
        </template>

        <template v-if="showExpress">
          <el-row :gutter="16">
            <el-col :span="8">
              <el-form-item label="预约快递">
                <el-date-picker
                  v-model="form.expressScheduledAt"
                  type="datetime"
                  value-format="YYYY-MM-DDTHH:mm"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="快递公司">
                <el-input v-model="form.expressCompany" placeholder="后续对接发货中心" />
              </el-form-item>
            </el-col>
          </el-row>
        </template>

        <el-form-item label="需采购">
          <el-switch v-model="form.needProcurement" />
          <span class="hint">勾选后可从本单生成采购单；到店安装同样支持</span>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" /></el-form-item>

        <el-divider>商品明细</el-divider>
        <OrderLineEditor v-model="lines" :store-id="storeId" />

        <template v-if="showInstallServices">
          <el-divider>服务目录（确认后生成服务工单）</el-divider>
          <SalesServicePicker v-model="serviceLines" />
        </template>
      </el-form>
      <div class="actions">
        <el-button @click="router.back()">取消</el-button>
        <el-button :loading="saving" @click="save(true)">预结算</el-button>
        <el-button type="primary" :loading="saving" @click="save(false)">保存草稿</el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.mt-16 { margin-top: 16px; }
.actions { margin-top: 16px; display: flex; justify-content: flex-end; gap: 8px; }
.hint { margin-left: 8px; color: #909399; font-size: 12px; }
</style>
