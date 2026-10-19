import type { Page } from '@playwright/test'
export const chooseTheme = async (page: Page, trigger: string, option: string) => {
  await page.getByRole('button', { name: trigger, exact: true }).click()
  await page.getByRole('menuitem', { name: option, exact: true }).click()
}
