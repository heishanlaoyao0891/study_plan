param(
  [string]$Source = "study_plan.db",
  [string]$BackupDir = "backups"
)

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$targetDir = Join-Path $BackupDir $timestamp
New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
Copy-Item -LiteralPath $Source -Destination (Join-Path $targetDir "study_plan.db")
Write-Output "Backup written to $targetDir"
