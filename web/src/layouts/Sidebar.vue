<script setup lang="ts">
import { computed, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  HomeFilled, Shop, Money, Sell, Tools, Box, ShoppingCart, VideoCamera, Ticket, List, Collection, Van,
} from '@element-plus/icons-vue'

type MenuChild = { path: string; title: string; icon?: Component }
type MenuItem = { path: string; title: string; icon: Component; children?: MenuChild[] }

const route = useRoute()
const router = useRouter()
const collapsed = defineModel<boolean>('collapsed', { default: false })

const activeMenu = computed(() => route.path)

const menuItems: MenuItem[] = [
  { path: '/dashboard', title: '工作台', icon: HomeFilled },
  { path: '/pos', title: '收银台', icon: Money },
  { path: '/pos/orders', title: '收银订单', icon: List },
  { path: '/receipt-templates', title: '小票模板', icon: Ticket },
  {
    path: '/service-catalog',
    title: '服务目录',
    icon: Collection,
    children: [
      { path: '/service-catalog', title: '服务项目', icon: Collection },
      { path: '/price-list-templates', title: '价目表模板', icon: Ticket },
    ],
  },
  {
    path: '/service-orders',
    title: '服务工单',
    icon: Tools,
    children: [
      { path: '/service-orders', title: '工单列表', icon: List },
      { path: '/service-templates', title: '服务工单模板', icon: Ticket },
    ],
  },
  {
    path: '/sales-orders',
    title: '销售订单',
    icon: Sell,
    children: [
      { path: '/sales-orders', title: '订单列表', icon: List },
      { path: '/sales-templates', title: '销售单模板', icon: Ticket },
    ],
  },
  {
    path: '/inventory',
    title: '门店库存',
    icon: Box,
    children: [
      { path: '/inventory', title: '库存查询', icon: Box },
      { path: '/stock-transfers', title: '调货入库', icon: Van },
    ],
  },
  { path: '/purchase-orders', title: '门店采购', icon: ShoppingCart },
  { path: '/surveillance', title: '监控管理', icon: VideoCamera },
  { path: '/stores', title: '门店档案', icon: Shop },
]

const openMenus = computed(() => {
  const path = route.path
  if (path === '/inventory' || path.startsWith('/stock-transfers')) return ['/inventory']
  if (path.startsWith('/service-catalog') || path.startsWith('/price-list-templates')) return ['/service-catalog']
  if (path.startsWith('/service-orders') || path.startsWith('/service-templates')) return ['/service-orders']
  if (path.startsWith('/sales-orders') || path.startsWith('/sales-templates')) return ['/sales-orders']
  return []
})

const logoText = computed(() => (collapsed.value ? 'SC' : 'StoreCore'))

function navigate(path: string) {
  router.push(path)
}
</script>

<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="logo">{{ logoText }}</div>
    <el-menu
      :default-active="activeMenu"
      :default-openeds="openMenus"
      :collapse="collapsed"
      background-color="#001529"
      text-color="#ffffffa6"
      active-text-color="#fff"
    >
      <template v-for="item in menuItems" :key="item.path">
        <el-sub-menu v-if="item.children?.length" :index="item.path">
          <template #title>
            <el-icon><component :is="item.icon" /></el-icon>
            <span>{{ item.title }}</span>
          </template>
          <el-menu-item
            v-for="child in item.children"
            :key="child.path"
            :index="child.path"
            @click="navigate(child.path)"
          >
            <el-icon v-if="child.icon"><component :is="child.icon" /></el-icon>
            <span>{{ child.title }}</span>
          </el-menu-item>
        </el-sub-menu>
        <el-menu-item
          v-else
          :index="item.path"
          @click="navigate(item.path)"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.title }}</span>
        </el-menu-item>
      </template>
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
