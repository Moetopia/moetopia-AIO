import { readdir, readFile, stat } from 'node:fs/promises'
import { resolve, dirname } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const files = ['README.md', 'AGENTS.md', 'CLAUDE.md']
const walk = async (directory) => {
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = resolve(directory, entry.name)
    if (entry.isDirectory()) await walk(path)
    else if (path.endsWith('.md')) files.push(path)
  }
}
await walk(resolve(root, 'docs'))
const errors = []
let links = 0
for (const file of files) {
  const path = resolve(root, file)
  const text = await readFile(path, 'utf8')
  if ((text.match(/^```/gm)?.length ?? 0) % 2) errors.push(`Unbalanced code fence: ${file}`)
  for (const match of text.matchAll(/\[[^\]]+\]\(([^)]+)\)/g)) {
    const target = match[1].split('#')[0]
    if (!target || /^[a-z][a-z\d+.-]*:/i.test(target)) continue
    links++
    try { await stat(resolve(dirname(path), target)) } catch { errors.push(`Broken link: ${file} -> ${target}`) }
  }
}
if (errors.length) { process.stderr.write(`${errors.join('\n')}\n`); process.exitCode = 1 }
else process.stdout.write(`Checked ${files.length} Markdown files and ${links} local links.\n`)
