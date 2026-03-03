# BlackCat Installer — PowerShell
# Usage: irm https://raw.githubusercontent.com/startower-observability/BlackCat/main/scripts/install.ps1 | iex
#    or: .\install.ps1 -Version v2026.3.1
param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

$Repo = "startower-observability/BlackCat"
$BinaryName = "blackcat"
$InstallDir = Join-Path $env:USERPROFILE ".blackcat\bin"

# --- Detect architecture ------------------------------------------------------
function Get-Arch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "x86_64" }
        "Arm64" { return "arm64" }
        default {
            # Fallback for older PowerShell
            $envArch = $env:PROCESSOR_ARCHITECTURE
            switch ($envArch) {
                "AMD64" { return "x86_64" }
                "ARM64" { return "arm64" }
                default {
                    Write-Error "Unsupported architecture: $envArch"
                    exit 1
                }
            }
        }
    }
}

# --- Resolve version ----------------------------------------------------------
function Get-LatestVersion {
    if ($Version -ne "") {
        return $Version
    }

    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        return $release.tag_name
    } catch {
        Write-Error "Could not determine latest release version: $_"
        exit 1
    }
}

# --- Main ---------------------------------------------------------------------
$Arch = Get-Arch
$Ver = Get-LatestVersion

# GoReleaser naming convention: blackcat_Windows_x86_64.zip
$Archive = "${BinaryName}_Windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/$Ver/$Archive"

Write-Host "Installing BlackCat $Ver (Windows/$Arch)..." -ForegroundColor Cyan

# Create install directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Download
$TempDir = Join-Path $env:TEMP "blackcat-install-$(Get-Random)"
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    $ArchivePath = Join-Path $TempDir $Archive

    Write-Host "  Downloading $Url..."
    Invoke-WebRequest -Uri $Url -OutFile $ArchivePath -UseBasicParsing

    Write-Host "  Extracting..."
    Expand-Archive -Path $ArchivePath -DestinationPath $TempDir -Force

    # Install binary
    $BinaryPath = Join-Path $TempDir "$BinaryName.exe"
    if (-not (Test-Path $BinaryPath)) {
        Write-Error "Binary '$BinaryName.exe' not found in archive"
        exit 1
    }

    $DestPath = Join-Path $InstallDir "$BinaryName.exe"
    Move-Item -Path $BinaryPath -Destination $DestPath -Force
} finally {
    Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}

# Add to user PATH
$CurrentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    $NewPath = "$InstallDir;$CurrentPath"
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    Write-Host "  Added $InstallDir to user PATH" -ForegroundColor Green
}

Write-Host ""
Write-Host "BlackCat $Ver installed to $DestPath" -ForegroundColor Green
Write-Host ""
Write-Host "To get started, open a new terminal and run:" -ForegroundColor Cyan
Write-Host ""
Write-Host "  blackcat onboard" -ForegroundColor White
Write-Host ""
