# Registers a daily Windows scheduled task that runs cert-watcher.
# ASCII only on purpose: Windows PowerShell 5.1 misreads BOM-less UTF-8
# as ANSI, which corrupts non-ASCII comments. Japanese guidance lives in
# INSTALL-windows.md.
#
# Run from an elevated (Administrator) PowerShell:
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\register-task.ps1
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\register-task.ps1 -Time 07:30 -CheckMode both
# Remove the task:
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\register-task.ps1 -Unregister
#
# Limitation: registered without stored credentials, so the task runs only
# while this user is logged on (a locked screen is fine, signed out is not).

param(
    [string]$TaskName   = "cert-watcher",
    [string]$InstallDir = "C:\tools\cert-watcher",
    [string]$Time       = "08:00",
    [ValidateSet("edge", "origin", "both")]
    [string]$CheckMode  = "edge",
    # When set, replaces the auto-built argument line entirely.
    [string]$Arguments  = "",
    [switch]$Unregister
)

$ErrorActionPreference = "Stop"

if ($Unregister) {
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    Write-Host "Removed scheduled task '$TaskName'."
    exit 0
}

$exePath = Join-Path $InstallDir "cert-watcher.exe"
if (-not (Test-Path $exePath)) {
    throw "cert-watcher.exe not found at $exePath. Pass -InstallDir pointing to the extracted folder."
}

if ($Arguments -eq "") {
    $Arguments = "check --sites config\sites.csv --report-root output\CertWatcher --check $CheckMode"

    if ($CheckMode -ne "edge") {
        $originsCsv = Join-Path $InstallDir "config\origins.secure.csv"
        if (-not (Test-Path $originsCsv)) {
            throw "config\origins.secure.csv is required for -CheckMode $CheckMode. Copy config\origins.secure.example.csv and fill it in first."
        }
        $Arguments = "check --sites config\sites.csv --origins config\origins.secure.csv --report-root output\CertWatcher --check $CheckMode"
    }
    if (Test-Path (Join-Path $InstallDir "config\settings.json")) {
        $Arguments += " --config config\settings.json"
    }
    if (Test-Path (Join-Path $InstallDir "config\action_status.csv")) {
        $Arguments += " --actions config\action_status.csv"
    }
}

$action   = New-ScheduledTaskAction -Execute $exePath -Argument $Arguments -WorkingDirectory $InstallDir
$trigger  = New-ScheduledTaskTrigger -Daily -At $Time
$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Hours 1)

Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings `
    -Description "Daily TLS certificate expiry check (cert-watcher)" -Force | Out-Null

Write-Host "Registered scheduled task '$TaskName'."
Write-Host "  Schedule : daily at $Time (missed runs start once the PC is available)"
Write-Host "  Command  : $exePath $Arguments"
Write-Host "  Reminder : runs only while this user is logged on (lock is OK, sign-out is not)."
