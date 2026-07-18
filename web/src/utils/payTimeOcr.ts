/** 从微信支付等截图 OCR 文本中解析付款时间（优先收款时间，其次转账时间） */

import Tesseract from 'tesseract.js'

type Worker = Tesseract.Worker

const LABEL_RECEIVE = /收\s*款\s*时\s*间/
const LABEL_TRANSFER = /转\s*账\s*时\s*间/

/** 2026年7月18日 20:33:22 / 2026-07-18 20:33:22 / 2026/7/18 20:33 */
const DATETIME_RE =
  /(\d{4})\s*[-/.年]\s*(\d{1,2})\s*[-/.月]\s*(\d{1,2})\s*日?\s*(\d{1,2})\s*[:：]\s*(\d{2})(?:\s*[:：]\s*(\d{2}))?/

const MAX_OCR_EDGE = 1280

function pad(n: number) {
  return String(n).padStart(2, '0')
}

function toLocalDateTimeString(y: number, mo: number, d: number, h: number, mi: number, s: number) {
  if (y < 2000 || y > 2100 || mo < 1 || mo > 12 || d < 1 || d > 31) return ''
  if (h > 23 || mi > 59 || s > 59) return ''
  return `${y}-${pad(mo)}-${pad(d)} ${pad(h)}:${pad(mi)}:${pad(s)}`
}

function matchNearLabel(text: string, labelRe: RegExp): string {
  const lines = text
    .replace(/\u00a0/g, ' ')
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter(Boolean)

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    if (!labelRe.test(line)) continue
    const same = line.match(DATETIME_RE)
    if (same) {
      return toLocalDateTimeString(+same[1], +same[2], +same[3], +same[4], +same[5], +(same[6] || 0))
    }
    if (i + 1 < lines.length) {
      const next = lines[i + 1].match(DATETIME_RE)
      if (next) {
        return toLocalDateTimeString(+next[1], +next[2], +next[3], +next[4], +next[5], +(next[6] || 0))
      }
    }
  }

  const compact = text.replace(/\s+/g, '')
  const m = compact.match(
    new RegExp(
      labelRe.source.replace(/\\s\*/g, '') +
        '.*?(\\d{4})年(\\d{1,2})月(\\d{1,2})日(\\d{1,2})[:：](\\d{2})(?:[:：](\\d{2}))?',
    ),
  )
  if (m) {
    return toLocalDateTimeString(+m[1], +m[2], +m[3], +m[4], +m[5], +(m[6] || 0))
  }
  return ''
}

/** 从 OCR 文本解析付款时间，返回 `YYYY-MM-DD HH:mm:ss` 或空串 */
export function parsePayTimeFromOcrText(raw: string): string {
  if (!raw?.trim()) return ''
  const text = raw.replace(/[|丨]/g, ' ')
  const receive = matchNearLabel(text, LABEL_RECEIVE)
  if (receive) return receive
  const transfer = matchNearLabel(text, LABEL_TRANSFER)
  if (transfer) return transfer
  return ''
}

/** 将本地时间串转为提交 API 用的 RFC3339（按浏览器本地时区） */
export function paidAtToApi(local: string): string | undefined {
  const t = local.trim()
  if (!t) return undefined
  const m = t.match(/^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?/)
  if (!m) return undefined
  const d = new Date(+m[1], +m[2] - 1, +m[3], +m[4], +m[5], +(m[6] || 0))
  if (Number.isNaN(d.getTime())) return undefined
  const off = -d.getTimezoneOffset()
  const sign = off >= 0 ? '+' : '-'
  const abs = Math.abs(off)
  const oh = pad(Math.floor(abs / 60))
  const om = pad(abs % 60)
  return `${m[1]}-${m[2]}-${m[3]}T${m[4]}:${m[5]}:${pad(+(m[6] || 0))}${sign}${oh}:${om}`
}

export type RecognizePayTimeResult = {
  paidAt: string
  rawText: string
  source: 'receive' | 'transfer' | ''
}

export type OcrProgress = {
  /** 0-100 */
  percent: number
  /** 展示给用户的阶段说明 */
  phase: string
}

let workerPromise: Promise<Worker> | null = null

function assetBase() {
  const base = (import.meta.env.BASE_URL || '/').replace(/\/?$/, '/')
  return base
}

