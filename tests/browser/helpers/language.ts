import type { Page } from '@playwright/test'

export const selectLanguage = async (page: Page, name: string) => {
  await page.getByRole('combobox', { name: /^(语言|語言|Language|言語)$/ }).click()
  await page.getByRole('option', { name, exact: true }).click()
}
