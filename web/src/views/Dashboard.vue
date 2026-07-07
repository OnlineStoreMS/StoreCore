<script setup lang="ts">
import { Money, Sell, Tools, Box, ShoppingCart, VideoCamera } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const modules = [
  { path: '/pos', title: '收银台', desc: '同步 ProductCore 商品/SKU，即时零售结算，静态二维码收款，电子小票', icon: Money, color: '#409eff' },
  { path: '/sales-orders', title: '销售订单', desc: '线下订货、提货、送货上门、发快递等非即时零售', icon: Sell, color: '#67c23a' },
  { path: '/service-orders', title: '服务工单', desc: '中高端自行车维修、预约服务单', icon: Tools, color: '#e6a23c' },
  { path: '/inventory', title: '门店库存', desc: '门店库存子集，引用中央 SKU，对接 OSMS 库存体系', icon: Box, color: '#909399' },
  { path: '/purchase-orders', title: '门店采购', desc: '销售驱动采购、门店备货，供应商来自 SupplyCore', icon: ShoppingCart, color: '#f56c6c' },
  { path: '/surveillance', title: '监控管理', desc: '门店室内外监控，实时预览与录像查阅', icon: VideoCamera, color: '#626aef' },
]
</script>

<template>
  <div class="dashboard">
    <el-alert
      title="StoreCore — OSMS 门店管理系统"
      type="info"
      description="与 ProductCore 共用商品底库，与 SupplyCore 共用供应商。请先维护门店档案，再使用各业务模块。"
      show-icon
      :closable="false"
      class="mb-16"
    />
    <el-row :gutter="16">
      <el-col v-for="m in modules" :key="m.path" :span="8" class="mb-16">
        <el-card shadow="hover" class="module-card" @click="router.push(m.path)">
          <div class="stat-card">
            <el-icon :size="32" :color="m.color"><component :is="m.icon" /></el-icon>
            <div>
              <div class="stat-title">{{ m.title }}</div>
              <div class="stat-desc">{{ m.desc }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.mb-16 { margin-bottom: 16px; }
.module-card { cursor: pointer; }
.stat-card {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.stat-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
}
.stat-desc {
  font-size: 13px;
  color: #909399;
  line-height: 1.5;
}
</style>
