<script setup lang="ts">
import { Money, Sell, Tools, Box, ShoppingCart, VideoCamera, List, Collection } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const modules = [
  { path: '/pos', title: '收银台', desc: '商品 + 服务一起结算，全屏收银，电子小票预览下载', icon: Money, color: '#409eff' },
  { path: '/pos/orders', title: '收银订单', desc: '即时零售订单查询、明细、确认收款与小票', icon: List, color: '#0ea5e9' },
  { path: '/service-catalog', title: '服务目录', desc: '服务分类与服务项目，供收银台与服务工单选用', icon: Collection, color: '#e6a23c' },
  { path: '/sales-orders', title: '销售订单', desc: '线下订货、提货、送货上门、发快递等非即时零售', icon: Sell, color: '#67c23a' },
  { path: '/service-orders', title: '服务工单', desc: '即时/预约服务履约，可选目录服务并预估费用', icon: Tools, color: '#f59e0b' },
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
