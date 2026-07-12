<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Download, View } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import html2canvas from 'html2canvas'

const props = defineProps<{
  html: string
  orderNo?: string
  compact?: boolean
  title?: string
  /** receipt=收银小票；sales-doc=横向销售单 */
  variant?: 'receipt' | 'sales-doc'
  /** 内容更新后自动打开预览弹窗（预结算单） */
  autoOpen?: boolean
}>()

const isSalesDoc = computed(() => props.variant === 'sales-doc')
const dialogWidth = computed(() => (isSalesDoc.value ? '920px' : '440px'))
const paperWidth = computed(() => (isSalesDoc.value ? '860px' : '320px'))
const exportWidth = computed(() => (isSalesDoc.value ? '860px' : '320px'))
const downloadLabel = computed(() => (isSalesDoc.value ? '下载销售单图片' : '下载图片'))
const successLabel = computed(() => (isSalesDoc.value ? '销售单图片已下载' : '小票图片已下载'))

const previewVisible = ref(false)
const exporting = ref(false)
const receiptRef = ref<HTMLElement>()
const previewRef = ref<HTMLElement>()

watch(
  () => props.html,
  () => {
    if (!props.html) {
      previewVisible.value = false
      return
    }
    previewVisible.value = !!props.autoOpen
  },
)

async function exportImage(target: HTMLElement | undefined, filename: string) {
  if (!target) return
  exporting.value = true
  try {
    await nextTick()
    const canvas = await html2canvas(target, {
      backgroundColor: '#ffffff',
      scale: 2,
      useCORS: true,
      allowTaint: true,
      logging: false,
    })
    const link = document.createElement('a')
    link.download = filename
    link.href = canvas.toDataURL('image/png')
    link.click()
    ElMessage.success(successLabel.value)
  } catch (e) {
    ElMessage.error((e as Error).message || '导出失败')
  } finally {
    exporting.value = false
  }
}

async function downloadFromCard() {
  // compact 无内嵌预览时，优先用弹窗内节点；否则用隐藏导出节点
  const target = receiptRef.value || previewRef.value
  if (!target) {
    previewVisible.value = true
    await nextTick()
    await exportImage(previewRef.value, `receipt-${props.orderNo || Date.now()}.png`)
    return
  }
  await exportImage(target, `receipt-${props.orderNo || Date.now()}.png`)
}

async function openPreview() {
  previewVisible.value = true
}

async function downloadFromPreview() {
  await nextTick()
  await exportImage(previewRef.value, `receipt-${props.orderNo || Date.now()}.png`)
}
</script>

<template>
  <div v-if="html" class="receipt-panel" :class="{ compact }">
    <div class="receipt-toolbar">
      <span class="toolbar-title">{{ title || '电子小票' }}</span>
      <div class="toolbar-actions">
        <el-button size="small" :icon="View" @click="openPreview">预览</el-button>
        <el-button size="small" type="primary" :icon="Download" :loading="exporting" @click="downloadFromCard">
          下载图片
        </el-button>
      </div>
    </div>

    <!-- 订单详情：内嵌完整小票；收银台 compact：仅保留操作按钮 -->
    <div v-if="!compact" class="receipt-scroll">
      <div ref="receiptRef" class="receipt-paper" v-html="html" />
    </div>
    <div v-else class="receipt-export-offscreen" :style="{ width: exportWidth }" aria-hidden="true">
      <div ref="receiptRef" class="receipt-paper" :class="{ 'sales-paper': isSalesDoc }" v-html="html" />
    </div>

    <el-dialog
      v-model="previewVisible"
      :title="title || '预览'"
      :width="dialogWidth"
      top="4vh"
      append-to-body
      destroy-on-close
      class="receipt-preview-dialog"
    >
      <div class="preview-wrap">
        <div
          ref="previewRef"
          class="receipt-paper preview"
          :class="{ 'sales-paper': isSalesDoc }"
          :style="{ width: paperWidth }"
          v-html="html"
        />
      </div>
      <template #footer>
        <el-button @click="previewVisible = false">关闭</el-button>
        <el-button type="primary" :icon="Download" :loading="exporting" @click="downloadFromPreview">
          {{ downloadLabel }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.receipt-panel {
  border-top: 1px solid #ebeef5;
  background: #f7f8fa;
}
.receipt-panel.compact {
  flex-shrink: 0;
}
.receipt-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
}
.toolbar-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}
.toolbar-actions {
  display: flex;
  gap: 6px;
}
.receipt-scroll {
  overflow: visible;
  max-height: none;
  padding: 0;
}
.receipt-panel:not(.compact) {
  border-top: none;
  background: transparent;
}
.receipt-panel:not(.compact) .receipt-toolbar {
  padding: 0 0 12px;
}
.receipt-export-offscreen {
  position: fixed;
  left: -10000px;
  top: 0;
  width: 320px;
  pointer-events: none;
}
.receipt-paper {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 16px 14px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
}
.preview-wrap {
  display: flex;
  justify-content: center;
  background: #f0f2f5;
  padding: 16px;
  border-radius: 8px;
  max-height: calc(100vh - 180px);
  overflow: auto;
}
.receipt-paper.preview {
  width: 320px;
}
.receipt-paper.sales-paper {
  padding: 20px 24px;
  border-radius: 4px;
}
</style>

