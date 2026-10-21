import { expect, test } from '@playwright/test'
const origin = 'http://127.0.0.1:56006/iframe.html?viewMode=story&id='
test('repeatable inputs preserve values, enforce the limit and remove one entry', async ({ page }) => {
  await page.goto(`${origin}controls-repeatableinput--default`)
  await page.getByLabel('回调地址 1', { exact: true }).fill('https://example.com/one')
  await page.getByRole('button', { name: '添加回调地址' }).click()
  await page.getByLabel('回调地址 2', { exact: true }).fill('https://example.com/two')
  await page.getByRole('button', { name: '添加回调地址' }).click()
  await expect(page.getByRole('button', { name: '添加回调地址' })).toBeDisabled()
  await page.getByRole('button', { name: '移除回调地址 2', exact: true }).click()
  await expect(page.getByRole('textbox')).toHaveCount(2)
  await expect(page.getByLabel('回调地址 1', { exact: true })).toHaveValue('https://example.com/one')
  await expect(page.getByLabel('回调地址 2', { exact: true })).toHaveValue('')
})
test('checkbox cards keep native keyboard interaction and disabled options', async ({ page }) => {
  await page.goto(`${origin}controls-checkboxgroup--cards`)
  await page.getByRole('checkbox', { name: '社区动态' }).focus()
  await page.keyboard.press('Space')
  await expect(page.getByRole('checkbox', { name: '社区动态' })).toBeChecked()
  await expect(page.getByRole('checkbox', { name: '其他通知' })).toBeDisabled()
})
