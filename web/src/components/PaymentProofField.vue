<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { ElMessage, type UploadRequestOptions } from 'element-plus'
import { Iphone, Plus } from '@element-plus/icons-vue'
import QRCode from 'qrcode'
import {
  createPhotoUploadSession,
  getPhotoUploadSession,
  uploadImage,
} from '../api/upload'
import { recognizePayTimeFromImage } from '../utils/payTimeOcr'

const props = withDefaults(
  defineProps<{
    proofUrl?: string
    paidAt?: string
    subdir?: string
    requireProof?: boolean
  }>(),
  {
    proofUrl: '',
    paidAt: '',
    subdir: 'payments/service',
    requireProof: true,
  },
)

const emit = defineEmits<{
  (e: 'update:proofUrl', v: string): void
  (e: 'update:paidAt', v: string): void
}>()

const uploading = ref(false)
const recognizing = ref(false)
const ocrProgress = ref(0)

const scanVisible = ref(false)
const scanLoading = ref(false)
const qrDataUrl = ref('')
const scanToken = ref('')
const scanStatus = ref<'idle' | 'waiting' | 'done' | 'expired'>('idle')
let pollTimer: ReturnType<typeof setInterval> | null = null

function stopPoll() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function runOcr(source: File | Blob | string) {
  recognizing.value = true
  ocrProgress.value = 0
  try {
    const res = await recognizePayTimeFromImage(source, (p) => {
      ocrProgress.value = p
    })
    if (res.paidAt) {
      emit('update:paidAt', res.paidAt)
      const tip = res.source === 'receive' ? '已识别收款时间' : '已识别转账时间'
      ElMessage.success(`${tip}：${res.paidAt}`)
    } else {
      ElMessage.warning('未能从截图识别付款时间，请手动填写')
    }
  } catch (e) {
    ElMessage.warning((e as Error).message || '付款时间识别失败，请手动填写')
  } finally {
    recognizing.value = false
    ocrProgress.value = 0
  }
}

async function doUpload(opt: UploadRequestOptions) {
  uploading.value = true
  try {
    const file = opt.file as File
    const url = await uploadImage(file, props.subdir)
    emit('update:proofUrl', url)
    ElMessage.success('截图已上传')
    await runOcr(file)
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
  } finally {
    uploading.value = false
  }
}

async function openScan() {
  scanVisible.value = true
  scanLoading.value = true
  scanStatus.value = 'idle'
  qrDataUrl.value = ''
  scanToken.value = ''
  stopPoll()
  try {
    const session = await createPhotoUploadSession(props.subdir)
    scanToken.value = session.token
    const pageUrl = `${window.location.origin}/m/photo-upload?token=${encodeURIComponent(session.token)}`
    qrDataUrl.value = await QRCode.toDataURL(pageUrl, {
      width: 220,
      margin: 2,
      errorCorrectionLevel: 'M',
    })
    scanStatus.value = 'waiting'
    pollTimer = setInterval(async () => {
      try {
        const s = await getPhotoUploadSession(scanToken.value)
        if (s.status === 'done' && s.url) {
          emit('update:proofUrl', s.url)
          scanStatus.value = 'done'
          stopPoll()
          ElMessage.success('付款截图已上传')
          setTimeout(() => {
            scanVisible.value = false
          }, 500)
          await runOcr(s.url)
        }
      } catch {
        scanStatus.value = 'expired'
        stopPoll()
      }
    }, 2000)
  } catch (e) {
    ElMessage.error((e as Error).message || '创建扫码会话失败')
    scanVisible.value = false
  } finally {
    scanLoading.value = false
  }
}

function closeScan() {
  scanVisible.value = false
  stopPoll()
}

function clearProof() {
  emit('update:proofUrl', '')
}

watch(
  () => props.proofUrl,
  () => {
    /* keep */
  },
)

onUnmounted(stopPoll)

defineExpose({ runOcr })
</script>

