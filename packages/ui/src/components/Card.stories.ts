import type { Meta, StoryObj } from '@storybook/vue3-vite'
import Card from './Card.vue'
const meta = { title: 'Feedback/Card', component: Card, tags: ['autodocs'], args: {"title":"应用资料","description":"独立的标题、内容和操作区域。"}, render: (args) => ({ components: { Card }, setup: () => ({ args }), template: '<Card v-bind="args"><p class="text-sm">这里是展示内容。</p></Card>' }) } satisfies Meta<typeof Card>
export default meta
type Story = StoryObj<typeof meta>
export const Default: Story = {}
export const Plain: Story = { args: { title: undefined, description: undefined } }
