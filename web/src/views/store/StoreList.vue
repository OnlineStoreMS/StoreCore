<script setup lang="ts">
import { nextTick, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type UploadRequestOptions } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { createStore, deleteStore, listStores, updateStore, type Store } from '../../api/store'
import { uploadImage } from '../../api/upload'
import StoreMapPicker from '../../components/StoreMapPicker.vue'

const loading = ref(false)
const list = ref<Store[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const dialogVisible = ref(false)
const editing = ref<Store | null>(null)
const saving = ref(false)
const mapPicker = ref<{ invalidate: () => void }>()

const emptyForm = () => ({
  code: '',
  name: '',
  shortName: '',
  phone: '',
  province: '',
  city: '',
  district: '',
  address: '',
  businessHours: '',
  brandLogo: '',
  coverPic: '',
  photos: [] as string[],
  guideText: '',
  guidePics: [] as string[],
  longitude: 0,
  latitude: 0,
  mapLabel: '',
  remark: '',
  status: 1,
})

const form = reactive(emptyForm())
/** 营业时间选择器：[开始, 结束]，提交时写成 HH:mm-HH:mm */
const businessHoursRange = ref<[string, string] | null>(null)

function parseBusinessHours(v?: string): [string, string] | null {
  if (!v?.trim()) return null
  const m = v.trim().match(/^(\d{1,2}:\d{2})\s*[-~～至到]\s*(\d{1,2}:\d{2})$/)
  if (!m) return null
  const norm = (t: string) => {
    const [h, min] = t.split(':')
    return `${h.padStart(2, '0')}:${min.padStart(2, '0')}`
  }
  return [norm(m[1]), norm(m[2])]
}

function formatBusinessHours(range: [string, string] | null) {
  if (!range?.[0] || !range?.[1]) return ''
  return `${range[0]}-${range[1]}`
}

async function load() {
  loading.value = true
  try {
    const data = await listStores(keyword.value, page.value, pageSize.value)
    list.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, emptyForm())
  businessHoursRange.value = null
  dialogVisible.value = true
  nextTick(() => mapPicker.value?.invalidate())
}

function openEdit(row: Store) {
  editing.value = row
  Object.assign(form, emptyForm(), {
    ...row,
    photos: [...(row.photos || [])],
    guidePics: [...(row.guidePics || [])],
    longitude: row.longitude || 0,
    latitude: row.latitude || 0,
    coverPic: row.coverPic || '',
    brandLogo: row.brandLogo || '',
    guideText: row.guideText || '',
    mapLabel: row.mapLabel || '',
  })
  businessHoursRange.value = parseBusinessHours(row.businessHours)
  dialogVisible.value = true
  nextTick(() => mapPicker.value?.invalidate())
}

async function submit() {
  if (!form.code || !form.name) {
    ElMessage.warning('请填写门店编码和名称')
    return
  }
  saving.value = true
  try {
    const payload = {
      ...form,
      businessHours: formatBusinessHours(businessHoursRange.value),
      photos: form.photos,
      guidePics: form.guidePics,
    }
    if (editing.value) {
      await updateStore(editing.value.id, payload)
      ElMessage.success('已更新')
    } else {
      await createStore(payload)
      ElMessage.success('已创建')
    }
    dialogVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function remove(row: Store) {
  await ElMessageBox.confirm(`确定删除门店「${row.name}」？`, '确认')
  await deleteStore(row.id)
  ElMessage.success('已删除')
  await load()
}

async function doUpload(options: UploadRequestOptions, target: 'logo' | 'cover' | 'photos' | 'guide') {
  try {
    const folder = target === 'guide' ? 'stores/guide' : target === 'logo' ? 'stores/logo' : 'stores'
    const url = await uploadImage(options.file as File, folder)
    if (target === 'logo') {
      form.brandLogo = url
    } else if (target === 'cover') {
      form.coverPic = url
    } else if (target === 'photos') {
      form.photos.push(url)
    } else {
      form.guidePics.push(url)
    }
    options.onSuccess?.(url as unknown as never)
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
  }
}

function removePhoto(index: number) {
  form.photos.splice(index, 1)
}

function removeGuidePic(index: number) {
  form.guidePics.splice(index, 1)
}

function clearBrandLogo() {
  form.brandLogo = ''
}

function clearCover() {
  form.coverPic = ''
}

onMounted(load)
</script>

<template>
  <el-card>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索门店" clearable style="width: 240px" @keyup.enter="load" />
      <el-button @click="load">查询</el-button>
      <el-button type="primary" @click="openCreate">新建门店</el-button>
    </div>
    <el-table v-loading="loading" :data="list" stripe>
      <el-table-column label="照片" width="72">
        <template #default="{ row }">
          <el-image
            v-if="row.coverPic || row.photos?.[0]"
            :src="row.coverPic || row.photos[0]"
            style="width: 40px; height: 40px; border-radius: 6px"
            fit="cover"
          />
          <span v-else class="muted">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="code" label="编码" width="120" />
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="phone" label="电话" width="130" />
      <el-table-column label="地址" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">
          {{ [row.province, row.city, row.district, row.address].filter(Boolean).join('') || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="地图" width="90">
        <template #default="{ row }">
          <el-tag v-if="row.latitude || row.longitude" type="success" size="small">已标注</el-tag>
          <span v-else class="muted">未标注</span>
        </template>
      </el-table-column>
      <el-table-column prop="businessHours" label="营业时间" width="130" />
      <el-table-column label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'info'">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="load"
      />
    </div>
  </el-card>

  <el-dialog
    v-model="dialogVisible"
    :title="editing ? '编辑门店' : '新建门店'"
    width="820px"
    destroy-on-close
    top="4vh"
    @opened="mapPicker?.invalidate()"
  >
    <el-form label-width="96px" class="store-form">
      <el-divider content-position="left">基本信息</el-divider>
      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item label="编码" required><el-input v-model="form.code" /></el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="简称"><el-input v-model="form.shortName" /></el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="电话"><el-input v-model="form.phone" /></el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="省"><el-input v-model="form.province" /></el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="市"><el-input v-model="form.city" /></el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="区"><el-input v-model="form.district" /></el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="详细地址"><el-input v-model="form.address" /></el-form-item>
        </el-col>
        <el-col :span="14">
          <el-form-item label="营业时间">
            <el-time-picker
              v-model="businessHoursRange"
              is-range
              range-separator="至"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              format="HH:mm"
              value-format="HH:mm"
              style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="10">
          <el-form-item label="状态">
            <el-radio-group v-model="form.status">
              <el-radio :value="1">启用</el-radio>
              <el-radio :value="0">停用</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-col>
      </el-row>

      <el-divider content-position="left">品牌与门店照片</el-divider>
      <el-form-item label="品牌 Logo">
        <div class="upload-row">
          <el-upload
            :show-file-list="false"
            accept="image/*"
            :http-request="(opt: UploadRequestOptions) => doUpload(opt, 'logo')"
          >
            <div v-if="form.brandLogo" class="thumb logo-thumb">
              <el-image :src="form.brandLogo" fit="contain" class="thumb-img" />
            </div>
            <div v-else class="thumb placeholder logo-thumb">
              <el-icon><Plus /></el-icon>
              <span>上传 Logo</span>
            </div>
          </el-upload>
          <el-button v-if="form.brandLogo" link type="danger" @click="clearBrandLogo">移除</el-button>
        </div>
      </el-form-item>
      <el-form-item label="封面图">
        <div class="upload-row">
          <el-upload
            :show-file-list="false"
            accept="image/*"
            :http-request="(opt: UploadRequestOptions) => doUpload(opt, 'cover')"
          >
            <div v-if="form.coverPic" class="thumb">
              <el-image :src="form.coverPic" fit="cover" class="thumb-img" />
            </div>
            <div v-else class="thumb placeholder">
              <el-icon><Plus /></el-icon>
              <span>上传封面</span>
            </div>
          </el-upload>
          <el-button v-if="form.coverPic" link type="danger" @click="clearCover">移除</el-button>
        </div>
      </el-form-item>
      <el-form-item label="门店相册">
        <div class="pic-list">
          <div v-for="(url, i) in form.photos" :key="url + i" class="thumb">
            <el-image :src="url" fit="cover" class="thumb-img" :preview-src-list="form.photos" />
            <button type="button" class="thumb-remove" @click="removePhoto(i)">×</button>
          </div>
          <el-upload
            :show-file-list="false"
            accept="image/*"
            multiple
            :http-request="(opt: UploadRequestOptions) => doUpload(opt, 'photos')"
          >
            <div class="thumb placeholder">
              <el-icon><Plus /></el-icon>
              <span>添加</span>
            </div>
          </el-upload>
        </div>
      </el-form-item>

      <el-divider content-position="left">到店指引</el-divider>
      <el-form-item label="指引说明">
        <el-input
          v-model="form.guideText"
          type="textarea"
          :rows="3"
          placeholder="如：地铁X号线A出口步行200米，商场负一层扶梯旁"
        />
      </el-form-item>
      <el-form-item label="指引图片">
        <div class="pic-list">
          <div v-for="(url, i) in form.guidePics" :key="url + i" class="thumb">
            <el-image :src="url" fit="cover" class="thumb-img" :preview-src-list="form.guidePics" />
            <button type="button" class="thumb-remove" @click="removeGuidePic(i)">×</button>
          </div>
          <el-upload
            :show-file-list="false"
            accept="image/*"
            multiple
            :http-request="(opt: UploadRequestOptions) => doUpload(opt, 'guide')"
          >
            <div class="thumb placeholder">
              <el-icon><Plus /></el-icon>
              <span>添加</span>
            </div>
          </el-upload>
        </div>
      </el-form-item>

      <el-divider content-position="left">地图标注</el-divider>
      <el-form-item label="标注名称">
        <el-input v-model="form.mapLabel" placeholder="地图上显示的名称，默认用门店名" />
      </el-form-item>
      <el-form-item label="位置">
        <StoreMapPicker
          ref="mapPicker"
          v-model:longitude="form.longitude"
          v-model:latitude="form.latitude"
        />
      </el-form-item>

      <el-divider content-position="left">其他</el-divider>
      <el-form-item label="备注">
        <el-input v-model="form.remark" type="textarea" :rows="2" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 16px; }
.pager { margin-top: 16px; display: flex; justify-content: flex-end; }
.muted { color: #c0c4cc; font-size: 12px; }
.store-form :deep(.el-divider) { margin: 12px 0 18px; }
.upload-row { display: flex; align-items: center; gap: 10px; }
.pic-list { display: flex; flex-wrap: wrap; gap: 10px; }
.thumb {
  width: 88px;
  height: 88px;
  border-radius: 8px;
  overflow: hidden;
  position: relative;
  border: 1px dashed #dcdfe6;
  background: #fafafa;
}
.thumb-img { width: 100%; height: 100%; }
.thumb.placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  color: #909399;
  font-size: 12px;
  cursor: pointer;
}
.logo-thumb {
  background: #fff;
  border-style: solid;
  border-color: #e4e7ed;
}
.thumb-remove {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  cursor: pointer;
  line-height: 18px;
  padding: 0;
}
</style>
