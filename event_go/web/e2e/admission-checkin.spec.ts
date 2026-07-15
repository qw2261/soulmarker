import { expect, test } from '@playwright/test'

test('free event admission can be viewed and idempotently checked in', async ({ page, request }, testInfo) => {
  const suffix = `${testInfo.project.name}-${Date.now()}`
  const adminHeaders = { 'X-Admin-Token': 'e2e-admin-token' }

  const organizerResponse = await request.post('/api/v1/organizers', {
    headers: adminHeaders,
    data: { name: `E2E 门店 ${suffix}` },
  })
  expect(organizerResponse.status()).toBe(201)
  const organizer = (await organizerResponse.json()).data

  const eventResponse = await request.post('/api/v1/events', {
    headers: adminHeaders,
    data: {
      organizer_id: organizer.id,
      title: `E2E 免费活动 ${suffix}`,
      description: 'Admission 与核销旅程',
      event_time: '2099-12-31T18:00:00+08:00',
      location: 'E2E 会场',
      capacity: 20,
      price: 0,
    },
  })
  expect(eventResponse.status()).toBe(201)

  await page.goto('/register')
  await page.getByPlaceholder('你的名字').fill('E2E 用户')
  await page.getByPlaceholder('手机号或邮箱').fill(`${suffix}@example.com`)
  await page.getByPlaceholder('至少 6 位').fill('e2e-password')
  await page.getByRole('button', { name: '注册' }).click()
  await expect(page).toHaveURL(/\/$/)

  await page.getByText(`E2E 免费活动 ${suffix}`, { exact: true }).click()
  await page.getByRole('button', { name: '立即报名' }).click()
  const credentialCode = page.locator('.code')
  await expect(page.getByAltText('入场凭证二维码')).toBeVisible()
  await expect(credentialCode).toHaveText(/^[0-9a-f]{32}$/)
  const code = await credentialCode.textContent()
  expect(code).toBeTruthy()
  await page.goto('/admin')
  await page.getByPlaceholder('请输入 ADMIN_TOKEN').fill('e2e-admin-token')
  await page.getByRole('button', { name: '登录' }).click()
  const eventRow = page.getByRole('row').filter({ hasText: `E2E 免费活动 ${suffix}` })
  await eventRow.getByRole('button', { name: '报名 / 核销' }).click()

  const checkinInput = page.getByPlaceholder('soulmark:admission:...')
  await checkinInput.fill(code!)
  await page.getByRole('button', { name: '核销' }).click()
  await expect(page.getByText('核销成功', { exact: true })).toBeVisible()

  await checkinInput.fill(code!)
  await page.getByRole('button', { name: '核销' }).click()
  await expect(page.getByText('该凭证此前已核销，未重复记录')).toBeVisible()
  await page.getByRole('tab', { name: /核销记录/ }).click()
  await expect(page.getByRole('row').filter({ hasText: 'E2E 用户' })).toHaveCount(1)

  await page.goto('/me/registrations')
  await expect(page.getByText(`E2E 免费活动 ${suffix}`, { exact: true })).toBeVisible()
  await expect(page.getByText('已入场', { exact: true }).first()).toBeVisible()
  await expect(page.getByAltText('入场凭证二维码')).toBeVisible()
})
