import { describe, expect, it } from 'vitest'
import { buildRegistrationCSV } from './registration-export'

describe('registration CSV export', () => {
  it('quotes fields, preserves UTF-8 and neutralizes spreadsheet formulas', () => {
    const csv = buildRegistrationCSV([{
      id: 7,
      event_id: 3,
      name: '=HYPERLINK("https://bad.example")',
      contact: 'user@example.com',
      ticket_name: '普通票,早鸟',
      identity_status: 'verified',
      created_at: '2030-01-02T03:04:05Z',
    }])

    expect(csv.startsWith('\uFEFF')).toBe(true)
    expect(csv).toContain('"\'=HYPERLINK(""https://bad.example"")"')
    expect(csv).toContain('"普通票,早鸟"')
    expect(csv).toContain('"verified"')
  })
})
