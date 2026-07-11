<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  HomeFilled, Shop, Money, Sell, Tools, Box, ShoppingCart, VideoCamera, Ticket, List, Collection,
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const collapsed = defineModel<boolean>('collapsed', { default: false })

const activeMenu = computed(() => route.path)

const menuItems = [
  { path: '/dashboard', title: '工作台', icon: HomeFilled },
  { path: '/stores', title: '门店档案', icon: Shop },
  { path: '/pos', title: '收银台', icon: Money },
  { path: '/pos/orders', title: '收银订单', icon: List },
  { path: '/receipt-templates', title: '小票模板', icon: Ticket },
  { path: '/service-catalog', title: '服务目录', icon: Collection },
  { path: '/sales-orders', title: '销售订单', icon: Sell },
  { path: '/service-orders', title: '服务工单', icon: Tools },
  { path: '/inventory', title: '门店库存', icon: Box },
  { path: '/purchase-orders', title: '门店采购', icon: ShoppingCart },
  { path: '/surveillance', title: '监控管理', icon: VideoCamera },
]

const logoText = computed(() => (collapsed.value ? 'SC' : 'StoreCore'))

function navigate(path: string) {
  router.push(path)
}

watch(() => route.path, () => {})
</script>

<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="logo">{{ logoText }}</div>
    <el-menu
      :default-active="activeMenu"
      :collapse="collapsed"
      background-color="#001529"
      text-color="#ffffffa6"
      active-text-color="#fff"
    >
      <el-menu-item
        v-for="item in menuItems"
        :key="item.path"
        :index="item.path"
        @click="navigate(item.path)"
      >
        <el-icon><component :is="item.icon" /></el-icon>
        <span>{{ item.title }}</span>
      </el-menu-item>
    </el-menu>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  background: #001529;
  transition: width 0.2s;
  flex-shrink: 0;
}
.sidebar.collapsed {
  width: 64px;
}
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 600;
  font-size: 16px;
  border-bottom: 1px solid #ffffff14;
}
.sidebar :deep(.el-menu) {
  border-right: none;
}
</style>
