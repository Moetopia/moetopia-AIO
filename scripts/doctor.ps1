param([switch]$RemoteDependencies, [string]$ServicesHost = '192.168.101.41')

$ErrorActionPreference = 'Continue'
$missing = @()
$tools = @('go', 'node', 'pnpm')
if (-not $RemoteDependencies) { $tools += 'docker' }
foreach ($toolName in $tools) {
  if (Get-Command $toolName -ErrorAction SilentlyContinue) { Write-Host "[ok] $toolName" }
  else { Write-Host "[missing] $toolName"; $missing += $toolName }
}
if ($RemoteDependencies) {
  foreach ($servicePort in @(55432, 56379, 51025, 58025)) {
    $client = [System.Net.Sockets.TcpClient]::new()
    try {
      $connection = $client.ConnectAsync($ServicesHost, $servicePort)
      if (-not $connection.Wait(2000)) { throw 'timeout' }
      Write-Host "[ok] ${ServicesHost}:$servicePort"
    } catch {
      Write-Host "[unavailable] ${ServicesHost}:$servicePort"
      $missing += "port-$servicePort"
    } finally { $client.Dispose() }
  }
  if ($missing.Count -gt 0) { exit 1 }
  Write-Host 'Remote ports are reachable; authentication and schema are checked separately by the API and integration tests.'
  exit 0
}
if ($missing -contains 'docker') {
  Write-Host 'Use pnpm dev:doctor:remote for Ubuntu-hosted dependencies, or install Docker Desktop for local containers.'
  exit 1
}
docker info --format '{{.OSType}}'
if ($LASTEXITCODE -ne 0) { Write-Host 'Docker engine is unavailable. Start Docker Desktop first.'; exit 1 }
if ($missing.Count -gt 0) { exit 1 }
