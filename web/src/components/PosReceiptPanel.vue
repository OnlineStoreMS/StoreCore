<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { Download, View } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import html2canvas from 'html2canvas'

const props = defineProps<{
  html: string
  orderNo?: string
  compact?: boolean
}>()

const previewVisible = ref(false)
const exporting = ref(false)
const receiptRef = ref<HTMLElement>()
const previewRef = ref<HTMLElement>()

watch(
  () => props.html,
  () => {
    if (props.html) previewVisible.value = false
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
    ElMessage.success('小票图片已下载')
  } catch (e) {
    ElMessage.error((e as Error).message || '导出失败')
  } finally {
    exporting.value = false
  }
}

async function downloadFromCard() {
  await exportImage(receiptRef.value, `receipt-${props.orderNo || Date.now()}.png`)
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
      <span class="toolbar-title">电子小票</span>
      <div class="toolbar-actions">
        <el-button size="small" :icon="View" @click="openPreview">预览</el-button>
        <el-button size="small" type="primary" :icon="Download" :loading="exporting" @click="downloadFromCard">
          下载图片
        </el-button>
      </div>
    </div>
    <div class="receipt-scroll">
      <div ref="receiptRef" class="receipt-paper" v-html="html" />
    </div>

    <el-dialog
      v-model="previewVisible"
      title="电子小票预览"
      width="420px"
      append-to-body
      destroy-on-close
      class="receipt-preview-dialog"
    >
      <div class="preview-wrap">
        <div ref="previewRef" class="receipt-paper preview" v-html="html" />
      </div>
      <template #footer>
        <el-button @click="previewVisible = false">关闭</el-button>
        <el-button type="primary" :icon="Download" :loading="exporting" @click="downloadFromPreview">
          下载图片
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
.receipt-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px 6px;
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
  max-height: 280px;
  overflow: auto;
  padding: 0 12px 12px;
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
}
.receipt-paper.preview {
  width: 320px;
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
</style>
