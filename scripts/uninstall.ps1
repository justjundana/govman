# govman uninstallation script for Windows
# This script removes govman from $env:USERPROFILE\.govman\bin and removes it from PATH

param(
    [switch]$Help
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

# Colors and styles for Windows Terminal
$Colors = @{
    Red = "`e[0;31m"
    Green = "`e[0;32m"
    Yellow = "`e[1;33m"
    Blue = "`e[0;34m"
    Purple = "`e[0;35m"
    Cyan = "`e[0;36m"
    White = "`e[1;37m"
    Gray = "`e[0;90m"
    Reset = "`e[0m"
    Bold = "`e[1m"
    Dim = "`e[2m"
}

# Unicode characters for better UI
$Icons = @{
    Checkmark = "✓"
    Crossmark = "✗"
    Arrow = "→"
    Trash = "🗑"
    Warning = "⚠"
    Question = "❓"
    Stop = "🛑"
    Clean = "🧹"
    Shield = "🛡"
    Info = "ℹ"
}

# Get terminal width
$TermWidth = try { $Host.UI.RawUI.WindowSize.Width } catch { 80 }

# Print separator line
function Print-Separator {
    param([string]$Char = "-")
    Write-Host ($Char * $TermWidth)
}

# Print fancy header
function Print-Header {
    if (-not [Console]::IsOutputRedirected) { Clear-Host }
    Print-Separator "═"
    Write-Host ""
    Write-Host ""
    Write-Host "    ██╗   ██╗███╗   ██╗██╗███╗   ██╗███████╗████████╗ █████╗ ██╗     ██╗"
    Write-Host "    ██║   ██║████╗  ██║██║████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██║     ██║"
    Write-Host "    ██║   ██║██╔██╗ ██║██║██╔██╗ ██║███████╗   ██║   ███████║██║     ██║"
    Write-Host "    ██║   ██║██║╚██╗██║██║██║╚██╗██║╚════██║   ██║   ██╔══██║██║     ██║"
    Write-Host "    ╚██████╔╝██║ ╚████║██║██║ ╚████║███████║   ██║   ██║  ██║███████╗███████╗"
    Write-Host "     ╚═════╝ ╚═╝  ╚═══╝╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝"
    Write-Host ""
    Write-Host ""
    Write-Host "$($Colors.Bold)$($Colors.White)                        Go Version Manager Uninstaller$($Colors.Reset)"
    Write-Host "$($Colors.Dim)$($Colors.Gray)                  Safe and complete uninstallation process$($Colors.Reset)"
    Write-Host ""
    Print-Separator "═"
    Write-Host ""
}

# Print functions with icons and styling
function Print-Info {
    param([string]$Message)
    Write-Host "$($Colors.Blue)$($Colors.Bold) $($Icons.Info)  INFO$($Colors.Reset) $($Colors.Gray)│$($Colors.Reset) $Message"
}

function Print-Success {
    param([string]$Message)
    Write-Host "$($Colors.Green)$($Colors.Bold) $($Icons.Checkmark)  SUCCESS$($Colors.Reset) $($Colors.Gray)│$($Colors.Reset) $Message"
}

function Print-Warning {
    param([string]$Message)
    Write-Host "$($Colors.Yellow)$($Colors.Bold) $($Icons.Warning)  WARNING$($Colors.Reset) $($Colors.Gray)│$($Colors.Reset) $Message"
}

function Print-Error {
    param([string]$Message)
    Write-Host "$($Colors.Red)$($Colors.Bold) $($Icons.Crossmark)  ERROR$($Colors.Reset) $($Colors.Gray)│$($Colors.Reset) $Message"
}

function Print-Step {
    param([string]$Message)
    Write-Host "$($Colors.Purple)$($Colors.Bold) $($Icons.Arrow)  STEP$($Colors.Reset) $($Colors.Gray)│$($Colors.Reset) $Message"
}

function Print-Clean {
    param([string]$Message)
    Write-Host "$($Colors.Cyan)$($Colors.Bold) $($Icons.Clean)  CLEANING$($Colors.Reset) $($Colors.Gray)│$($Colors.Reset) $Message"
}

function Print-Question {
    param([string]$Message)
    Write-Host "$($Colors.Yellow)$($Colors.Bold) $($Icons.Question)  QUESTION$($Colors.Reset) $($Colors.Gray)│$($Colors.Reset) $Message"
}

# User input function
function Get-UserInput {
    param([string]$Prompt)

    Write-Host -NoNewline $Prompt
    return Read-Host
}

# Show help information
function Show-Help {
    Write-Host "govman uninstaller - Go Version Manager Uninstallation Script for Windows"
    Write-Host ""
    Write-Host "Usage: .\uninstall.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Help           Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\uninstall.ps1         # Run interactive uninstaller"
    Write-Host "  .\uninstall.ps1 -Help   # Show help"
}

function Normalize-WindowsPathEntry {
    param([AllowEmptyString()][string]$Entry)

    if ($null -eq $Entry) { return "" }
    return $Entry.Trim().TrimEnd([char[]]@('\', '/'))
}

function Test-WindowsPathEntry {
    param(
        [AllowEmptyString()][string]$PathValue,
        [string]$Entry
    )

    $expected = Normalize-WindowsPathEntry $Entry
    foreach ($candidate in $PathValue.Split([char[]]@(';'), [StringSplitOptions]::None)) {
        if ([StringComparer]::OrdinalIgnoreCase.Equals((Normalize-WindowsPathEntry $candidate), $expected)) {
            return $true
        }
    }
    return $false
}

function Update-WindowsPathValue {
    param(
        [AllowEmptyString()][string]$PathValue,
        [string]$Entry,
        [ValidateSet("Add", "Remove")][string]$Action
    )

    $matches = Test-WindowsPathEntry $PathValue $Entry
    if ($Action -eq "Add") {
        if ($matches) { return $PathValue }
        if ([string]::IsNullOrEmpty($PathValue)) { return $Entry }
        return "$PathValue;$Entry"
    }
    if (-not $matches) { return $PathValue }

    $expected = Normalize-WindowsPathEntry $Entry
    return (@($PathValue.Split([char[]]@(';'), [StringSplitOptions]::None) | Where-Object {
        -not [StringComparer]::OrdinalIgnoreCase.Equals((Normalize-WindowsPathEntry $_), $expected)
    })) -join ";"
}

function Remove-UserPathEntry {
    param([string]$Entry)

    $key = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey("Environment")
    if ($null -eq $key) { throw "Unable to open HKCU\Environment" }

    $backupName = "Path.govman-backup-$PID-$([Guid]::NewGuid().ToString('N'))"
    try {
        $hadPath = @($key.GetValueNames()) -contains "Path"
        if (-not $hadPath) { return $false }
        $oldPath = [string]$key.GetValue("Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        $valueKind = $key.GetValueKind("Path")
        if (-not (Test-WindowsPathEntry $oldPath $Entry)) { return $false }

        $newPath = Update-WindowsPathValue $oldPath $Entry Remove

        $key.SetValue($backupName, $oldPath, $valueKind)
        try {
            $key.SetValue("Path", $newPath, $valueKind)
            $actualPath = [string]$key.GetValue("Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
            if ($actualPath -cne $newPath -or $key.GetValueKind("Path") -ne $valueKind) {
                throw "PATH verification failed after registry write"
            }
        }
        catch {
            $key.SetValue("Path", $oldPath, $valueKind)
            throw
        }
        finally {
            $key.DeleteValue($backupName, $false)
        }
        return $true
    }
    finally {
        $key.Dispose()
    }
}

function Get-GovmanProfilePaths {
    $documents = [Environment]::GetFolderPath([Environment+SpecialFolder]::MyDocuments)
    return @(
        [string]$PROFILE,
        (Join-Path $documents "WindowsPowerShell\Microsoft.PowerShell_profile.ps1"),
        (Join-Path $documents "PowerShell\Microsoft.PowerShell_profile.ps1")
    ) | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique
}

function Remove-GovmanProfileIntegration {
    param([string[]]$ProfilePaths = @(Get-GovmanProfilePaths))

    $startPattern = '(?m)^# GOVMAN - Go Version Manager\r?$'
    $endPattern = '(?m)^# END GOVMAN\r?$'
    $changed = $false

    foreach ($profilePath in $ProfilePaths) {
        if (-not (Test-Path -LiteralPath $profilePath -PathType Leaf)) { continue }
        $content = [IO.File]::ReadAllText($profilePath)
        $starts = [regex]::Matches($content, $startPattern)
        $ends = [regex]::Matches($content, $endPattern)
        if ($starts.Count -eq 0 -and $ends.Count -eq 0) { continue }
        if ($starts.Count -ne 1 -or $ends.Count -ne 1 -or $ends[0].Index -lt $starts[0].Index) {
            throw "Malformed or duplicate govman marker block in $profilePath"
        }

        $removeStart = $starts[0].Index
        $removeEnd = $ends[0].Index + $ends[0].Length
        if ($removeEnd -lt $content.Length -and $content[$removeEnd] -eq "`n") { $removeEnd++ }
        $updated = $content.Remove($removeStart, $removeEnd - $removeStart)

        $directory = Split-Path -Parent $profilePath
        $leaf = Split-Path -Leaf $profilePath
        $suffix = "$(Get-Date -Format 'yyyyMMddHHmmssfff')-$PID"
        $backupPath = Join-Path $directory "$leaf.govman-backup-$suffix"
        $tempPath = Join-Path $directory ".$leaf.govman-$suffix.tmp"
        $acl = Get-Acl -LiteralPath $profilePath
        Copy-Item -LiteralPath $profilePath -Destination $backupPath -ErrorAction Stop
        try {
            [IO.File]::WriteAllText($tempPath, $updated, [Text.UTF8Encoding]::new($false))
            Set-Acl -LiteralPath $tempPath -AclObject $acl
            [IO.File]::Replace($tempPath, $profilePath, $null)
        }
        catch {
            Remove-Item -LiteralPath $tempPath -Force -ErrorAction SilentlyContinue
            Copy-Item -LiteralPath $backupPath -Destination $profilePath -Force
            throw
        }
        $changed = $true
        Print-Success "Removed shell integration from $profilePath (backup: $backupPath)"
    }
    return $changed
}

function Remove-ProfileIntegration {
    Print-Step "Cleaning PowerShell profile integration..."
    if (-not (Remove-GovmanProfileIntegration)) {
        Print-Info "No govman PowerShell profile integration found"
    }
}

# Check if govman is installed
function Test-GovmanInstallation {
    $installDir = Join-Path $env:USERPROFILE ".govman\bin"
    $govmanDir = Join-Path $env:USERPROFILE ".govman"
    $binaryFound = (Test-Path (Join-Path $installDir "govman-real.exe")) -or
        (Test-Path (Join-Path $installDir "govman.exe")) -or
        (Test-Path (Join-Path $installDir "govman.cmd"))
    $commandFound = $null -ne (Get-Command govman -ErrorAction SilentlyContinue)
    $dataFound = Test-Path $govmanDir

    Print-Step "Checking govman installation..."

    Write-Host ""
    Print-Separator "┄"
    Write-Host "$($Colors.Bold)$($Colors.White)Installation Status:$($Colors.Reset)"
    Print-Separator "┄"

    if ($binaryFound) {
        Write-Host "$($Colors.Green) $($Icons.Checkmark)$($Colors.Reset) Binary directory: $($Colors.Bold)$installDir$($Colors.Reset)"
    } else {
        Write-Host "$($Colors.Gray) $($Icons.Crossmark)$($Colors.Reset) Binary directory: $($Colors.Dim)$installDir (not found)$($Colors.Reset)"
    }

    # Check PATH configuration
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    $pathConfigured = Test-WindowsPathEntry $userPath $installDir

    if ($pathConfigured) {
        Write-Host "$($Colors.Green) $($Icons.Checkmark)$($Colors.Reset) PATH configuration: $($Colors.Bold)Found in user PATH$($Colors.Reset)"
    } else {
        Write-Host "$($Colors.Gray) $($Icons.Crossmark)$($Colors.Reset) PATH configuration: $($Colors.Dim)No govman PATH found$($Colors.Reset)"
    }

    if ($commandFound) {
        try {
            $version = & govman --version 2>$null | Select-Object -First 1
        } catch {
            $version = "unknown"
        }
        Write-Host "$($Colors.Green) $($Icons.Checkmark)$($Colors.Reset) Command available: $($Colors.Bold)govman$($Colors.Reset) $($Colors.Dim)($version)$($Colors.Reset)"
    } else {
        Write-Host "$($Colors.Gray) $($Icons.Crossmark)$($Colors.Reset) Command available: $($Colors.Dim)govman (not found)$($Colors.Reset)"
    }

    if ($dataFound) {
        $dirSize = "{0:N2} MB" -f ((Get-ChildItem $govmanDir -Recurse -ErrorAction SilentlyContinue | Measure-Object -Property Length -Sum).Sum / 1MB)
        Write-Host "$($Colors.Blue) $($Icons.Info)$($Colors.Reset) Data directory: $($Colors.Bold)$govmanDir$($Colors.Reset) $($Colors.Dim)($dirSize)$($Colors.Reset)"
    } else {
        Write-Host "$($Colors.Gray) $($Icons.Crossmark)$($Colors.Reset) Data directory: $($Colors.Dim)$govmanDir (not found)$($Colors.Reset)"
    }

    Print-Separator "┄"
    Write-Host ""

    # Return status: $true if something to uninstall, $false if nothing found
    return ($binaryFound -or $pathConfigured -or $dataFound)
}

# Show what will be removed based on option
function Show-RemovalPreview {
    param([string]$Option)

    Write-Host "$($Colors.Bold)$($Colors.White)Removal Preview:$($Colors.Reset)"
    Print-Separator "┄"

    $installDir = Join-Path $env:USERPROFILE ".govman\bin"
    $govmanDir = Join-Path $env:USERPROFILE ".govman"

    # Check binary
    if ((Test-Path (Join-Path $installDir "govman-real.exe")) -or
        (Test-Path (Join-Path $installDir "govman.exe")) -or
        (Test-Path (Join-Path $installDir "govman.cmd"))) {
        Write-Host "$($Colors.Red) $($Icons.Trash)$($Colors.Reset) Binary directory: $($Colors.Bold)$installDir$($Colors.Reset)"
    } else {
        Write-Host "$($Colors.Gray) $($Icons.Crossmark)$($Colors.Reset) Binary directory: $($Colors.Dim)$installDir (not found)$($Colors.Reset)"
    }

    # Check PATH configuration
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if (Test-WindowsPathEntry $userPath $installDir) {
        Write-Host "$($Colors.Red) $($Icons.Trash)$($Colors.Reset) PATH configuration: $($Colors.Bold)User PATH entry$($Colors.Reset)"
    } else {
        Write-Host "$($Colors.Gray) $($Icons.Crossmark)$($Colors.Reset) PATH configuration: $($Colors.Dim)No govman PATH found$($Colors.Reset)"
    }

    # Show data directory based on option
    if (Test-Path $govmanDir) {
        $dirSize = "{0:N2} MB" -f ((Get-ChildItem $govmanDir -Recurse -ErrorAction SilentlyContinue | Measure-Object -Property Length -Sum).Sum / 1MB)
        if ($Option -eq "complete") {
            Write-Host "$($Colors.Red) $($Icons.Trash)$($Colors.Reset) Data directory: $($Colors.Bold)$govmanDir$($Colors.Reset) $($Colors.Dim)($dirSize)$($Colors.Reset)"
        } else {
            Write-Host "$($Colors.Green) $($Icons.Shield)$($Colors.Reset) Data directory: $($Colors.Bold)$govmanDir$($Colors.Reset) $($Colors.Dim)($dirSize - will be kept)$($Colors.Reset)"
        }
    } else {
        Write-Host "$($Colors.Gray) $($Icons.Crossmark)$($Colors.Reset) Data directory: $($Colors.Dim)$govmanDir (not found)$($Colors.Reset)"
    }

    Print-Separator "┄"
    Write-Host ""
}

# Remove binary with feedback
function Remove-Binary {
    $installDir = Join-Path $env:USERPROFILE ".govman\bin"

    Print-Step "Removing govman binary..."

    if (Test-Path $installDir) {
        try {
            Remove-Item -Path $installDir -Recurse -Force
            Print-Success "Removed govman binary from $installDir"
        }
        catch {
            Print-Error "Failed to remove binary directory: $($_.Exception.Message)"
            throw
        }
    } else {
        Print-Warning "govman binary directory not found at $installDir"
    }
}

# Remove from PATH with feedback
function Remove-FromPath {
    $installDir = Join-Path $env:USERPROFILE ".govman\bin"

    Print-Step "Cleaning PATH configuration..."

    if (Remove-UserPathEntry $installDir) {
        Print-Success "Cleaned PATH configuration"
    } else {
        Print-Info "No govman PATH configuration found"
    }

    if (Test-WindowsPathEntry $env:PATH $installDir) {
        $expected = Normalize-WindowsPathEntry $installDir
        $env:PATH = (@($env:PATH.Split([char[]]@(';'), [StringSplitOptions]::None) | Where-Object {
            -not [StringComparer]::OrdinalIgnoreCase.Equals((Normalize-WindowsPathEntry $_), $expected)
        })) -join ";"
    }
}

# Remove entire govman directory with feedback
function Remove-GovmanDir {
    $govmanDir = Join-Path $env:USERPROFILE ".govman"

    Print-Step "Removing govman data directory..."

    if (Test-Path $govmanDir) {
        # Show what's being removed
        $dirSize = "{0:N2} MB" -f ((Get-ChildItem $govmanDir -Recurse -ErrorAction SilentlyContinue | Measure-Object -Property Length -Sum).Sum / 1MB)
        Print-Info "Removing directory: $govmanDir ($dirSize)"

        try {
            Remove-Item -Path $govmanDir -Recurse -Force
            Print-Success "Removed govman data directory"
        }
        catch {
            Print-Error "Failed to remove data directory: $($_.Exception.Message)"
            throw
        }
    } else {
        Print-Warning "govman directory not found at $govmanDir"
    }
}

# Show uninstall options
function Show-UninstallOptions {
    Print-Separator "═"
    Write-Host "$($Colors.Bold)$($Colors.White) $($Icons.Question)  UNINSTALLATION OPTIONS$($Colors.Reset)"
    Print-Separator "═"
    Write-Host ""
    Write-Host "$($Colors.Cyan)$($Colors.Bold)1)$($Colors.Reset) $($Colors.White)Minimal Removal$($Colors.Reset) $($Colors.Dim)(Recommended)$($Colors.Reset)"
    Write-Host "   • Remove govman binary and executable"
    Write-Host "   • Clean PATH configuration"
    Write-Host "   • $($Colors.Green)Keep$($Colors.Reset) downloaded Go versions for future use"
    Write-Host ""
    Write-Host "$($Colors.Red)$($Colors.Bold)2)$($Colors.Reset) $($Colors.White)Complete Removal$($Colors.Reset) $($Colors.Dim)(Permanent)$($Colors.Reset)"
    Write-Host "   • Remove govman binary and executable"
    Write-Host "   • Clean PATH configuration"
    Write-Host "   • $($Colors.Red)Delete$($Colors.Reset) all downloaded Go versions and data"
    Write-Host "   • $($Colors.Red)Delete$($Colors.Reset) entire .govman directory"
    Write-Host ""
    Write-Host "$($Colors.Gray)$($Colors.Bold)3)$($Colors.Reset) $($Colors.White)Cancel$($Colors.Reset)"
    Write-Host "   • Exit without making any changes"
    Write-Host ""
    Print-Separator "┄"
}

# Show completion message
function Show-Completion {
    param([bool]$CompleteRemoval)

    Write-Host ""
    Print-Separator "═"
    Write-Host ""
    if ($CompleteRemoval) {
        Write-Host "$($Colors.Green)$($Colors.Bold) $($Icons.Checkmark)  COMPLETE UNINSTALLATION SUCCESSFUL!$($Colors.Reset)"
        Write-Host ""
        Print-Separator "┄"
        Write-Host "$($Colors.Bold)$($Colors.White)What was removed:$($Colors.Reset)"
        Write-Host " • govman binary and executable"
        Write-Host " • PATH configuration"
        Write-Host " • PowerShell profile integration (when present)"
        Write-Host " • All downloaded Go versions"
        Write-Host " • Complete .govman directory"
    } else {
        Write-Host "$($Colors.Green)$($Colors.Bold) $($Icons.Checkmark)  MINIMAL UNINSTALLATION COMPLETE!$($Colors.Reset)"
        Write-Host ""
        Print-Separator "┄"
        Write-Host "$($Colors.Bold)$($Colors.White)What was removed:$($Colors.Reset)"
        Write-Host " • govman binary and executable"
        Write-Host " • PATH configuration"
        Write-Host " • PowerShell profile integration (when present)"
        Write-Host ""
        Write-Host "$($Colors.Bold)$($Colors.White)What was kept:$($Colors.Reset)"
        Write-Host " • Downloaded Go versions in .govman directory"
    }
    Print-Separator "┄"
    Write-Host "$($Colors.Bold)$($Colors.White)Final Steps:$($Colors.Reset)"
    Write-Host " 1. Restart your PowerShell/Command Prompt to complete the process"
    Write-Host " 2. Verify with 'govman --version' (should show 'not recognized')"
    if (-not $CompleteRemoval) {
        Write-Host " 3. Manually remove '.govman' directory if you change your mind later"
    }
    Print-Separator "┄"
    Write-Host "Thank you for using govman!"
    Print-Separator "═"
    Write-Host ""
}

# Main uninstallation function
function Main {
    # Handle help parameter
    if ($Help) {
        Show-Help
        exit 0
    }

    # Show header
    Print-Header

    Print-Info "Starting govman uninstallation process..."
    Write-Host ""

    # Check if govman is installed
    if (-not (Test-GovmanInstallation)) {
        Print-Warning "govman does not appear to be installed on this system"
        Write-Host ""
        Print-Separator "┄"
        Write-Host "$($Colors.Bold)$($Colors.White)No govman installation found!$($Colors.Reset)"
        Print-Separator "┄"
        Write-Host "It looks like govman is not installed or has already been removed."
        Write-Host "Common reasons:"
        Write-Host " • govman was never installed"
        Write-Host " • govman was already uninstalled"
        Write-Host " • govman was installed in a different location"
        Write-Host " • Installation was incomplete or corrupted"
        Print-Separator "┄"
        Write-Host ""

        $response = Get-UserInput "Do you want to clean any remaining traces? $($Colors.Dim)(y/N):$($Colors.Reset) "

        if ($response -notmatch "^[Yy]$") {
            Write-Host ""
            Print-Info "Exiting without making changes"
            Print-Separator "═"
            Write-Host "$($Colors.Dim)$($Colors.Gray)No changes were made to your system.$($Colors.Reset)"
            Print-Separator "═"
            Write-Host ""
            exit 0
        }

        Write-Host ""
        Print-Info "Proceeding with cleanup of any remaining traces..."
        Write-Host ""
    } else {
        Print-Success "govman installation detected"
        Write-Host ""
    }

    # Show uninstall options
    Show-UninstallOptions

    # Get user choice
    $response = Get-UserInput "Choose an option $($Colors.Dim)(1/2/3):$($Colors.Reset) "

    Write-Host ""

    switch ($response) {
        "1" {
            Print-Info "Proceeding with minimal removal..."
            Write-Host ""
            Show-RemovalPreview "minimal"

            # Final confirmation for minimal removal
            Print-Separator "┄"
            Write-Host "$($Colors.Yellow)$($Colors.Bold) $($Icons.Stop)  FINAL CONFIRMATION$($Colors.Reset)"
            Print-Separator "┄"
            $confirm = Get-UserInput "Proceed with minimal removal? $($Colors.Dim)(y/N):$($Colors.Reset) "

            if ($confirm -match "^[Yy]$") {
                Write-Host ""
                Remove-ProfileIntegration
                Write-Host ""
                Remove-FromPath
                Write-Host ""
                Remove-Binary
                Write-Host ""
                Show-Completion $false
            } else {
                Write-Host ""
                Print-Info "Uninstallation cancelled by user"
                Print-Separator "═"
                Write-Host "$($Colors.Dim)$($Colors.Gray)No changes were made to your system.$($Colors.Reset)"
                Print-Separator "═"
                Write-Host ""
            }
        }

        "2" {
            Print-Info "Proceeding with complete removal..."
            Write-Host ""
            Show-RemovalPreview "complete"

            # Final confirmation for complete removal
            Print-Separator "┄"
            Write-Host "$($Colors.Red)$($Colors.Bold) $($Icons.Stop)  DANGER: COMPLETE REMOVAL$($Colors.Reset)"
            Print-Separator "┄"
            Write-Host "$($Colors.Red)This will permanently delete ALL govman data and cannot be undone!$($Colors.Reset)"
            Print-Separator "┄"
            $confirm = Get-UserInput "Type 'DELETE' to confirm complete removal: "

            if ($confirm -eq "DELETE") {
                Write-Host ""
                Remove-ProfileIntegration
                Write-Host ""
                Remove-FromPath
                Write-Host ""
                Remove-GovmanDir
                Write-Host ""
                Show-Completion $true
            } else {
                Write-Host ""
                Print-Info "Complete removal cancelled - confirmation text did not match"
                Print-Separator "═"
                Write-Host "$($Colors.Dim)$($Colors.Gray)No changes were made to your system.$($Colors.Reset)"
                Print-Separator "═"
                Write-Host ""
            }
        }

        default {
            Write-Host ""
            Print-Info "Uninstallation cancelled by user"
            Print-Separator "═"
            Write-Host "$($Colors.Dim)$($Colors.Gray)No changes were made to your system.$($Colors.Reset)"
            Print-Separator "═"
            Write-Host ""
        }
    }
}

# Trap for clean exit
trap {
    Write-Host ""
    Print-Error "Uninstallation interrupted. Incomplete removal may have occurred."
    exit 1
}

# Run only when executed, allowing the path helpers to be dot-sourced by tests.
if ($MyInvocation.InvocationName -ne ".") {
    Main
}