function mapStatus(status: string, progress: number): OcrProgress {
  const p = Math.max(0, Math.min(100, Math.round((progress || 0) * 100)))
  if (status.includes('loading tesseract core') || status.includes('initializing tesseract')) {
    return { percent: Math.max(5, Math.round(p * 0.25)), phase: '加载识别引擎…' }
  }
  if (status.includes('loading language') || status.includes('initializing api') || status.includes('loaded language')) {
    return { percent: 25 + Math.round(p * 0.25), phase: '加载中文识别包…' }
  }
  if (status.includes('recognizing')) {
    return { percent: 50 + Math.round(p * 0.5), phase: '正在识别付款时间…' }
  }
  return { percent: Math.max(1, Math.round(p * 0.5)), phase: '准备识别…' }
}

/** 复用 Worker，资源走本站静态文件，避免每次从外网 CDN 下载卡住 */
async function getWorker(onProgress?: (info: OcrProgress) => void): Promise<Worker> {
  if (!workerPromise) {
    workerPromise = (async () => {
      const base = assetBase()
      const worker = await Tesseract.createWorker('chi_sim', 1, {
        workerPath: `${base}tesseract/worker.min.js`,
        // 目录路径：自动选择 simd / 非 simd 的 lstm core
        corePath: `${base}tesseract`,
        langPath: `${base}tessdata`,
        gzip: true,
        logger: (m) => {
          if (!onProgress) return
          const status = String(m.status || '')
          const progress = typeof m.progress === 'number' ? m.progress : 0
          onProgress(mapStatus(status, progress))
        },
        errorHandler: (err) => {
          console.warn('[payTimeOcr]', err)
        },
      })
      // 截图多为整屏，稀疏文本模式更快
      await worker.setParameters({
        tessedit_pageseg_mode: Tesseract.PSM.SPARSE_TEXT,
        preserve_interword_spaces: '1',
      })
      return worker
    })().catch((err) => {
      workerPromise = null
      throw err
    })
  }
  return workerPromise
}

/** 缩小图片再 OCR，显著降低耗时 */
async function prepareImageForOcr(source: File | Blob | string): Promise<Blob | string> {
  try {
    let bitmapSource: ImageBitmapSource
    if (typeof source === 'string') {
      const res = await fetch(source)
      bitmapSource = await res.blob()
    } else {
      bitmapSource = source
    }
    const bitmap = await createImageBitmap(bitmapSource)
    const maxEdge = Math.max(bitmap.width, bitmap.height)
    if (maxEdge <= MAX_OCR_EDGE) {
      bitmap.close()
      return typeof source === 'string' ? source : source
    }
    const scale = MAX_OCR_EDGE / maxEdge
    const w = Math.max(1, Math.round(bitmap.width * scale))
    const h = Math.max(1, Math.round(bitmap.height * scale))
    const canvas = document.createElement('canvas')
    canvas.width = w
    canvas.height = h
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      bitmap.close()
      return typeof source === 'string' ? source : source
    }
    ctx.drawImage(bitmap, 0, 0, w, h)
    bitmap.close()
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.85))
    return blob || (typeof source === 'string' ? source : source)
  } catch {
    return typeof source === 'string' ? source : source
  }
}

/** 预加载识别引擎（打开确认收款弹窗时可调用） */
export function preloadPayTimeOcr(onProgress?: (info: OcrProgress) => void): Promise<void> {
  return getWorker(onProgress).then(() => undefined)
}

/** OCR 图片并识别微信等转账截图中的付款时间 */
export async function recognizePayTimeFromImage(
  source: File | Blob | string,
  onProgress?: (info: OcrProgress | number) => void,
): Promise<RecognizePayTimeResult> {
  const report = (info: OcrProgress) => onProgress?.(info)

  report({ percent: 1, phase: '准备识别…' })
  const image = await prepareImageForOcr(source)
  report({ percent: 8, phase: '准备识别…' })

  const worker = await getWorker((info) => report(info))
  report({ percent: 55, phase: '正在识别付款时间…' })

  const result = await worker.recognize(image)
  report({ percent: 100, phase: '识别完成' })

  const rawText = result.data.text || ''
  const receive = matchNearLabel(rawText, LABEL_RECEIVE)
  if (receive) return { paidAt: receive, rawText, source: 'receive' }
  const transfer = matchNearLabel(rawText, LABEL_TRANSFER)
  if (transfer) return { paidAt: transfer, rawText, source: 'transfer' }
  return { paidAt: '', rawText, source: '' }
}
