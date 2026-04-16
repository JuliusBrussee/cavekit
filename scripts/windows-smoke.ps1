[CmdletBinding()]
param(
  [ValidateSet("baseline", "codex-present")]
  [string]$Pass = "baseline",
  [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path,
  [string]$OutputDir = (Join-Path $env:TEMP ("cavekit-windows-smoke-" + (Get-Date -Format "yyyyMMdd-HHmmss"))),
  [switch]$SkipInteractive,
  [switch]$SkipBuildAndTest,
  [switch]$SkipInstall,
  [switch]$NoTranscript
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$LogsDir = Join-Path $OutputDir "logs"
$ScreenshotsDir = Join-Path $OutputDir "screenshots"
$ReportPath = Join-Path $OutputDir "smoke-report.md"
$TranscriptPath = Join-Path $OutputDir "transcript.txt"
$Results = @()
$CommandHistory = @()
$TranscriptStarted = $false
$StateFilePath = ""
$InstallBinDir = Join-Path $env:LOCALAPPDATA "Programs\cavekit\bin"
$InstallBinary = Join-Path $InstallBinDir "cavekit.exe"
$ClaudeSettings = Join-Path $env:USERPROFILE ".claude\settings.json"
$ClaudeMarketplace = Join-Path $env:USERPROFILE ".claude\plugins\local\cavekit-marketplace\.claude-plugin\marketplace.json"
$ClaudePluginManifest = Join-Path $env:USERPROFILE ".claude\plugins\local\cavekit-marketplace\.claude-plugin\plugin.json"
$ClaudePluginCk = Join-Path $env:USERPROFILE ".claude\plugins\local\cavekit-marketplace\ck"
$ClaudePluginBp = Join-Path $env:USERPROFILE ".claude\plugins\local\cavekit-marketplace\bp"
$CodexPluginDir = Join-Path $env:USERPROFILE "plugins\ck"
$CodexMarketplace = Join-Path $env:USERPROFILE ".agents\plugins\marketplace.json"
$CodexPromptsDir = Join-Path $env:USERPROFILE ".codex\prompts"
$WindowsPtyRegistry = Join-Path $env:USERPROFILE ".cavekit\windowspty-sessions.json"
$WindowsPtyLogsDir = Join-Path $env:USERPROFILE ".cavekit\logs"

function Write-Section {
  param([string]$Title)
  Write-Host ""
  Write-Host "== $Title ==" -ForegroundColor Cyan
}

function Sanitize-Name {
  param([string]$Name)
  return ([regex]::Replace($Name.ToLowerInvariant(), "[^a-z0-9]+", "-")).Trim("-")
}

function Add-Result {
  param(
    [string]$Category,
    [string]$Step,
    [string]$Status,
    [string]$Details,
    [string]$LogPath = ""
  )

  $script:Results += [pscustomobject]@{
    Category = $Category
    Step = $Step
    Status = $Status
    Details = $Details
    LogPath = $LogPath
  }
}

function Quote-CommandPart {
  param([string]$Value)
  if ($Value -match '\s') {
    return '"' + $Value.Replace('"', '\"') + '"'
  }
  return $Value
}

function Invoke-LoggedCommand {
  param(
    [string]$Label,
    [string]$FilePath,
    [string[]]$Arguments = @(),
    [string]$WorkingDirectory = $RepoRoot,
    [switch]$AllowFailure
  )

  $logPath = Join-Path $LogsDir ((Sanitize-Name $Label) + ".txt")
  $display = ($FilePath, ($Arguments | ForEach-Object { Quote-CommandPart $_ })) -join " "
  Write-Host "-> $display" -ForegroundColor DarkCyan

  $global:LASTEXITCODE = 0
  Push-Location $WorkingDirectory
  try {
    $outputLines = & $FilePath @Arguments 2>&1
    $exitCode = if ($null -ne $LASTEXITCODE) { [int]$LASTEXITCODE } else { 0 }
  } finally {
    Pop-Location
  }

  $outputText = ($outputLines | ForEach-Object { "$_" }) -join [Environment]::NewLine
  Set-Content -Path $logPath -Value $outputText

  $script:CommandHistory += [pscustomobject]@{
    Label = $Label
    Command = $display
    ExitCode = $exitCode
    LogPath = $logPath
  }

  if (-not $AllowFailure -and $exitCode -ne 0) {
    throw "Command failed with exit code ${exitCode}: $display"
  }

  return [pscustomobject]@{
    Output = $outputText
    ExitCode = $exitCode
    LogPath = $logPath
    Command = $display
  }
}

function Invoke-Step {
  param(
    [string]$Category,
    [string]$Step,
    [scriptblock]$Body
  )

  try {
    $details = & $Body
    if ($null -eq $details -or [string]::IsNullOrWhiteSpace([string]$details)) {
      $details = "Passed."
    }
    Add-Result -Category $Category -Step $Step -Status "PASS" -Details ([string]$details)
  } catch {
    Add-Result -Category $Category -Step $Step -Status "FAIL" -Details $_.Exception.Message
    Write-Warning "$Step failed: $($_.Exception.Message)"
  }
}

function Add-WarningResult {
  param(
    [string]$Category,
    [string]$Step,
    [string]$Details
  )
  Add-Result -Category $Category -Step $Step -Status "WARN" -Details $Details
}

function Add-PendingResult {
  param(
    [string]$Category,
    [string]$Step,
    [string]$Details
  )
  Add-Result -Category $Category -Step $Step -Status "PENDING" -Details $Details
}

function Assert-True {
  param(
    [bool]$Condition,
    [string]$Message
  )
  if (-not $Condition) {
    throw $Message
  }
}

function Read-JsonFile {
  param([string]$Path)
  if (-not (Test-Path $Path)) {
    return $null
  }
  $raw = Get-Content -Path $Path -Raw
  if ([string]::IsNullOrWhiteSpace($raw)) {
    return $null
  }
  return $raw | ConvertFrom-Json
}

function Get-SessionRegistryEntries {
  $data = Read-JsonFile -Path $WindowsPtyRegistry
  if ($null -eq $data) {
    return @()
  }

  return @(
    $data.PSObject.Properties | ForEach-Object {
      [pscustomobject]@{
        Name = $_.Name
        Value = $_.Value
      }
    }
  )
}

function Get-WindowsPtyLogs {
  if (-not (Test-Path $WindowsPtyLogsDir)) {
    return @()
  }
  return @(Get-ChildItem -Path $WindowsPtyLogsDir -Filter *.log -File -ErrorAction SilentlyContinue)
}

function Get-UserAndMachinePath {
  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
  if ([string]::IsNullOrWhiteSpace($userPath)) {
    return $machinePath
  }
  if ([string]::IsNullOrWhiteSpace($machinePath)) {
    return $userPath
  }
  return "$userPath;$machinePath"
}

function Test-PathEntryContains {
  param(
    [string]$PathValue,
    [string]$Expected
  )

  foreach ($entry in ($PathValue -split ";")) {
    if ($entry.Trim().ToLowerInvariant() -eq $Expected.Trim().ToLowerInvariant()) {
      return $true
    }
  }
  return $false
}

function Test-ReparsePoint {
  param([string]$Path)
  if (-not (Test-Path $Path)) {
    return $false
  }
  $item = Get-Item -LiteralPath $Path -Force
  return (($item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0)
}

function Get-CategorySummary {
  param([string]$Category)

  $group = @($script:Results | Where-Object { $_.Category -eq $Category })
  if ($group.Count -eq 0) {
    return "NOT RUN"
  }
  if ($group.Status -contains "FAIL") {
    return "FAIL"
  }
  if ($group.Status -contains "PENDING") {
    return "PENDING"
  }
  if ($group.Status -contains "WARN") {
    return "WARN"
  }
  return "PASS"
}

function Start-SmokeTranscript {
  if ($NoTranscript) {
    return
  }
  Start-Transcript -Path $TranscriptPath -Force | Out-Null
  $script:TranscriptStarted = $true
}

function Stop-SmokeTranscript {
  if ($script:TranscriptStarted) {
    Stop-Transcript | Out-Null
  }
}

function Pause-ForManualStep {
  param([string]$Message)
  if ($SkipInteractive) {
    Write-Host "SKIP INTERACTIVE: $Message" -ForegroundColor Yellow
    return
  }
  Read-Host "$Message`nPress Enter to continue"
}

function Write-Report {
  $lines = @()
  $lines += "# Windows Smoke Report"
  $lines += ""
  $lines += "- Pass: `$Pass = $Pass"
  $lines += "- Repo root: `$RepoRoot = $RepoRoot"
  $lines += "- Output directory: `$OutputDir = $OutputDir"
  $lines += "- Transcript: `$TranscriptPath = $TranscriptPath"
  $lines += ""
  $lines += "## Environment"
  $lines += ""
  $lines += "| Item | Value |"
  $lines += "|------|-------|"
  $lines += "| USERPROFILE | `$env:USERPROFILE |"
  $lines += "| LOCALAPPDATA | `$env:LOCALAPPDATA |"
  $lines += "| Installed binary | `$InstallBinary |"
  $lines += "| Windows PTY registry | `$WindowsPtyRegistry |"
  $lines += "| Windows PTY logs | `$WindowsPtyLogsDir |"
  $lines += ""
  $lines += "## Final Pass/Fail Table"
  $lines += ""
  $lines += "| Area | Status |"
  $lines += "|------|--------|"
  foreach ($category in @("install", "subcommands", "monitor-1-session", "monitor-2-session", "status-kill-reset", "codex-present")) {
    if ($category -eq "codex-present" -and $Pass -ne "codex-present") {
      continue
    }
    $lines += "| $category | $(Get-CategorySummary -Category $category) |"
  }
  $lines += ""
  $lines += "## Step Results"
  $lines += ""
  $lines += "| Category | Step | Status | Details | Log |"
  $lines += "|----------|------|--------|---------|-----|"
  foreach ($result in $script:Results) {
    $logValue = if ($result.LogPath) { "`$($result.LogPath)" } else { "" }
    $detail = ($result.Details -replace "\r?\n", " ").Trim()
    $lines += "| $($result.Category) | $($result.Step) | $($result.Status) | $detail | $logValue |"
  }
  $lines += ""
  $lines += "## Commands Run"
  $lines += ""
  $lines += "| Label | Command | Exit | Log |"
  $lines += "|-------|---------|------|-----|"
  foreach ($entry in $script:CommandHistory) {
    $lines += "| $($entry.Label) | `$($entry.Command) | $($entry.ExitCode) | `$($entry.LogPath) |"
  }
  $lines += ""
  $lines += "## Screenshot Checklist"
  $lines += ""
  $lines += "- [ ] $ScreenshotsDir\install.png"
  $lines += "- [ ] $ScreenshotsDir\monitor-one-session.png"
  $lines += "- [ ] $ScreenshotsDir\monitor-two-sessions.png"
  $lines += "- [ ] $ScreenshotsDir\terminal-scrollback.png"
  $lines += "- [ ] $ScreenshotsDir\cleanup.png"
  $lines += ""
  $lines += "## Notes"
  $lines += ""
  $lines += "- Expected limitation: `cavekit setup-build` is unsupported on Windows today."
  $lines += "- Expected limitation: `Open` in the Windows monitor stays inside the TUI and uses Terminal-tab scrollback instead of full-screen attach."
  $lines += ""
  Set-Content -Path $ReportPath -Value ($lines -join [Environment]::NewLine)
}

function Ensure-Directory {
  param([string]$Path)
  if (-not (Test-Path $Path)) {
    New-Item -ItemType Directory -Path $Path -Force | Out-Null
  }
}

Ensure-Directory -Path $OutputDir
Ensure-Directory -Path $LogsDir
Ensure-Directory -Path $ScreenshotsDir

try {
  Start-SmokeTranscript

  Write-Section "Environment"
  Add-Result -Category "install" -Step "paths" -Status "PASS" -Details "Output written to $OutputDir"

  Invoke-Step -Category "install" -Step "windows-host" -Body {
    Assert-True ($env:OS -eq "Windows_NT") "This smoke harness must run on Windows."
    return "Running on Windows."
  }

  Invoke-Step -Category "install" -Step "git-present" -Body {
    $git = Get-Command git -ErrorAction Stop
    return "git found at $($git.Source)"
  }

  Invoke-Step -Category "install" -Step "go-present" -Body {
    $go = Get-Command go -ErrorAction Stop
    return "go found at $($go.Source)"
  }

  Invoke-Step -Category "monitor-1-session" -Step "claude-present" -Body {
    $claude = Get-Command claude -ErrorAction Stop
    return "claude found at $($claude.Source)"
  }

  if ($Pass -eq "baseline") {
    if (Get-Command codex -ErrorAction SilentlyContinue) {
      Add-WarningResult -Category "subcommands" -Step "codex-absent-baseline" -Details "codex is installed even though baseline pass expects it to be absent."
    } else {
      Add-Result -Category "subcommands" -Step "codex-absent-baseline" -Status "PASS" -Details "codex is absent, as expected."
    }
  }

  if (-not $SkipBuildAndTest) {
    Write-Section "Go Build and Test"
    Invoke-Step -Category "install" -Step "go-build" -Body {
      $cmd = Invoke-LoggedCommand -Label "go-build" -FilePath "go" -Arguments @("build", "./...")
      return "go build ./... passed. Log: $($cmd.LogPath)"
    }
    Invoke-Step -Category "install" -Step "go-test" -Body {
      $cmd = Invoke-LoggedCommand -Label "go-test" -FilePath "go" -Arguments @("test", "./...")
      return "go test ./... passed. Log: $($cmd.LogPath)"
    }
  } else {
    Add-PendingResult -Category "install" -Step "go-build" -Details "Skipped by -SkipBuildAndTest."
    Add-PendingResult -Category "install" -Step "go-test" -Details "Skipped by -SkipBuildAndTest."
  }

  if (-not $SkipInstall) {
    Write-Section "Native Install"
    Invoke-Step -Category "install" -Step "install-ps1" -Body {
      $cmd = Invoke-LoggedCommand -Label "install-ps1" -FilePath "powershell.exe" -Arguments @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        (Join-Path $RepoRoot "install.ps1")
      )
      return "install.ps1 completed. Log: $($cmd.LogPath)"
    }
  } else {
    Add-PendingResult -Category "install" -Step "install-ps1" -Details "Skipped by -SkipInstall."
  }

  Invoke-Step -Category "install" -Step "binary-exists" -Body {
    Assert-True (Test-Path $InstallBinary) "Missing installed binary at $InstallBinary"
    return "Installed binary found at $InstallBinary"
  }

  Invoke-Step -Category "install" -Step "path-visible" -Body {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    Assert-True (Test-PathEntryContains -PathValue $userPath -Expected $InstallBinDir) "User PATH does not contain $InstallBinDir"
    $command = '$env:Path = [Environment]::GetEnvironmentVariable("Path","User") + ";" + [Environment]::GetEnvironmentVariable("Path","Machine"); cavekit version'
    $cmd = Invoke-LoggedCommand -Label "cavekit-version-new-shell" -FilePath "powershell.exe" -Arguments @("-NoProfile", "-Command", $command)
    Assert-True ($cmd.Output -match "cavekit v") "New shell smoke did not print cavekit version."
    return "User PATH contains the install bin dir and cavekit version works in a simulated new shell."
  }

  Invoke-Step -Category "install" -Step "claude-artifacts" -Body {
    foreach ($path in @($ClaudeSettings, $ClaudeMarketplace, $ClaudePluginManifest, $ClaudePluginCk, $ClaudePluginBp)) {
      Assert-True (Test-Path $path) "Missing Claude artifact: $path"
    }
    Assert-True (Test-ReparsePoint -Path $ClaudePluginCk) "ck plugin link is not a symlink/junction."
    Assert-True (Test-ReparsePoint -Path $ClaudePluginBp) "bp plugin link is not a symlink/junction."
    return "Claude settings, marketplace JSON, plugin manifest, and ck/bp links are present."
  }

  Invoke-Step -Category "install" -Step "codex-artifacts" -Body {
    foreach ($path in @($CodexPluginDir, $CodexMarketplace, $CodexPromptsDir)) {
      Assert-True (Test-Path $path) "Missing Codex artifact: $path"
    }
    $prompts = @(Get-ChildItem -Path $CodexPromptsDir -Filter ck-*.md -File -ErrorAction Stop)
    $legacyPrompts = @(Get-ChildItem -Path $CodexPromptsDir -Filter bp-*.md -File -ErrorAction Stop)
    Assert-True ($prompts.Count -gt 0) "No ck-* prompts found in $CodexPromptsDir"
    Assert-True ($legacyPrompts.Count -gt 0) "No bp-* prompts found in $CodexPromptsDir"
    $marketplace = Read-JsonFile -Path $CodexMarketplace
    $ckEntries = @($marketplace.plugins | Where-Object { $_.name -eq "ck" })
    Assert-True ($ckEntries.Count -eq 1) "Expected exactly one ck marketplace entry, found $($ckEntries.Count)"
    return "Codex plugin dir, marketplace entry, and prompt copies are present."
  }

  Invoke-Step -Category "install" -Step "reinstall-idempotent" -Body {
    $cmd = Invoke-LoggedCommand -Label "install-ps1-rerun" -FilePath "powershell.exe" -Arguments @(
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-File",
      (Join-Path $RepoRoot "install.ps1")
    )
    $marketplace = Read-JsonFile -Path $CodexMarketplace
    $ckEntries = @($marketplace.plugins | Where-Object { $_.name -eq "ck" })
    Assert-True ($ckEntries.Count -eq 1) "Reinstall created duplicate ck marketplace entries."
    return "install.ps1 re-ran cleanly with no duplicate ck marketplace entries. Log: $($cmd.LogPath)"
  }

  Write-Section "New Subcommands"

  Invoke-Step -Category "subcommands" -Step "debug" -Body {
    $cmd = Invoke-LoggedCommand -Label "cavekit-debug" -FilePath $InstallBinary -Arguments @("debug")
    Assert-True ($cmd.Output -match "State file:\s*(.+)") "cavekit debug did not print a state file path."
    $script:StateFilePath = $Matches[1].Trim()
    return "cavekit debug printed state file path $StateFilePath"
  }

  Invoke-Step -Category "subcommands" -Step "reset" -Body {
    $cmd = Invoke-LoggedCommand -Label "cavekit-reset-initial" -FilePath $InstallBinary -Arguments @("reset")
    Assert-True ($cmd.Output -match "State cleared") "cavekit reset did not report state cleared."
    return "cavekit reset cleared state."
  }

  Invoke-Step -Category "subcommands" -Step "status-before-monitor" -Body {
    $cmd = Invoke-LoggedCommand -Label "cavekit-status-before-monitor" -FilePath $InstallBinary -Arguments @("status")
    Assert-True ($cmd.Output -match "No Cavekit worktrees found") "Expected no worktrees before monitor launch."
    return "status reported no Cavekit worktrees before monitor."
  }

  Invoke-Step -Category "subcommands" -Step "kill-before-monitor" -Body {
    $cmd = Invoke-LoggedCommand -Label "cavekit-kill-before-monitor" -FilePath $InstallBinary -Arguments @("kill")
    return "cavekit kill succeeded before monitor. Output: $($cmd.Output.Trim())"
  }

  Invoke-Step -Category "subcommands" -Step "config" -Body {
    $null = Invoke-LoggedCommand -Label "cavekit-config-init" -FilePath $InstallBinary -Arguments @("config", "init")
    $summary = Invoke-LoggedCommand -Label "cavekit-config-summary" -FilePath $InstallBinary -Arguments @("config", "summary")
    Assert-True ($summary.Output -match "Cavekit preset:") "config summary did not print a preset summary."
    $null = Invoke-LoggedCommand -Label "cavekit-config-show-initial" -FilePath $InstallBinary -Arguments @("config", "show")
    $null = Invoke-LoggedCommand -Label "cavekit-config-preset-fast" -FilePath $InstallBinary -Arguments @("config", "preset", "fast")
    $source = Invoke-LoggedCommand -Label "cavekit-config-source-project" -FilePath $InstallBinary -Arguments @("config", "source", "bp_model_preset")
    Assert-True ($source.Output.Trim() -eq "project") "Expected project source after setting preset fast."
    $null = Invoke-LoggedCommand -Label "cavekit-config-preset-balanced-global" -FilePath $InstallBinary -Arguments @("config", "preset", "balanced", "--global")
    $show = Invoke-LoggedCommand -Label "cavekit-config-show-final" -FilePath $InstallBinary -Arguments @("config", "show")
    Assert-True ($show.Output -match "bp_model_preset=fast") "Project preset should continue to override global preset."
    Assert-True ($show.Output -match "project_config=") "config show missing project_config path."
    Assert-True ($show.Output -match "global_config=") "config show missing global_config path."
    return "config init/show/preset/source behaved as expected and project override remained active."
  }

  Invoke-Step -Category "subcommands" -Step "command-gate" -Body {
    $safe = Invoke-LoggedCommand -Label "command-gate-classify-safe" -FilePath $InstallBinary -Arguments @("command-gate", "classify", "git status")
    Assert-True ($safe.Output.Trim() -eq "APPROVE") "Expected APPROVE for git status."
    $blocked = Invoke-LoggedCommand -Label "command-gate-classify-blocked" -FilePath $InstallBinary -Arguments @("command-gate", "classify", "rm -rf /")
    Assert-True ($blocked.Output -match "^BLOCK\|") "Expected BLOCK for rm -rf /."
    $normalized = Invoke-LoggedCommand -Label "command-gate-normalize" -FilePath $InstallBinary -Arguments @("command-gate", "normalize", 'git show 1234567 "./foo/bar.txt"')
    Assert-True ($normalized.Output -match "<HASH>") "Expected normalized command to contain <HASH>."
    Assert-True ($normalized.Output -match "<STR>") "Expected normalized command to contain <STR>."
    $hookCommand = '$payload = ''{"tool_name":"Bash","input":{"command":"rm -rf /"}}''; $payload | & "' + $InstallBinary + '" command-gate hook'
    $hook = Invoke-LoggedCommand -Label "command-gate-hook-blocked" -FilePath "powershell.exe" -Arguments @("-NoProfile", "-Command", $hookCommand)
    Assert-True ($hook.Output -match '"decision"\s*:\s*"block"') "Expected blocked hook JSON payload."
    return "command-gate classify, normalize, and hook blocked-path checks passed."
  }

  Invoke-Step -Category "subcommands" -Step "codex-review-absent" -Body {
    $cmd = Invoke-LoggedCommand -Label "cavekit-codex-review" -FilePath $InstallBinary -Arguments @("codex-review")
    Assert-True ($cmd.Output -match "Codex is not available") "Expected graceful codex-review fallback when codex is absent."
    return "codex-review degraded gracefully without codex."
  }

  Invoke-Step -Category "subcommands" -Step "sync-codex" -Body {
    $cmd = Invoke-LoggedCommand -Label "cavekit-sync-codex" -FilePath $InstallBinary -Arguments @("sync-codex")
    Assert-True ($cmd.Output -match "Codex sync complete") "sync-codex did not complete cleanly."
    $marketplace = Read-JsonFile -Path $CodexMarketplace
    $ckEntries = @($marketplace.plugins | Where-Object { $_.name -eq "ck" })
    Assert-True ($ckEntries.Count -eq 1) "sync-codex created duplicate ck marketplace entries."
    return "sync-codex completed cleanly and kept a single ck marketplace entry."
  }

  Invoke-Step -Category "subcommands" -Step "setup-build-unsupported" -Body {
    $cmd = Invoke-LoggedCommand -Label "cavekit-setup-build" -FilePath $InstallBinary -Arguments @("setup-build") -AllowFailure
    Assert-True ($cmd.ExitCode -ne 0) "setup-build unexpectedly succeeded on Windows."
    Assert-True ($cmd.Output -match "not ported natively on Windows yet") "setup-build did not print the expected unsupported message."
    return "setup-build failed on Windows with the expected unsupported message."
  }

  Write-Section "Monitor Runtime"
  if ($SkipInteractive) {
    Add-PendingResult -Category "monitor-1-session" -Step "manual-monitor-one-session" -Details "Skipped by -SkipInteractive. Use references/windows-smoke.md for the TUI checklist."
    Add-PendingResult -Category "monitor-2-session" -Step "manual-monitor-two-sessions" -Details "Skipped by -SkipInteractive. Use references/windows-smoke.md for the TUI checklist."
    Add-PendingResult -Category "status-kill-reset" -Step "manual-cleanup" -Details "Skipped by -SkipInteractive."
  } else {
    $claudeCommand = Get-Command claude -ErrorAction SilentlyContinue
    if ($null -eq $claudeCommand) {
      Add-Result -Category "monitor-1-session" -Step "launch-monitor" -Status "FAIL" -Details "claude is not on PATH, so the interactive monitor smoke cannot run."
      Add-Result -Category "monitor-2-session" -Step "launch-second-session" -Status "FAIL" -Details "claude is not on PATH, so the interactive monitor smoke cannot run."
      Add-Result -Category "status-kill-reset" -Step "cleanup" -Status "FAIL" -Details "claude is not on PATH, so the interactive monitor smoke cannot run."
    } else {
      $monitorCommand = "& '$InstallBinary' monitor --program claude"
      Start-Process -FilePath "powershell.exe" -WorkingDirectory $RepoRoot -ArgumentList @("-NoExit", "-Command", $monitorCommand) | Out-Null

      Pause-ForManualStep -Message "A new PowerShell window has been opened for the Windows monitor. In that window, press 'n', confirm the picker shows build-site files from context/plans, and launch one site with Enter."
      Invoke-Step -Category "monitor-1-session" -Step "one-session-launch" -Body {
        $entries = Get-SessionRegistryEntries
        $logs = Get-WindowsPtyLogs
        Assert-True ($entries.Count -ge 1) "Expected at least one live windowspty session entry after launching one site."
        Assert-True ($logs.Count -ge 1) "Expected at least one windowspty log file after launching one site."
        return "User confirmed the picker flow; registry has $($entries.Count) live session(s) and $($logs.Count) log file(s)."
      }

      Pause-ForManualStep -Message "In the monitor window, stay on Preview long enough to confirm output refreshes, then press Enter to switch to the Terminal tab, verify full scrollback appears, press 'i', send a harmless input, press Enter, and finally Esc."
      Invoke-Step -Category "monitor-1-session" -Step "terminal-scrollback-and-injection" -Body {
        $entries = Get-SessionRegistryEntries
        Assert-True ($entries.Count -ge 1) "Expected the first monitor session to remain alive after Terminal-tab and input checks."
        return "User confirmed Preview refresh, Terminal-tab scrollback, and input injection while the session remained live."
      }

      Pause-ForManualStep -Message "In the monitor window, launch a second site and verify j/k changes selection across both sessions."
      Invoke-Step -Category "monitor-2-session" -Step "two-session-launch" -Body {
        $entries = Get-SessionRegistryEntries
        $logs = Get-WindowsPtyLogs
        Assert-True ($entries.Count -ge 2) "Expected at least two live windowspty sessions after launching a second site."
        Assert-True ($logs.Count -ge 2) "Expected at least two windowspty log files after launching a second site."
        return "User confirmed two active instances; registry has $($entries.Count) session(s) and $($logs.Count) log file(s)."
      }

      Invoke-Step -Category "status-kill-reset" -Step "status-during-monitor" -Body {
        $cmd = Invoke-LoggedCommand -Label "cavekit-status-during-monitor" -FilePath $InstallBinary -Arguments @("status")
        Assert-True (-not [string]::IsNullOrWhiteSpace($cmd.Output)) "cavekit status returned empty output while monitor was active."
        return "status returned output while monitor was active."
      }
 
      Pause-ForManualStep -Message "Quit the monitor window with 'q', then return here so Cavekit cleanup can be verified."
      Invoke-Step -Category "status-kill-reset" -Step "kill-after-monitor" -Body {
        $cmd = Invoke-LoggedCommand -Label "cavekit-kill-after-monitor" -FilePath $InstallBinary -Arguments @("kill")
        $entries = Get-SessionRegistryEntries
        Assert-True ($entries.Count -eq 0) "Expected no live windowspty sessions after cavekit kill."
        return "kill completed after monitor and removed all live session registry entries."
      }

      Invoke-Step -Category "status-kill-reset" -Step "reset-after-monitor" -Body {
        $cmd = Invoke-LoggedCommand -Label "cavekit-reset-after-monitor" -FilePath $InstallBinary -Arguments @("reset")
        if (-not [string]::IsNullOrWhiteSpace($StateFile)) {
          Assert-True (-not (Test-Path -LiteralPath $StateFile)) "Expected reset to remove the Cavekit state file."
        }
        return "reset completed and removed the Cavekit state file."
      }
    }
  }

  if ($Pass -eq "codex-present") {
    Write-Section "Codex-Present Confidence Pass"
    $codexCommand = Get-Command codex -ErrorAction SilentlyContinue
    if ($null -eq $codexCommand) {
      Add-Result -Category "codex-present" -Step "codex-on-path" -Status "FAIL" -Details "Pass was set to codex-present, but codex is not on PATH."
    } else {
      Invoke-Step -Category "codex-present" -Step "sync-codex-repeat" -Body {
        $cmd = Invoke-LoggedCommand -Label "codex-present-sync-codex" -FilePath $InstallBinary -Arguments @("sync-codex")
        Assert-True ($cmd.Output -match "Codex sync complete") "sync-codex did not complete cleanly with codex installed."
        return "sync-codex succeeded with codex installed."
      }

      $fixtureDir = Join-Path $OutputDir "codex-review-fixture"
      Ensure-Directory -Path $fixtureDir

      Invoke-Step -Category "codex-present" -Step "codex-review-fixture" -Body {
        if (Test-Path -LiteralPath (Join-Path $fixtureDir ".git")) {
          Remove-Item -LiteralPath (Join-Path $fixtureDir ".git") -Recurse -Force
        }

        Set-Content -LiteralPath (Join-Path $fixtureDir "sample.txt") -Value "before`n" -Encoding utf8
        Invoke-LoggedCommand -Label "codex-review-fixture-git-init" -FilePath "git.exe" -Arguments @("init") -WorkingDirectory $fixtureDir | Out-Null
        Invoke-LoggedCommand -Label "codex-review-fixture-git-config-email" -FilePath "git.exe" -Arguments @("config", "user.email", "smoke@example.com") -WorkingDirectory $fixtureDir | Out-Null
        Invoke-LoggedCommand -Label "codex-review-fixture-git-config-name" -FilePath "git.exe" -Arguments @("config", "user.name", "Windows Smoke") -WorkingDirectory $fixtureDir | Out-Null
        Invoke-LoggedCommand -Label "codex-review-fixture-git-add-initial" -FilePath "git.exe" -Arguments @("add", "sample.txt") -WorkingDirectory $fixtureDir | Out-Null
        Invoke-LoggedCommand -Label "codex-review-fixture-git-commit-initial" -FilePath "git.exe" -Arguments @("commit", "-m", "initial") -WorkingDirectory $fixtureDir | Out-Null
        Set-Content -LiteralPath (Join-Path $fixtureDir "sample.txt") -Value "before`nafter`n" -Encoding utf8

        $cmd = Invoke-LoggedCommand -Label "codex-present-codex-review" -FilePath $InstallBinary -Arguments @("codex-review") -WorkingDirectory $fixtureDir -AllowFailure
        Assert-True ($cmd.ExitCode -eq 0) "codex-review returned a non-zero exit code in the disposable git repo."
        $findingsPath = Join-Path $fixtureDir "context\\impl\\impl-review-findings.md"
        $hasRecognizedOutput = ($cmd.Output -match "No findings") -or ($cmd.Output -match "Findings") -or (Test-Path -LiteralPath $findingsPath)
        Assert-True $hasRecognizedOutput "codex-review did not report findings output or create the findings file."
        return "codex-review completed in the disposable git repo and produced a recognized review result."
      }

      Invoke-Step -Category "codex-present" -Step "command-gate-ambiguous-hook" -Body {
        $hookCommand = '$payload = ''{"tool_name":"Bash","input":{"command":"terraform apply -auto-approve"}}''; $payload | & "' + $InstallBinary + '" command-gate hook'
        $started = Get-Date
        $cmd = Invoke-LoggedCommand -Label "codex-present-command-gate-hook" -FilePath "powershell.exe" -Arguments @("-NoProfile", "-Command", $hookCommand)
        $elapsed = (Get-Date) - $started
        Assert-True ($elapsed.TotalSeconds -lt 30) "command-gate hook took too long to return for the ambiguous command path."
        Assert-True (-not [string]::IsNullOrWhiteSpace($cmd.Output)) "command-gate hook returned no output for the ambiguous command path."
        return ("command-gate hook returned within {0:N1}s with a non-empty payload." -f $elapsed.TotalSeconds)
      }
    }
  }
}
finally {
  Stop-SmokeTranscript
  Write-Report
  Write-Section "Artifacts"
  Write-Host "Report: $ReportPath"
  Write-Host "Logs:   $LogsDir"
  if (-not $NoTranscript) {
    Write-Host "Transcript: $TranscriptPath"
  }
}