<template>
  <div class="pay-proof">
    <div class="proof-row">
      <el-upload
        :show-file-list="false"
        accept="image/*"
        :disabled="uploading || recognizing"
        :http-request="doUpload"
      >
        <div v-if="proofUrl" class="proof-thumb">
          <el-image :src="proofUrl" fit="contain" class="proof-thumb-img" />
        </div>
        <div v-else class="proof-thumb placeholder">
          <el-icon><Plus /></el-icon>
          <span>{{ uploading ? '上传中…' : '本机上传' }}</span>
        </div>
      </el-upload>
      <div class="proof-actions">
        <el-button type="primary" plain :icon="Iphone" :disabled="recognizing" @click="openScan">
          手机扫码上传
        </el-button>
        <el-button v-if="proofUrl" link type="danger" @click="clearProof">移除</el-button>
        <el-button
          v-if="proofUrl"
          link
          type="primary"
          :loading="recognizing"
          @click="runOcr(proofUrl)"
        >
          重新识别时间
        </el-button>
      </div>
    </div>

    <div v-if="recognizing" class="ocr-tip">正在识别付款时间… {{ ocrProgress }}%</div>

    <div class="paid-at-row">
      <span class="label">付款时间</span>
      <el-date-picker
        :model-value="paidAt"
        type="datetime"
        value-format="YYYY-MM-DD HH:mm:ss"
        format="YYYY-MM-DD HH:mm:ss"
        placeholder="上传截图后自动识别，可手动修改"
        style="width: 100%"
        @update:model-value="(v: string) => emit('update:paidAt', v || '')"
      />
    </div>
    <div v-if="requireProof" class="hint">转账收款请上传付款截图；优先识别「收款时间」，其次「转账时间」</div>
  </div>

  <el-dialog
    v-model="scanVisible"
    title="手机扫码上传付款截图"
    width="360px"
    append-to-body
    destroy-on-close
    @closed="closeScan"
  >
    <div class="scan-body" v-loading="scanLoading">
      <img v-if="qrDataUrl" :src="qrDataUrl" alt="扫码上传" class="qr" />
      <p v-if="scanStatus === 'waiting'" class="scan-hint">请用手机扫描二维码，拍照或从相册选择付款截图</p>
      <p v-else-if="scanStatus === 'done'" class="scan-hint ok">上传成功</p>
      <p v-else-if="scanStatus === 'expired'" class="scan-hint err">会话已过期，请关闭后重试</p>
      <p v-else class="scan-hint">正在生成二维码…</p>
    </div>
    <template #footer>
      <el-button @click="closeScan">关闭</el-button>
      <el-button type="primary" :disabled="scanLoading" @click="openScan">刷新二维码</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.pay-proof { width: 100%; }
.proof-row { display: flex; align-items: flex-start; gap: 12px; flex-wrap: wrap; }
.proof-actions { display: flex; flex-direction: column; align-items: flex-start; gap: 8px; padding-top: 8px; }
.proof-thumb {
  width: 140px; height: 140px; border-radius: 8px; border: 1px dashed #dcdfe6;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 6px; color: #909399; background: #fafafa; cursor: pointer; overflow: hidden;
}
.proof-thumb.placeholder { font-size: 12px; }
.proof-thumb-img { width: 140px; height: 140px; }
.ocr-tip { margin-top: 8px; font-size: 12px; color: #409eff; }
.paid-at-row { margin-top: 14px; display: flex; align-items: center; gap: 12px; }
.paid-at-row .label { flex-shrink: 0; width: 64px; color: #606266; font-size: 14px; }
.hint { margin-top: 8px; font-size: 12px; color: #909399; line-height: 1.5; }
.scan-body {
  display: flex; flex-direction: column; align-items: center; min-height: 260px; justify-content: center;
}
.qr { width: 220px; height: 220px; border: 1px solid #ebeef5; border-radius: 8px; }
.scan-hint { margin: 12px 0 0; font-size: 13px; color: #606266; text-align: center; line-height: 1.5; }
.scan-hint.ok { color: #67c23a; }
.scan-hint.err { color: #f56c6c; }
</style>
