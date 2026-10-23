import { chromium } from '@playwright/test'
import { createHash } from 'node:crypto'
import { readFile, writeFile, mkdir } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const root = fileURLToPath(new URL('../', import.meta.url))
const batches = [
  ['2026-10-23-first-batch', ['mascot-auth-welcome', 'state-not-found', 'state-no-applications', 'state-no-results', 'state-service-unavailable']],
  ['2026-10-23-second-batch', ['mascot-developer-welcome', 'state-access-restricted', 'state-session-expired', 'state-no-records', 'state-complete', 'mascot-guide-helper', 'mascot-face-neutral']],
]
const target = path.join(root, 'packages/ui/src/assets/illustrations')
await mkdir(target, { recursive: true })
const browser = await chromium.launch()
const entries = []
try {
  const page = await browser.newPage()
  for (const [batch, names] of batches) {
    for (const name of names) {
      const source = `output/illustrations/${batch}/${name}.png`
      const input = await readFile(path.join(root, source))
      const limit = name === 'mascot-auth-welcome' ? 1100 : name === 'mascot-developer-welcome' ? 900 : name === 'mascot-face-neutral' ? 192 : 480
      const result = await page.evaluate(async ({ base64, limit }) => {
        const image = new Image()
        image.src = `data:image/png;base64,${base64}`
        await image.decode()
        const scale = Math.min(1, limit / Math.max(image.naturalWidth, image.naturalHeight))
        const canvas = document.createElement('canvas')
        canvas.width = Math.round(image.naturalWidth * scale)
        canvas.height = Math.round(image.naturalHeight * scale)
        const context = canvas.getContext('2d')
        context.imageSmoothingQuality = 'high'
        context.drawImage(image, 0, 0, canvas.width, canvas.height)
        const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data
        let transparent = 0
        let opaque = 0
        for (let i = 3; i < pixels.length; i += 4) { if (pixels[i] === 0) transparent++; if (pixels[i] >= 240) opaque++ }
        return { width: canvas.width, height: canvas.height, transparent, opaque, data: canvas.toDataURL('image/webp', .88).split(',')[1] }
      }, { base64: input.toString('base64'), limit })
      if (!result.transparent || !result.opaque) throw new Error(`Missing transparent background or visible content: ${name}`)
      const output = Buffer.from(result.data, 'base64')
      if (output.toString('ascii', 8, 12) !== 'WEBP') throw new Error(`WebP encoder unavailable: ${name}`)
      await writeFile(path.join(target, `${name}.webp`), output)
      const key = name.replace(/^(mascot|state)-/, '')
      entries.push({ key, file: `${name}.webp`, width: result.width, height: result.height, bytes: output.length, source, sourceBytes: input.length, sourceSHA256: createHash('sha256').update(input).digest('hex'), transparentPixels: result.transparent })
      console.log(`${name}: ${result.width}x${result.height}, ${Math.round(output.length / 1024)} KiB, alpha verified`)
    }
  }
} finally { await browser.close() }
await writeFile(path.join(target, 'manifest.json'), JSON.stringify({ encoder: 'Chromium Canvas WebP quality 0.88', entries }, null, 2) + '\n')
const imports = entries.map((entry, i) => `import art${i} from './assets/illustrations/${entry.file}'`).join('\n')
const records = entries.map((entry, i) => `  '${entry.key}': { src: art${i}, width: ${entry.width}, height: ${entry.height} },`).join('\n')
await writeFile(path.join(root, 'packages/ui/src/illustrations.ts'), `${imports}\n\nexport const illustrations = {\n${records}\n} as const\n\nexport type MoeIllustrationName = keyof typeof illustrations\n`)
