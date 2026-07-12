<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import OrderLineEditor, { type OrderLine } from '../../components/OrderLineEditor.vue'
import { createPurchaseOrder } from '../../api/purchase'
import { listSuppliers, type Supplier } from '../../api/supplier'
import { useStores } from '../../composables/useStores'

const router = useRouter()
const { stores, storeId } = useStores()
const saving = ref(false)
const suppliers = ref<Supplier[]>([])
const form = ref({
  purchaseType: 'stock',
  supplierId: undefined as number | undefined,
  supplierName: '',
  remark: '',
})
const lines = ref<OrderLine[]>([])

onMounted(async () => {
  try {
    const data = await listSuppliers()
    suppliers.value = data.list
  } catch { /* supplycore optional */ }
})

function onSupplierChange(id: number) {
  const s = suppliers.value.find((x) => x.id === id)
  form.value.supplierName = s?.name || ''
}

async function save() {
  if (!storeId.value || !lines.value.length) {
    ElMessage.warning('请选择门店并添加明细')
    return
  }
  saving.value = true
  try {
    const po = await createPurchaseOrder({
      storeId: storeId.value,
      ...form.value,
      items: lines.value,
    })
    ElMessage.success('已创建')
    router.push(`/purchase-orders/${po.id}`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-page-header :icon="ArrowLeft" @back="router.back()">
    <template #content>新建采购单</template>
  </el-page-header>
  <el-card class="mt-16">
    <el-form label-width="100px">
      <el-form-item label="门店" required>
        <el-select v-model="storeId" style="width: 240px">
          <el-option v-for="s in stores" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="采购类型">
        <el-select v-model="form.purchaseType" style="width: 200px">
          <el-option label="门店备货" value="stock" />
          <el-option label="销售驱动" value="sales_driven" />
        </el-select>
      </el-form-item>
      <el-form-item label="供应商">
        <el-select
          v-model="form.supplierId"
          filterable
          clearable
          placeholder="从 SupplyCore 选择"
          style="width: 280px"
          @change="onSupplierChange"
        >
          <el-option v-for="s in suppliers" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
        <el-input v-model="form.supplierName" placeholder="或手动输入" style="width: 200px; margin-left: 8px" />
      </el-form-item>
      <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" /></el-form-item>
      <el-divider>采购明细</el-divider>
      <OrderLineEditor v-model="lines" :store-id="storeId" />
    </el-form>
    <div class="actions">
      <el-button @click="router.back()">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </div>
  </el-card>
</template>

<style scoped>
.mt-16 { margin-top: 16px; }
.actions { margin-top: 16px; display: flex; justify-content: flex-end; gap: 8px; }
</style>
