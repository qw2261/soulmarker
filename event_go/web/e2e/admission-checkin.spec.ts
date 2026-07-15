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
  await page.getByPlaceholder('name@example.com').fill(`${suffix}@example.com`)
  await page.getByPlaceholder('8 到 72 位').fill('e2e-password')
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

  await page.evaluate(() => localStorage.setItem('user_token', 'expired-e2e-token'))
  await page.reload()
  await expect(page).toHaveURL(/\/login\?reason=expired&redirect=/)
  await page.getByPlaceholder('手机号或邮箱').fill(`${suffix}@example.com`)
  await page.getByPlaceholder('输入密码').fill('e2e-password')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await expect(page).toHaveURL(/\/me\/registrations$/)
  await expect.poll(() => page.evaluate(() => localStorage.getItem('user_token'))).not.toBeNull()

  if (testInfo.project.name === 'mobile-chromium') {
    const mobileMenu = page.getByRole('button', { name: '打开导航菜单' })
    await expect(mobileMenu).toBeVisible()
    await mobileMenu.click()
    await expect(page.getByRole('heading', { name: '导航' })).toBeVisible()
    await expect(page.locator('.mobile-nav')).toContainText('退出登录')
    await page.getByRole('button', { name: '退出登录', exact: true }).click()
  } else {
    await page.getByRole('button', { name: '退出', exact: true }).click()
  }
  await expect(page).toHaveURL(/\/$/)
  await page.goto('/me/registrations')
  await expect(page).toHaveURL(/\/login\?.*redirect=/)

  await page.goto('/forgot-password')
  await page.getByPlaceholder('name@example.com').fill(`missing-${suffix}@example.com`)
  await page.getByRole('button', { name: '发送重置链接' }).click()
  await expect(page.getByText('请检查邮箱')).toBeVisible()
})
