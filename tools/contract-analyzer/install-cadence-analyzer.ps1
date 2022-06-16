<#
.Synopsis
    Install the Cadence Analyzer on Windows.
.DESCRIPTION
    By default, the latest release will be installed.
    If '-Version' is specified, then the given version is installed.
.Parameter Directory
    The destination path to install to.
.Parameter Version
    The version to install.
.Parameter AddToPath
    Add the absolute destination path to the 'User' scope environment variable 'Path'.
.EXAMPLE
    Install the current version
    .\install-cadence-analyzer.ps1
.EXAMPLE
    Install version v0.1
    .\install-cadence-analyzer.ps1 -Version v0.1
.EXAMPLE
    Invoke-Expression "& { $(Invoke-RestMethod 'https://storage.googleapis.com/flow-cli/install-cadence-analyzer.ps1') }"
#>
param (
    [string] $version="",
    [string] $directory = "$env:APPDATA\Cadence",
    [bool] $addToPath = $true
)

Set-StrictMode -Version 3.0

# Enable support for ANSI escape sequences
Set-ItemProperty HKCU:\Console VirtualTerminalLevel -Type DWORD 1

$ErrorActionPreference = "Stop"

$baseURL = "https://storage.googleapis.com/flow-cli"

if (!$version) {
    $version = (Invoke-WebRequest -Uri "$baseURL/cadence-analyzer-version.txt" -UseBasicParsing).Content.Trim()
}

Write-Output("Installing version {0} ..." -f $version)

New-Item -ItemType Directory -Force -Path $directory | Out-Null

$progressPreference = 'silentlyContinue'

Invoke-WebRequest -Uri "$baseURL/cadence-analyzer-x86_64-windows-$version" -UseBasicParsing -OutFile "$directory\cadence-analyzer.exe"

if ($addToPath) {
    Write-Output "Adding to PATH ..."
    $newPath = $Env:Path + ";$directory"
    [System.Environment]::SetEnvironmentVariable("PATH", $newPath)
    [System.Environment]::SetEnvironmentVariable("PATH", $newPath, [System.EnvironmentVariableTarget]::User)
}

Write-Output "Done."

Start-Sleep -Seconds 1