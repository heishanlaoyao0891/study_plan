param(
  [string[]]$Changes = @('quality-and-testing')
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

Push-Location $PSScriptRoot
try {
  Push-Location ..
  try {
    Push-Location backend
    try {
      go test ./...
      go build -o study_plan_backend .
    } finally {
      Pop-Location
    }

    Push-Location frontend
    try {
      npm.cmd run type-check
      npm.cmd run build:mp-weixin
    } finally {
      Pop-Location
    }

    if (Test-Path -LiteralPath "admin\package.json") {
      Push-Location admin
      try {
        npm.cmd run type-check
        npm.cmd run build
      } finally {
        Pop-Location
      }
    }

    foreach ($Change in $Changes) {
      openspec validate $Change --type change --strict --json --no-interactive
    }
  } finally {
    Pop-Location
  }
} finally {
  Pop-Location
}
