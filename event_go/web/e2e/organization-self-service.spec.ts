import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

async function selectOption(page: Page, comboboxName: string | RegExp, optionName: string) {
  await page.getByRole('combobox', { name: comboboxName }).press('ArrowDown')
  await page.locator('.el-select-dropdown:visible').getByRole('option', { name: optionName, exact: true }).click()
}

async function register(page: Page, name: string, email: string) {
  await page.goto('/register')
  await completeRegistration(page, name, email)
  await expect(page).toHaveURL(/\/$/)
}

async function completeRegistration(page: Page, name: string, email: string) {
  await page.getByPlaceholder('你的名字').fill(name)
  await page.getByPlaceholder('name@example.com').fill(email)
  await page.getByPlaceholder('8 到 72 位').fill('e2e-password')
  await page.getByRole('button', { name: '注册' }).click()
}

async function logout(page: Page, mobile: boolean) {
  if (mobile) {
    await page.getByRole('button', { name: '打开导航菜单' }).click()
    await page.getByRole('button', { name: '退出登录', exact: true }).click()
  } else {
    await page.getByRole('button', { name: '退出', exact: true }).click()
  }
  await expect(page).toHaveURL(/\/$/)
}

async function invitationLink(request: APIRequestContext, email: string) {
  let link = ''
  await expect.poll(async () => {
    const response = await request.get(`http://127.0.0.1:12526/messages?to=${encodeURIComponent(email)}`)
    const messages = await response.json() as { body: string }[]
    const match = messages.at(-1)?.body.match(/http:\/\/127\.0\.0\.1:18080\/organization-invitations\/accept\?token=[^\s]+/)
    link = match?.[0] || ''
    return link
  }, { timeout: 15_000 }).not.toBe('')
  return link
}

