# Tsuzuri installer for Windows (PowerShell 5.1+ or PowerShell 7).
#
#   irm https://raw.githubusercontent.com/jaisuriya-11/tsuzuri/main/install.ps1 | iex
#
# Environment overrides: TSUZURI_VERSION (e.g. v0.1.0), TSUZURI_INSTALL_DIR,
# TSUZURI_BASE_URL.
$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$repo = 'jaisuriya-11/tsuzuri'

# One x64 build: it also runs on Windows on ARM through emulation.
if ($env:PROCESSOR_ARCHITECTURE -eq 'x86' -and -not $env:PROCESSOR_ARCHITEW6432) {
    throw 'Tsuzuri needs 64-bit Windows.'
}
$archive = 'tsuzuri-windows.zip'

$version = $env:TSUZURI_VERSION
$base = $env:TSUZURI_BASE_URL
if (-not $base) {
    if ($version) { $base = "https://github.com/$repo/releases/download/$version" }
    else { $base = "https://github.com/$repo/releases/latest/download"; $version = 'latest' }
}

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("tsuzuri-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading Tsuzuri ($version) for Windows..."
    $zip = Join-Path $tmp $archive
    Invoke-WebRequest "$base/$archive" -OutFile $zip -UseBasicParsing

    try {
        $sums = (Invoke-WebRequest "$base/checksums.txt" -UseBasicParsing).Content
        $line = ($sums -split "`n") | Where-Object { $_ -match [regex]::Escape($archive) + '\s*$' } | Select-Object -First 1
        if ($line) {
            $want = ($line -split '\s+')[0].ToLower()
            $got = (Get-FileHash $zip -Algorithm SHA256).Hash.ToLower()
            if ($want -ne $got) { throw "Checksum mismatch for $archive" }
        }
    } catch [System.Net.WebException] { }

    Expand-Archive $zip -DestinationPath $tmp -Force

    $dir = $env:TSUZURI_INSTALL_DIR
    if (-not $dir) { $dir = Join-Path $env:LOCALAPPDATA 'Programs\tsuzuri' }
    New-Item -ItemType Directory -Path $dir -Force | Out-Null
    Copy-Item (Join-Path $tmp 'tsuzuri.exe') (Join-Path $dir 'tsuzuri.exe') -Force

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not (($userPath -split ';') -contains $dir)) {
        [Environment]::SetEnvironmentVariable('Path', ($userPath.TrimEnd(';') + ';' + $dir).TrimStart(';'), 'User')
        $env:Path = "$env:Path;$dir"
        Write-Host "Added $dir to your PATH (open a new terminal to use it everywhere)."
    }

    $ver = & (Join-Path $dir 'tsuzuri.exe') --version
    Write-Host "Installed $ver to $dir\tsuzuri.exe"
    Write-Host "Run 'tsuzuri' in a folder of notes. Windows Terminal with a Nerd Font is recommended."
} finally {
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
