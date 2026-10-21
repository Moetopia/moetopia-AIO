import { createServer } from 'node:net'

export const services = {
  api: { port: 8080, description: 'Go API / automatic rebuild' },
  account: { port: 3000, description: 'account.localhost' },
  admin: { port: 3001, description: 'admin.localhost' },
  storybook: { port: 6006, description: 'UI components' },
}

export const selectedServices = (args) => {
  const names = args.filter(value => value !== '--' && value !== '--plain')
  for (const name of names) {
    if (!Object.hasOwn(services, name)) throw new Error(`Unknown service: ${name}. Choose api, account, admin or storybook. Developer pages are served by account at /developer.`)
  }
  return names.length ? [...new Set(names)] : ['api', 'account', 'admin']
}

export const servicePort = (name, env = process.env) => {
  if (name !== 'api') return services[name].port
  const address = env.MOETOPIA_LISTEN ?? '127.0.0.1:8080'
  const port = Number(address.slice(address.lastIndexOf(':') + 1))
  if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error('MOETOPIA_LISTEN must contain a valid explicit port')
  return port
}

export const checkPort = (name, port) => new Promise((resolvePort, reject) => {
  const probe = createServer()
  probe.once('error', (error) => reject(new Error(error.code === 'EADDRINUSE'
    ? `${name}: port ${port} is occupied. Stop your own previous instance in its original terminal; no process was stopped.`
    : `${name}: cannot listen on port ${port} (${error.code ?? 'unknown error'}). Check OS access restrictions or select another port; no process was stopped.`)))
  probe.listen(port, '127.0.0.1', () => probe.close(resolvePort))
})

export const serviceCommand = (name, node, pnpm) => {
  if (name === 'api') return [node, 'scripts/api-watch.mjs']
  const args = name === 'storybook' ? ['--filter', '@moetopia/ui', 'storybook'] : ['--filter', `@moetopia/${name}`, 'dev']
  return /\.[cm]?js$/.test(pnpm) ? [node, pnpm, ...args] : [pnpm, ...args]
}

export const tuiConfig = (root, node, names) => ({
  procs: Object.fromEntries(names.map(name => [name, {
    cmd: [node, 'scripts/dev-service.mjs', name],
    cwd: root,
    autorestart: false,
    stop: { 'send-keys': ['<C-c>'] },
  }])),
  scrollback: 10000,
  proc_list_width: 24,
  proc_log: false,
  keymap_procs: {
    reset: true,
    '<Up>': { c: 'prev-proc' }, '<k>': { c: 'prev-proc' },
    '<Down>': { c: 'next-proc' }, '<j>': { c: 'next-proc' },
    '<Tab>': { c: 'focus-term' }, '<C-a>': { c: 'focus-term' },
    '<r>': { c: 'restart-proc' }, '<s>': { c: 'start-proc' },
    '<x>': { c: 'term-proc' }, '<z>': { c: 'zoom' },
    '<v>': { c: 'copy-mode-enter' },
    '<PageUp>': { c: 'scroll-up' }, '<PageDown>': { c: 'scroll-down' },
    '<q>': { c: 'quit' }, '<C-c>': { c: 'quit' },
  },
  keymap_term: {
    reset: true,
    '<Tab>': { c: 'focus-procs' }, '<C-a>': { c: 'focus-procs' },
    '<C-c>': { c: 'quit' },
  },
})
