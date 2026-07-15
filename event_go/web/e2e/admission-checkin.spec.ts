import { expect, test, type Page } from '@playwright/test'
import { readFile } from 'node:fs/promises'

async function expectNoHorizontalOverflow(page: Page) {
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
}

async function logoutUser(page: Page, mobile: boolean) {
  if (mobile) {
    const mobileMenu = page.getByRole('button', { name: '打开导航菜单' })
    await expect(mobileMenu).toBeVisible()
    await mobileMenu.click()
    await expect(page.getByRole('heading', { name: '导航' })).toBeVisible()
    await page.getByRole('button', { name: '退出登录', exact: true }).click()
    return
  }
  await page.getByRole('button', { name: '退出', exact: true }).click()
}

test('operator and attendee can complete the free event workflow', async ({ page }, testInfo) => {
  test.setTimeout(120_000)
  page.setDefaultTimeout(5_000)
  const suffix = `${testInfo.project.name}-${Date.now()}`
  const organizerName = `E2E 门店 ${suffix}`
  const disposableOrganizer = `E2E 临时门店 ${suffix}`
  const eventTitle = `E2E 免费活动 ${suffix}`
  const ticketName = `E2E 免费票 ${suffix}`
  const disposableTicket = `E2E 临时票 ${suffix}`
  const discussionUserEmail = `discussion-${suffix}@example.com`
  const userEmail = `checkin-${suffix}@example.com`
  const postTitle = `E2E 讨论 ${suffix}`
  const replyContent = `E2E 回复 ${suffix}`
  const coverURL = 'https://assets.example.com/event-cover.png'
  const mobile = testInfo.project.name === 'mobile-chromium'

  await page.route(coverURL, async (route) => {
    await route.fulfill({
      contentType: 'image/png',
      body: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Wl2nWQAAAAASUVORK5CYII=', 'base64'),
    })
  })

  await page.goto('/admin/events')
  await expect(page).toHaveURL(/\/admin\?redirect=/)
  await page.getByPlaceholder('请输入 ADMIN_TOKEN').fill('e2e-admin-token')
  await page.getByRole('button', { name: '登录' }).click()
  await expect(page).toHaveURL(/\/admin\/events$/)

  const adminNavigation = page.getByRole('navigation', { name: '运营后台导航' })
  await adminNavigation.getByRole('link', { name: '门店' }).click()
  await page.getByRole('button', { name: '创建门店' }).click()
  await page.getByPlaceholder('门店或主办方名称').fill(organizerName)
  await page.getByPlaceholder('邮箱或电话').fill('operator@example.com')
  await page.getByPlaceholder('线下地址').fill('E2E 初始地址')
  await page.getByRole('button', { name: '创建门店' }).click()
  const organizerRow = page.getByRole('row').filter({ hasText: organizerName })
  await expect(organizerRow).toContainText('E2E 初始地址')
  await organizerRow.getByRole('button', { name: '编辑' }).click()
  await page.getByPlaceholder('线下地址').fill('E2E 更新地址')
  await page.getByRole('button', { name: '保存修改' }).click()
  await expect(page.getByRole('row').filter({ hasText: organizerName })).toContainText('E2E 更新地址')

  await page.getByRole('button', { name: '创建门店' }).click()
  await page.getByPlaceholder('门店或主办方名称').fill(disposableOrganizer)
  await page.getByRole('button', { name: '创建门店' }).click()
  const disposableOrganizerRow = page.getByRole('row').filter({ hasText: disposableOrganizer })
  await disposableOrganizerRow.getByRole('button', { name: '删除' }).click()
  await page.getByRole('button', { name: '确定' }).click()
  await expect(disposableOrganizerRow).toHaveCount(0)

  await adminNavigation.getByRole('link', { name: '活动' }).click()
  await page.getByRole('button', { name: '创建活动' }).click()
  await page.getByRole('combobox', { name: /门店/ }).press('ArrowDown')
  await page.getByRole('option', { name: organizerName }).click()
  await page.getByPlaceholder('活动标题').fill(eventTitle)
  await page.getByPlaceholder('活动描述').fill('Admission、导出与核销旅程')
  await page.getByPlaceholder('https://example.com/event-cover.jpg').fill(coverURL)
  await page.getByPlaceholder('2026-12-31T18:00:00+08:00').fill('2099-12-31T18:00:00+08:00')
  await page.getByPlaceholder('活动地点').fill('E2E 会场')
  await page.getByRole('button', { name: '创建活动' }).click()
  const eventRow = page.getByRole('row').filter({ hasText: eventTitle })
  await expect(eventRow).toContainText(organizerName)
  await eventRow.getByRole('button', { name: '票种' }).click()

  await page.getByPlaceholder('例如：普通票').fill(ticketName)
  await page.getByRole('button', { name: '创建票种' }).click()
  let ticketRow = page.getByRole('row').filter({ hasText: ticketName })
  await expect(ticketRow).toContainText('20')
  await ticketRow.getByRole('button', { name: '编辑' }).click()
  await page.getByRole('spinbutton').nth(1).fill('25')
  await page.getByRole('button', { name: '保存票种' }).click()
  ticketRow = page.getByRole('row').filter({ hasText: ticketName })
  await expect(ticketRow).toContainText('25')

  await page.getByPlaceholder('例如：普通票').fill(disposableTicket)
  await page.getByRole('button', { name: '创建票种' }).click()
  const disposableTicketRow = page.getByRole('row').filter({ hasText: disposableTicket })
  await disposableTicketRow.getByRole('button', { name: '删除' }).click()
  await page.getByRole('button', { name: '确定' }).click()
  await expect(disposableTicketRow).toHaveCount(0)

  await page.goto('/register')
  await page.getByPlaceholder('你的名字').fill('E2E 讨论用户')
  await page.getByPlaceholder('name@example.com').fill(discussionUserEmail)
  await page.getByPlaceholder('8 到 72 位').fill('e2e-password')
  await page.getByRole('button', { name: '注册' }).click()
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByAltText(`${eventTitle}活动封面`)).toBeVisible()
  await expectNoHorizontalOverflow(page)

  await page.getByText(eventTitle, { exact: true }).click()
  await expect(page.getByAltText(`${eventTitle}活动封面`)).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await page.getByText(ticketName, { exact: true }).click()
  await page.getByRole('button', { name: '立即报名' }).click()
  await expect(page.getByAltText('入场凭证二维码')).toBeVisible()

  await page.getByRole('button', { name: '去讨论区' }).click()
  await expectNoHorizontalOverflow(page)
  await page.getByRole('button', { name: '发帖' }).click()
  await page.getByPlaceholder('帖子标题').fill(postTitle)
  await page.getByPlaceholder('说点什么...').fill('端到端用户讨论内容')
  await page.getByRole('button', { name: '发布', exact: true }).click()
  const postCard = page.locator('.post-card').filter({ hasText: postTitle })
  await expect(postCard).toBeVisible()
  await postCard.click()
  await expectNoHorizontalOverflow(page)
  await page.getByPlaceholder('写下你的回复...').fill(replyContent)
  await page.getByRole('button', { name: '回复', exact: true }).click()
  await expect(page.getByText(replyContent, { exact: true })).toBeVisible()
  await testInfo.attach('attendee-discussion-reply', {
    body: await page.screenshot({ fullPage: true }),
    contentType: 'image/png',
  })
  await page.getByRole('button', { name: /返回讨论区/ }).click()
  await page.getByRole('button', { name: /返回活动/ }).click()
  await page.getByRole('button', { name: '取消报名' }).click()
  await page.getByRole('button', { name: '确定' }).click()
  await expect(page.getByRole('button', { name: '立即报名' })).toBeVisible()

  await page.goto('/me/registrations')
  await expect(page.getByText(eventTitle, { exact: true })).toBeVisible()
  await expect(page.getByText('已取消', { exact: true }).first()).toBeVisible()
  await expectNoHorizontalOverflow(page)
  await testInfo.attach('attendee-cancelled-activity', {
    body: await page.screenshot({ fullPage: true }),
    contentType: 'image/png',
  })
  await logoutUser(page, mobile)
  await expect(page).toHaveURL(/\/$/)

  await page.goto('/register')
  await page.getByPlaceholder('你的名字').fill('E2E 核销用户')
  await page.getByPlaceholder('name@example.com').fill(userEmail)
  await page.getByPlaceholder('8 到 72 位').fill('e2e-password')
  await page.getByRole('button', { name: '注册' }).click()
  await page.getByText(eventTitle, { exact: true }).click()
  await page.getByText(ticketName, { exact: true }).click()
  await page.getByRole('button', { name: '立即报名' }).click()
  const credentialCode = page.locator('.code')
  await expect(page.getByAltText('入场凭证二维码')).toBeVisible()
  await expect(credentialCode).toHaveText(/^[0-9a-f]{32}$/)
  const code = await credentialCode.textContent()
  expect(code).toBeTruthy()

  await page.goto('/admin/events')
  const managedEventRow = page.getByRole('row').filter({ hasText: eventTitle })
  await managedEventRow.getByRole('button', { name: '报名 / 核销' }).click()

  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: '导出 CSV' }).click()
  const download = await downloadPromise
  expect(download.suggestedFilename()).toMatch(/^event-\d+-registrations\.csv$/)
  const downloadPath = await download.path()
  expect(downloadPath).toBeTruthy()
  const csv = await readFile(downloadPath!, 'utf8')
  expect(csv).toContain(userEmail)
  expect(csv).toContain(ticketName)

  const checkinInput = page.getByPlaceholder('soulmark:admission:...')
  await checkinInput.fill(code!)
  await page.getByRole('button', { name: '核销' }).click()
  await expect(page.getByText('核销成功', { exact: true })).toBeVisible()

  await checkinInput.fill(code!)
  await page.getByRole('button', { name: '核销' }).click()
  await expect(page.getByText('该凭证此前已核销，未重复记录')).toBeVisible()
  await page.getByRole('tab', { name: /核销记录/ }).click()
  await expect(page.getByRole('row').filter({ hasText: 'E2E 核销用户' })).toHaveCount(1)
  await testInfo.attach('operator-checkin-audit', {
    body: await page.screenshot({ fullPage: true }),
    contentType: 'image/png',
  })

  await page.goto('/me/registrations')
  await expect(page.getByText(eventTitle, { exact: true })).toBeVisible()
  await expect(page.getByText('已入场', { exact: true }).first()).toBeVisible()
  await expect(page.getByAltText('入场凭证二维码')).toBeVisible()

  await page.evaluate(() => localStorage.setItem('user_token', 'expired-e2e-token'))
  await page.reload()
  await expect(page).toHaveURL(/\/login\?reason=expired&redirect=/)
  await page.getByPlaceholder('手机号或邮箱').fill(userEmail)
  await page.getByPlaceholder('输入密码').fill('e2e-password')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await expect(page).toHaveURL(/\/me\/registrations$/)
  await expect.poll(() => page.evaluate(() => localStorage.getItem('user_token'))).not.toBeNull()

  await logoutUser(page, mobile)
  await expect(page).toHaveURL(/\/$/)
  await page.goto('/me/registrations')
  await expect(page).toHaveURL(/\/login\?.*redirect=/)

  await page.goto('/admin/events')
  if (mobile) {
    await page.getByRole('button', { name: '打开导航菜单' }).click()
    await page.getByRole('button', { name: '退出管理', exact: true }).click()
  } else {
    await page.getByRole('button', { name: '退出管理', exact: true }).click()
  }
  await page.goto('/admin/events')
  await expect(page).toHaveURL(/\/admin\?redirect=/)

  await page.goto('/forgot-password')
  await page.getByPlaceholder('name@example.com').fill(`missing-${suffix}@example.com`)
  await page.getByRole('button', { name: '发送重置链接' }).click()
  await expect(page.getByText('请检查邮箱')).toBeVisible()
})

test('user pages expose retry and offline feedback', async ({ page, context }) => {
  let shouldFail = true
  await page.route('**/api/v1/events?**', async (route) => {
    if (shouldFail) {
      shouldFail = false
      await route.abort('internetdisconnected')
      return
    }
    await route.continue()
  })

  await page.goto('/')
  await expect(page.getByText('加载失败', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: '重新加载' }).click()
  await expect(page.getByText('加载失败', { exact: true })).toHaveCount(0)

  await page.getByPlaceholder('搜索活动标题或描述...').fill(`missing-${Date.now()}`)
  await page.getByRole('button', { name: '搜索', exact: true }).click()
  await expect(page.getByText('没有符合筛选条件的活动')).toBeVisible()

  await page.goto('/route-that-does-not-exist')
  await expect(page.getByText('404', { exact: true })).toBeVisible()
  await expect(page.getByText('页面不存在')).toBeVisible()
  await page.getByRole('button', { name: '返回首页' }).click()

  await context.setOffline(true)
  await expect(page.getByText('网络连接已断开，恢复连接后可重新加载')).toBeVisible()
  await context.setOffline(false)
  await expect(page.getByText('网络连接已断开，恢复连接后可重新加载')).toHaveCount(0)
})
