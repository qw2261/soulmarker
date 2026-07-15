import type { Registration } from '@/api/types'

const spreadsheetFormulaPrefix = /^[=+\-@]/

function csvCell(value: unknown) {
  let text = value == null ? '' : String(value)
  if (spreadsheetFormulaPrefix.test(text)) text = `'${text}`
  return `"${text.replace(/"/g, '""')}"`
}

export function buildRegistrationCSV(registrations: Registration[]) {
  const rows = [
    ['报名 ID', '姓名', '联系方式', '票种', '身份状态', '报名时间'],
    ...registrations.map((item) => [
      item.id,
      item.name,
      item.contact,
      item.ticket_name || '',
      item.identity_status || '',
      item.created_at,
    ]),
  ]
  return `\uFEFF${rows.map((row) => row.map(csvCell).join(',')).join('\r\n')}\r\n`
}

export function downloadRegistrationCSV(registrations: Registration[], eventId: number) {
  const blob = new Blob([buildRegistrationCSV(registrations)], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `event-${eventId}-registrations.csv`
  anchor.style.display = 'none'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 0)
}
