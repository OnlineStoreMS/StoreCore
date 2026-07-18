/** 从微信支付等截图 OCR 文本中解析付款时间（优先收款时间，其次转账时间） */

const LABEL_RECEIVE = /收\s*款\s*时\s*间/
const LABEL_TRANSFER = /转\s*账\s*时\s*间/

/** 2026年7月18日 20:33:22 / 2026-07-18 20:33:22 / 2026/7/18 20:33 */
const DATETIME_RE =
  /(\d{4})\s*[-/.年]\s*(\d{1,2})\s*[-/.月]\s*(\d{1,2})\s*日?\s*(\d{1,2})\s*[:：]\s*(\d{2})(?:\s*[:：]\s*(\d{2}))?/

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
    // 同行
    const same = line.match(DATETIME_RE)
    if (same) {
      return toLocalDateTimeString(+same[1], +same[2], +same[3], +same[4], +same[5], +(same[6] || 0))
    }
    // 下一行
    if (i + 1 < lines.length) {
      const next = lines[i + 1].match(DATETIME_RE)
      if (next) {
        return toLocalDateTimeString(+next[1], +next[2], +next[3], +next[4], +next[5], +(next[6] || 0))
      }
    }
  }

  // 全文：标签后紧跟时间
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
  // YYYY-MM-DD HH:mm:ss 或 YYYY-MM-DDTHH:mm:ss
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

/** OCR 图片并识别微信等转账截图中的付款时间 */
export async function recognizePayTimeFromImage(
  source: File | Blob | string,
  onProgress?: (p: number) => void,
): Promise<RecognizePayTimeResult> {
  const { default: Tesseract } = await import('tesseract.js')
  const result = await Tesseract.recognize(source, 'chi_sim+eng', {
    logger: (m) => {
      if (m.status === 'recognizing text' && typeof m.progress === 'number') {
        onProgress?.(Math.round(m.progress * 100))
      }
    },
  })
  const rawText = result.data.text || ''
  const receive = matchNearLabel(rawText, LABEL_RECEIVE)
  if (receive) return { paidAt: receive, rawText, source: 'receive' }
  const transfer = matchNearLabel(rawText, LABEL_TRANSFER)
  if (transfer) return { paidAt: transfer, rawText, source: 'transfer' }
  return { paidAt: '', rawText, source: '' }
}
