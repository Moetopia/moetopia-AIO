export const oidcAPIPort = (env = process.env) => {
  const raw = env.MOETOPIA_TEST_API_PORT ?? '58490'
  const port = Number(raw)
  if (!/^[0-9]{5}$/.test(raw) || !Number.isInteger(port) || port <= 50000 || port > 65535 || [54305, 54310, 54311].includes(port)) throw new Error('MOETOPIA_TEST_API_PORT must be a dedicated test port between 50001 and 65535, excluding the browser/RP ports')
  return port
}

export const oidcAPIOrigin = (env = process.env) => `http://127.0.0.1:${oidcAPIPort(env)}`
