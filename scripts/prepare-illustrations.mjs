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
  const portrait = await readFile(path.join(root, 'output/illustrations/2026-10-23-second-batch/mascot-face-neutral.png'))
  const icons = await page.evaluate(async base64 => {
    const image = new Image()
    image.src = `data:image/png;base64,${base64}`
    await image.decode()
    return [16, 32, 48, 64, 180].map(size => {
      const canvas = document.createElement('canvas')
      canvas.width = size; canvas.height = size
      const context = canvas.getContext('2d')
      context.fillStyle = '#e4f4ed'
      context.beginPath(); context.roundRect(0, 0, size, size, Math.round(size * .22)); context.fill(); context.clip()
      context.imageSmoothingQuality = 'high'
      const inset = Math.max(1, Math.round(size * .04))
      context.drawImage(image, inset, inset, size - inset * 2, size - inset * 2)
      return { size, data: canvas.toDataURL('image/png').split(',')[1] }
    })
  }, portrait.toString('base64'))
  const publicRoot = path.join(root, 'packages/portal/public')
  await mkdir(path.join(publicRoot, 'brand'), { recursive: true })
  const frames = icons.filter(icon => icon.size < 180).map(icon => ({ size: icon.size, data: Buffer.from(icon.data, 'base64') }))
  const directory = Buffer.alloc(6 + frames.length * 16)
  directory.writeUInt16LE(1, 2); directory.writeUInt16LE(frames.length, 4)
  let offset = directory.length
  for (const [index, frame] of frames.entries()) {
    const entry = 6 + index * 16
    directory[entry] = frame.size; directory[entry + 1] = frame.size
    directory.writeUInt16LE(1, entry + 4); directory.writeUInt16LE(32, entry + 6)
    directory.writeUInt32LE(frame.data.length, entry + 8); directory.writeUInt32LE(offset, entry + 12)
    offset += frame.data.length
  }
  await writeFile(path.join(publicRoot, 'favicon.ico'), Buffer.concat([directory, ...frames.map(frame => frame.data)]))
  for (const icon of icons.filter(icon => [32, 48, 180].includes(icon.size))) {
    await writeFile(path.join(publicRoot, 'brand', icon.size === 180 ? 'apple-touch-icon.png' : `favicon-${icon.size}.png`), Buffer.from(icon.data, 'base64'))
  }
  console.log('Site icons: 16/32/48/64 ICO, 32/48 PNG and 180px Apple touch icon')
} finally { await browser.close() }
await writeFile(path.join(target, 'manifest.json'), JSON.stringify({ encoder: 'Chromium Canvas WebP quality 0.88', entries }, null, 2) + '\n')
const imports = entries.map((entry, i) => `import art${i} from './assets/illustrations/${entry.file}'`).join('\n')
const records = entries.map((entry, i) => `  '${entry.key}': { src: art${i}, width: ${entry.width}, height: ${entry.height} },`).join('\n')
await writeFile(path.join(root, 'packages/ui/src/illustrations.ts'), `${imports}\n\nexport const illustrations = {\n${records}\n} as const\n\nexport type MoeIllustrationName = keyof typeof illustrations\n`)
