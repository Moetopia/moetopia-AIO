import type { StorybookConfig } from '@storybook/vue3-vite'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'

const require = createRequire(import.meta.url)
const sharedSource = dirname(require.resolve('@moetopia/ui'))

const config: StorybookConfig = {
  framework: '@storybook/vue3-vite',
  stories: [join(sharedSource, '**/*.stories.ts').replaceAll('\\', '/')],
  addons: ['@storybook/addon-docs', '@storybook/addon-a11y'],
  viteFinal: async (config) => ({ ...config, plugins: [...(config.plugins ?? []), vue(), tailwindcss()] }),
}
export default config
