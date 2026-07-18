/** 将分钟格式化为易读文案：不足 1 小时用「xx分钟」，否则「x小时」或「x小时xx分」 */
export function formatDurationMin(minutes?: number | null): string {
  const m = Math.round(Number(minutes) || 0)
  if (m <= 0) return '-'
  if (m < 60) return `${m}分钟`
  const h = Math.floor(m / 60)
  const rest = m % 60
  if (rest === 0) return `${h}小时`
  return `${h}小时${rest}分`
}

/** 带「约」前缀，用于收银台/价目表参考时长 */
export function formatDurationApprox(minutes?: number | null): string {
  const text = formatDurationMin(minutes)
  if (text === '-') return text
  return `约 ${text}`
}
