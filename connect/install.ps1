# valo-tui — one-line connector for Windows (PowerShell).
#
#   irm https://valo.black-pantha.com/install.ps1 | iex
#
# It installs Cloudflare's cloudflared (needed to reach the tunnel) and writes an
# `ssh valo` shortcut into %USERPROFILE%\.ssh\config. After that, watching
# Valorant esports is just:  ssh valo
#
# It needs no admin rights, installs only cloudflared + one ssh config block, and
# touches nothing else. The server end is a read-only TUI — no shell, no file
# access — so the connection can only ever draw a scoreboard.
#
# Override the host:  $env:VALO_HOST='valo.yourdomain.com'; irm ... | iex

$ErrorActionPreference = 'Stop'

# The public hostname your tunnel is routed to. EDIT THIS to your domain (or bake
# it in when you host the script); users override with $env:VALO_HOST.
$ValoHost = if ($env:VALO_HOST) { $env:VALO_HOST } else { 'valo.black-pantha.com' }
$Alias    = if ($env:VALO_ALIAS) { $env:VALO_ALIAS } else { 'valo' }

function Info($m) { Write-Host $m -ForegroundColor Cyan }
function Ok($m)   { Write-Host $m -ForegroundColor Green }
function Die($m)  { Write-Host "X $m" -ForegroundColor Red; exit 1 }

# ── Locate or install cloudflared ────────────────────────────────────────────
function Find-Cloudflared {
	$c = Get-Command cloudflared -ErrorAction SilentlyContinue
	if ($c) { return $c.Source }
	$local = Join-Path $env:LOCALAPPDATA 'valo-tui\cloudflared.exe'
	if (Test-Path $local) { return $local }
	return $null
}

$Cf = Find-Cloudflared
if ($Cf) {
	Ok "cloudflared already installed ($Cf)"
} else {
	Info 'Installing cloudflared...'
	$winget = Get-Command winget -ErrorAction SilentlyContinue
	if ($winget) {
		winget install --id Cloudflare.cloudflared --silent --accept-source-agreements --accept-package-agreements | Out-Null
		# winget may not refresh PATH in this session; re-probe, then fall back.
		$Cf = Find-Cloudflared
	}
	if (-not $Cf) {
		# Direct download to a per-user dir (no admin needed).
		$arch = if ([Environment]::Is64BitOperatingSystem) { 'amd64' } else { '386' }
		$dir  = Join-Path $env:LOCALAPPDATA 'valo-tui'
		New-Item -ItemType Directory -Force -Path $dir | Out-Null
		$Cf  = Join-Path $dir 'cloudflared.exe'
		$url = "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-windows-$arch.exe"
		Info "Downloading $url"
		Invoke-WebRequest -Uri $url -OutFile $Cf -UseBasicParsing
	}
	if (-not $Cf -or -not (Test-Path $Cf)) { Die 'cloudflared install failed' }
	Ok "cloudflared installed at $Cf"
}

if (-not (Get-Command ssh -ErrorAction SilentlyContinue)) {
	Die "The 'ssh' client is missing. Install it: Settings > Apps > Optional Features > OpenSSH Client."
}

# ── Write a managed block into %USERPROFILE%\.ssh\config ─────────────────────
$sshDir = Join-Path $env:USERPROFILE '.ssh'
$cfg    = Join-Path $sshDir 'config'
New-Item -ItemType Directory -Force -Path $sshDir | Out-Null
if (-not (Test-Path $cfg)) { New-Item -ItemType File -Path $cfg | Out-Null }

$begin = '# >>> valo-tui (managed) >>>'
$end   = '# <<< valo-tui (managed) <<<'

# Strip any previous managed block, then append a fresh one (idempotent).
$lines = Get-Content $cfg -ErrorAction SilentlyContinue
$kept  = New-Object System.Collections.Generic.List[string]
$skip  = $false
foreach ($l in $lines) {
	if ($l -eq $begin) { $skip = $true }
	if (-not $skip) { $kept.Add($l) }
	if ($l -eq $end) { $skip = $false }
}

$known = Join-Path $sshDir 'known_hosts_valo'
$block = @(
	$begin
	"Host $Alias"
	"    HostName $ValoHost"
	"    User viewer"
	"    ProxyCommand `"$Cf`" access ssh --hostname %h"
	"    RequestTTY yes"
	"    StrictHostKeyChecking accept-new"
	"    UserKnownHostsFile `"$known`""
	$end
)
($kept + $block) | Set-Content -Path $cfg -Encoding ascii
Ok "Added 'Host $Alias' to $cfg"

Write-Host ''
# One command only: connect right now. Re-running this same command later just
# reconnects (cloudflared + the ssh config are already in place). Set
# $env:VALO_NO_CONNECT=1 to only set things up without launching.
if ($env:VALO_NO_CONNECT) {
	Ok "Set up. Connect any time with:  ssh $Alias"
	return
}
Ok "Set up - connecting now.  (Next time, run the same command, or just: ssh $Alias)"
Write-Host ''
& ssh $Alias
