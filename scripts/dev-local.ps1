$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true
Set-Location (Join-Path $PSScriptRoot '..')

if (-not (Test-Path .env)) {
    node -e "const fs = require('node:fs'); const { randomBytes } = require('node:crypto'); const config = fs.readFileSync('.env.example', 'utf8').replaceAll('replace-with-a-local-only-password', randomBytes(32).toString('hex')).replaceAll('replace-with-a-valkey-only-password', randomBytes(32).toString('hex')).replace(/^MOETOPIA_ENCRYPTION_KEY=/m, 'MOETOPIA_ENCRYPTION_KEY=' + randomBytes(32).toString('base64')); fs.writeFileSync('.env', config, { flag: 'wx' })"
}

node --env-file=.env -e "const db = new URL(process.env.MOETOPIA_DATABASE_URL); if (process.env.MOETOPIA_ENV !== 'development' || db.hostname !== '127.0.0.1' || db.port !== '55432' || db.pathname !== '/moetopia_dev' || process.env.MOETOPIA_VALKEY_ADDRESS !== '127.0.0.1:56379' || process.env.MOETOPIA_SMTP_ADDRESS !== '127.0.0.1:51025' || process.env.MOETOPIA_DEV_BIND_ADDRESS !== '127.0.0.1') throw Error('The local launcher requires the loopback development configuration in .env')"

$linuxRoot = (wsl.exe -d docker -- pwd).Trim()
$wslSession = Start-Job { wsl.exe -d docker -- sleep infinity }
try {
    wsl.exe -d docker -- env MOETOPIA_DEV_MAILPIT_PORT=58325 docker compose --project-directory $linuxRoot --env-file "$linuxRoot/.env" -f "$linuxRoot/compose.dev.yml" -p moetopia-local-dev up -d --wait
    pnpm api:migrate
    pnpm api:keys:init
    pnpm api:first-party:init
    pnpm api:admin:init
    Write-Host 'Mailpit: http://127.0.0.1:58325'
    pnpm dev @args
} finally {
    Remove-Job -Job $wslSession -Force
}