test('organization roles complete a self-service operation without an admin token', async ({ page, request }, testInfo) => {
  test.setTimeout(150_000)
  page.setDefaultTimeout(8_000)
  const suffix = `${testInfo.project.name}-${Date.now()}`.toLowerCase()
  const mobile = testInfo.project.name === 'mobile-chromium'
  const ownerEmail = `owner-${suffix}@example.com`
  const attendeeEmail = `attendee-${suffix}@example.com`
  const editorEmail = `editor-${suffix}@example.com`
  const checkerEmail = `checker-${suffix}@example.com`
  const financeEmail = `finance-${suffix}@example.com`
  const organizationName = `自助组织 ${suffix}`
  const eventTitle = `自助免费活动 ${suffix}`
  const ticketName = `免费票 ${suffix}`

  await register(page, '组织所有者', ownerEmail)
  expect(await page.evaluate(() => localStorage.getItem('admin_token'))).toBeNull()
  await page.goto('/workspace/new')
  await page.getByPlaceholder('组织或团队名称').fill(organizationName)
  await page.getByPlaceholder('例如: soulmark-team').fill(`team-${suffix}`)
  await page.getByPlaceholder('公开展示名称').fill(organizationName)
  await page.getByPlaceholder('公开联系方式').fill(ownerEmail)
  await page.getByRole('button', { name: '创建并进入' }).click()
  await expect(page).toHaveURL(/\/workspace\/\d+$/)
  const organizationId = Number(new URL(page.url()).pathname.split('/').at(-1))

  await page.getByRole('button', { name: '创建活动' }).click()
  await page.getByPlaceholder('活动标题').fill(eventTitle)
  await page.getByPlaceholder('活动描述').fill('自助组织角色权限端到端验证')
  await page.getByPlaceholder('2026-12-31T18:00:00+08:00').fill('2099-12-31T18:00:00+08:00')
  await page.getByPlaceholder('活动地点').fill('自助测试会场')
  await page.getByRole('button', { name: '创建活动' }).click()
  const eventRow = page.getByRole('row').filter({ hasText: eventTitle })
  await eventRow.getByRole('button', { name: '票种' }).click()
  await page.getByPlaceholder('例如：普通票').fill(ticketName)
  await page.getByRole('button', { name: '创建票种' }).click()
  await expect(page.getByRole('row').filter({ hasText: ticketName })).toBeVisible()
  await page.goto(`/workspace/${organizationId}`)
  await eventRow.getByRole('button', { name: '编辑' }).click()
  await selectOption(page, '状态', '已发布')
  await page.getByRole('button', { name: '保存修改' }).click()

  await page.goto(`/workspace/${organizationId}/members`)
  for (const [email, role] of [[editorEmail, '编辑'], [checkerEmail, '核销'], [financeEmail, '财务']] as const) {
    const emailInput = page.getByPlaceholder('member@example.com')
    await emailInput.fill(email)
    await selectOption(page, '角色', role)
    await page.getByRole('button', { name: '发送邀请' }).click()
    await expect(emailInput).toHaveValue('')
    await expect(page.getByText('邀请已发送').last()).toBeVisible()
  }
  const editorLink = await invitationLink(request, editorEmail)
  const checkerLink = await invitationLink(request, checkerEmail)
  const financeLink = await invitationLink(request, financeEmail)

  await logout(page, mobile)
  await register(page, '活动参与者', attendeeEmail)
  await page.getByText(eventTitle, { exact: true }).click()
  await page.getByText(ticketName, { exact: true }).click()
  await page.getByRole('button', { name: '立即报名' }).click()
  const credentialCode = (await page.locator('.code').textContent())?.trim() || ''
  expect(credentialCode).toMatch(/^[0-9a-f]{32}$/)
  await logout(page, mobile)

  await page.goto(editorLink)
  await expect(page).toHaveURL(/\/login\?redirect=/)
  await page.getByRole('link', { name: '去注册' }).click()
  await expect(page).toHaveURL(/\/register\?redirect=/)
  await completeRegistration(page, '组织编辑', editorEmail)
  await expect(page.getByText('已加入组织', { exact: true })).toBeVisible()
  await page.goto(`/workspace/${organizationId}`)
  await expect(page.getByRole('link', { name: '成员' })).toHaveCount(0)
  await eventRow.getByRole('button', { name: '编辑' }).click()
  await page.getByPlaceholder('活动描述').fill('编辑角色已完成活动更新')
  await page.getByRole('button', { name: '保存修改' }).click()
  await expect(page.getByRole('button', { name: '报名 / 核销' })).toHaveCount(0)
  await logout(page, mobile)

  await register(page, '现场核销员', checkerEmail)
  await page.goto(checkerLink)
  await expect(page.getByText('已加入组织', { exact: true })).toBeVisible()
  await page.goto(`/workspace/${organizationId}`)
  await expect(page.getByRole('button', { name: '编辑' })).toHaveCount(0)
  await page.getByRole('button', { name: '核销', exact: true }).click()
  await expect(page.getByRole('heading', { name: '入场核销', exact: true }).first()).toBeVisible()
  await expect(page.getByText(attendeeEmail, { exact: true })).toHaveCount(0)
  await page.getByPlaceholder('soulmark:admission:...').fill(credentialCode)
  await page.getByRole('button', { name: '核销', exact: true }).click()
  await expect(page.getByText('核销成功')).toBeVisible()
  await logout(page, mobile)

  await register(page, '财务人员', financeEmail)
  await page.goto(financeLink)
  await expect(page.getByText('已加入组织', { exact: true })).toBeVisible()
  await page.goto(`/workspace/${organizationId}`)
  await expect(page.getByRole('button', { name: '编辑' })).toHaveCount(0)
  await expect(page.getByRole('link', { name: '成员' })).toHaveCount(0)
  await page.getByRole('button', { name: '报名 / 导出' }).click()
  await expect(page.getByText(attendeeEmail, { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: '导出 CSV' })).toBeVisible()
  await expect(page.getByPlaceholder('soulmark:admission:...')).toHaveCount(0)
  expect(await page.evaluate(() => localStorage.getItem('admin_token'))).toBeNull()
})