<style>
/* 小票正式样式（非 scoped，供 v-html 使用） */
.receipt-doc {
  font-family: "PingFang SC", "Microsoft YaHei", "Helvetica Neue", Arial, sans-serif;
  color: #1f2937;
  font-size: 12px;
  line-height: 1.5;
}
.receipt-header {
  text-align: center;
  margin-bottom: 10px;
}
.receipt-cover {
  display: flex;
  justify-content: center;
  margin-bottom: 8px;
}
.receipt-cover img {
  width: 64px;
  height: 64px;
  object-fit: cover;
  border-radius: 8px;
  display: block;
}
.receipt-logo {
  display: flex;
  justify-content: center;
  margin-bottom: 8px;
}
.receipt-logo img {
  max-width: 120px;
  max-height: 56px;
  width: auto;
  height: auto;
  object-fit: contain;
  display: block;
}
.receipt-title {
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: #111827;
}
.receipt-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: #6b7280;
}
.receipt-store {
  margin-top: 6px;
  font-size: 13px;
  font-weight: 600;
}
.receipt-meta-line,
.receipt-extra {
  margin-top: 4px;
  color: #6b7280;
  font-size: 11px;
}
.receipt-divider {
  border-top: 1px dashed #d1d5db;
  margin: 10px 0;
}
.receipt-meta {
  display: grid;
  gap: 4px;
}
.receipt-meta > div {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}
.receipt-meta span {
  color: #9ca3af;
}
.receipt-items {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.receipt-item {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}
.receipt-item-pic {
  width: 42px;
  height: 42px;
  border-radius: 6px;
  overflow: hidden;
  background: #f3f4f6;
  flex-shrink: 0;
}
.receipt-item-pic img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.receipt-item-pic-empty {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  color: #9ca3af;
}
.receipt-item-body {
  flex: 1;
  min-width: 0;
}
.receipt-item-name {
  font-size: 13px;
  font-weight: 600;
  color: #111827;
}
.receipt-item-type {
  display: inline-block;
  font-size: 10px;
  font-weight: 600;
  padding: 1px 5px;
  border-radius: 3px;
  background: #eef2ff;
  color: #4338ca;
  margin-right: 4px;
  vertical-align: middle;
}
.receipt-item-spec,
.receipt-item-code {
  font-size: 11px;
  color: #9ca3af;
}
.receipt-item-row {
  margin-top: 4px;
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.receipt-item-row strong {
  color: #111827;
}
.receipt-orig {
  text-decoration: line-through;
  color: #9ca3af;
}
.receipt-orig-sum {
  text-decoration: line-through;
  color: #9ca3af;
  font-weight: 500 !important;
}
.receipt-summary {
  display: grid;
  gap: 4px;
}
.receipt-summary > div {
  display: flex;
  justify-content: space-between;
}
.receipt-total {
  font-size: 15px;
  margin-top: 2px;
}
.receipt-total b {
  color: #dc2626;
  font-size: 18px;
}
.receipt-footer {
  text-align: center;
}
.receipt-thanks {
  font-size: 13px;
  font-weight: 600;
  color: #374151;
}
.receipt-guide {
  margin-top: 10px;
  text-align: left;
  background: #f9fafb;
  border-radius: 6px;
  padding: 8px 10px;
}
.receipt-guide-title {
  font-size: 11px;
  font-weight: 600;
  color: #6b7280;
  margin-bottom: 4px;
}
.receipt-guide-body {
  font-size: 11px;
  color: #4b5563;
  line-height: 1.55;
}

/* 横向销售单（预结算 / 正式单据） */
.sales-doc {
  font-family: "PingFang SC", "Microsoft YaHei", "Helvetica Neue", Arial, sans-serif;
  color: #1f2937;
  font-size: 12px;
  line-height: 1.45;
  width: 100%;
}
.sales-doc-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  margin-bottom: 14px;
  padding-bottom: 12px;
  border-bottom: 2px solid #111827;
}
.sales-doc-brand { flex: 1; min-width: 0; }
.sales-doc-logo {
  margin-bottom: 8px;
}
.sales-doc-logo img {
  max-height: 48px;
  max-width: 180px;
  object-fit: contain;
  display: block;
}
.sales-doc-store { font-size: 18px; font-weight: 700; color: #111827; }
.sales-doc-muted { margin-top: 3px; color: #6b7280; font-size: 11px; }
.sales-doc-title-block { text-align: right; }
.sales-doc-title { font-size: 22px; font-weight: 700; letter-spacing: 0.12em; color: #111827; }
.sales-doc-badge {
  display: inline-block;
  margin-top: 6px;
  padding: 2px 8px;
  border: 1px solid #d1d5db;
  border-radius: 999px;
  font-size: 11px;
  color: #6b7280;
}
.sales-doc-info {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 12px;
  table-layout: fixed;
}
.sales-doc-info th,
.sales-doc-info td {
  border: 1px solid #e5e7eb;
  padding: 7px 10px;
  vertical-align: top;
  word-break: break-word;
}
.sales-doc-info th {
  width: 88px;
  background: #f9fafb;
  color: #6b7280;
  font-weight: 600;
  text-align: left;
}
.sales-doc-section {
  margin: 8px 0 6px;
  font-size: 13px;
  font-weight: 700;
  color: #111827;
}
.sales-doc-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 10px;
}
.sales-doc-table th,
.sales-doc-table td {
  border: 1px solid #e5e7eb;
  padding: 6px 8px;
  vertical-align: middle;
}
.sales-doc-table thead th {
  background: #f3f4f6;
  color: #374151;
  font-weight: 600;
  white-space: nowrap;
}
.sales-doc-table .col-idx { width: 36px; text-align: center; color: #9ca3af; }
.sales-doc-table .col-pic { width: 52px; text-align: center; }
.sales-doc-table .col-pic img {
  width: 36px;
  height: 36px;
  object-fit: cover;
  border-radius: 4px;
  display: inline-block;
  vertical-align: middle;
}
.sales-doc-table .pic-empty {
  display: inline-block;
  width: 36px;
  height: 36px;
  line-height: 36px;
  text-align: center;
  background: #f3f4f6;
  color: #9ca3af;
  font-size: 10px;
  border-radius: 4px;
}
.sales-doc-table .col-name .name { font-weight: 600; color: #111827; }
.sales-doc-table .col-name .spec { margin-top: 2px; color: #9ca3af; font-size: 11px; }
.sales-doc-table .num { text-align: right; white-space: nowrap; font-variant-numeric: tabular-nums; }
.sales-doc-table .strong { font-weight: 700; }
.sales-doc-table .empty { text-align: center; color: #9ca3af; padding: 16px; }
.sales-doc-summary {
  width: 100%;
  border-collapse: collapse;
  margin-top: 4px;
}
.sales-doc-summary th,
.sales-doc-summary td {
  border: 1px solid #e5e7eb;
  padding: 8px 10px;
}
.sales-doc-summary th {
  width: 20%;
  background: #f9fafb;
  color: #6b7280;
  font-weight: 600;
  text-align: left;
}
.sales-doc-summary td {
  width: 30%;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.sales-doc-summary tr.total th {
  background: #111827;
  color: #fff;
}
.sales-doc-summary tr.total .total-amt {
  background: #fff7ed;
  color: #dc2626;
  font-size: 18px;
  font-weight: 700;
}
.sales-doc-footer {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px dashed #d1d5db;
  color: #6b7280;
  font-size: 12px;
  letter-spacing: 0.02em;
}
</style>
