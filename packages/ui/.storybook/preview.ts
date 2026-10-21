import type { Preview } from '@storybook/vue3-vite'
import '../src/styles.css'
import Provider from '../src/components/Provider.vue'

const preview: Preview = {
  globalTypes: { theme: { toolbar: { icon: 'circlehollow', items: ['light', 'dark'] } } },
  initialGlobals: { theme: 'light' },
  decorators: [(story, context) => {
    document.documentElement.dataset.theme = context.globals.theme
    return { components: { story, Provider }, template: '<Provider><div class="bg-background p-6 text-foreground"><story /></div></Provider>' }
  }],
  parameters: { a11y: { test: 'error' }, options: { storySort: { order: ['Overview', 'Foundation', 'Navigation', 'Controls', 'Feedback', 'Overlays'] } } },
}
export default preview
