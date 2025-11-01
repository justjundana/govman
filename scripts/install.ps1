# govman installation script for Windows
# This script installs govman to $env:USERPROFILE\.govman\bin and adds it to PATH

param(
    [switch]$Quiet,
    [string]$Version,
    [switch]$Help
)

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
    Download = "⬇"
    Warning = "⚠"
    Install = "📦"
    Info = "ℹ"
    Rocket = "🚀"
    Gear = "⚙"
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
    if ($Quiet) { return }
    Clear-Host
    Print-Separator "═"
    Write-Host ""
    Write-Host ""
    Write-Host "    ██╗███╗   ██╗███████╗████████╗ █████╗ ██╗     ██╗     ███████╗██████╗"
    Write-Host "    ██║████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██║     ██║     ██╔════╝██╔══██╗"
    Write-Host "    ██║██╔██╗ ██║███████╗   ██║   ███████║██║     ██║     █████╗  ██████╔╝"
    Write-Host "    ██║██║╚██╗██║╚════██║   ██║   ██╔══██║██║     ██║     ██╔══╝  ██╔══██╗"
    Write-Host "    ██║██║ ╚████║███████║   ██║   ██║  ██║███████╗███████╗███████╗██║  ██║"
    Write-Host "    ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝"
    Write-Host ""
    Write-Host ""
    Write-Host "$($Colors.Bold)$($Colors.White)                        Go Version Manager Installer$($Colors.Reset)"
    Write-Host "$($Colors.Dim)$($Colors.Gray)                    Fast and secure installation process$($Colors.Reset)"
    Write-Host ""
    Print-Separator "═"
    Write-Host ""
}

# Print functions with icons and styling
function Print-Info {
    param([string]$Message)
    if ($Quiet) { return }
    Write-Host "$($Colors.Blue)$($Colors.Bold) $($Icons.Info)  INFO$($Colors.Reset) $($Colors.Gray)│$($Colors.Reset) $Message"
}

function Print-Success {
    param([string]$Message)
    if ($Quiet) { return }
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
    if ($Quiet) { return }
    Write-Host "$($Colors.Purple)$($Colors.Bold) $($Icons.Arrow)  STEP$($Colors.Reset) $($Colors.Gray)│$($Colors.Reset) $Message"
}

function Print-Install {
    param([string]$Message)
    if ($Quiet) { return }
    Write-Host "$($Colors.Cyan)$($Colors.Bold) $($Icons.Install)  INSTALLING$($Colors.Reset) $($Colors.Gray)│$($Colors.Reset) $Message"
}

# Show help information
function Show-Help {
    Write-Host "govman installer - Go Version Manager Installation Script for Windows"
    Write-Host ""
    Write-Host "Usage: .\install.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Quiet          Run in quiet mode (minimal output)"
    Write-Host "  -Version VER    Install specific version (e.g., v1.0.0)"
    Write-Host "  -Help           Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\install.ps1                   # Install latest version"
    Write-Host "  .\install.ps1 -Quiet            # Install quietly"
    Write-Host "  .\install.ps1 -Version v1.0.0   # Install specific version"
}

# Detect platform (Windows architecture)
function Get-Platform {
    $arch = if ($env:PROCESSOR_ARCHITECTURE -eq "AMD64" -or $env:PROCESSOR_ARCHITEW6432 -eq "AMD64") {
        "amd64"
    } elseif ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
        "arm64"
    } else {
        "amd64"  # Default to amd64 for Windows
    }
    return "windows/$arch"
}

# Get the latest release version from GitHub
function Get-LatestVersion {
    if ($Version) {
        return $Version
    }

    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/justjundana/govman/releases/latest" -TimeoutSec 30
        return $response.tag_name
    }
    catch {
        Print-Error "Failed to get latest version information"
        Print-Info "Error: $($_.Exception.Message)"
        exit 1
    }
}

# Verify binary (basic validation)
function Test-Binary {
    param([string]$BinaryPath)

    if (-not (Test-Path $BinaryPath)) {
        Print-Error "Binary file not found: $BinaryPath"
        return $false
    }

    # Check file size (should be > 1MB for a Go binary)
    $fileSize = (Get-Item $BinaryPath).Length
    if ($fileSize -lt 1048576) {
        Print-Warning "Binary file seems unusually small ($fileSize bytes)"
    }

    # Try to get version to ensure it's a valid govman binary
    try {
        $null = & $BinaryPath --version 2>$null
        Print-Success "Binary validation completed"
        return $true
    }
    catch {
        Print-Error "Downloaded binary appears to be corrupted or invalid"
        return $false
    }
}

# Animated loading for download process
function Show-DownloadProgress {
    param([string]$Item)
    if ($Quiet) { return }

    $spinChars = @('⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏')
    Write-Host -NoNewline "   $($Colors.Dim)Downloading $Item... $($Colors.Reset)"

    for ($i = 0; $i -lt 15; $i++) {
        $spinChar = $spinChars[$i % $spinChars.Length]
        Write-Host -NoNewline "`r   $($Colors.Dim)Downloading $Item... $($Colors.Cyan)$spinChar$($Colors.Reset) "
        Start-Sleep -Milliseconds 100
    }
    Write-Host "`r   $($Colors.Green)$($Icons.Checkmark)$($Colors.Reset) Downloaded $Item successfully.      "
}

# Animated loading for installation process
function Show-InstallProgress {
    param([string]$Item)
    if ($Quiet) { return }

    $spinChars = @('⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏')
    Write-Host -NoNewline "   $($Colors.Dim)Installing $Item... $($Colors.Reset)"

    for ($i = 0; $i -lt 10; $i++) {
        $spinChar = $spinChars[$i % $spinChars.Length]
        Write-Host -NoNewline "`r   $($Colors.Dim)Installing $Item... $($Colors.Purple)$spinChar$($Colors.Reset) "
        Start-Sleep -Milliseconds 100
    }
    Write-Host "`r   $($Colors.Green)$($Icons.Checkmark)$($Colors.Reset) Installed $Item successfully.      "
}

# Download the binary
function Download-Binary {
    param(
        [string]$Version,
        [string]$Platform,
        [string]$InstallDir
    )

    $parts = $Platform -split "/"
    $os = $parts[0]
    $arch = $parts[1]

    # Construct download URL
    $downloadUrl = "https://github.com/justjundana/govman/releases/download/$Version/govman-$os-$arch.exe"
    $binaryPath = Join-Path $InstallDir "govman.exe"

    Print-Step "Downloading govman $Version for $Platform..."
    Print-Info "Download URL: $downloadUrl"

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    # Show download progress animation
    if (-not $Quiet) {
        Show-DownloadProgress "govman binary"
    }

    # Download binary
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $binaryPath -TimeoutSec 60
    }
    catch {
        Print-Error "Failed to download govman binary"
        Print-Info "Error: $($_.Exception.Message)"
        exit 1
    }

    # Check if download was successful
    if (-not (Test-Path $binaryPath)) {
        Print-Error "Failed to download govman binary"
        exit 1
    }

    # Validate the downloaded binary
    if (-not (Test-Binary $binaryPath)) {
        Print-Error "Binary validation failed"
        Remove-Item $binaryPath -Force -ErrorAction SilentlyContinue
        exit 1
    }

    Print-Success "Downloaded govman binary to $binaryPath"
    return $binaryPath
}

# Add to PATH and initialize environment
function Add-ToPath {
    param([string]$InstallDir)

    $govmanBinary = Join-Path $InstallDir "govman.exe"

    if (-not (Test-Path $govmanBinary)) {
        Print-Error "govman binary not found at $govmanBinary"
        exit 1
    }

    Print-Step "Configuring Windows environment..."

    # Show install progress animation
    if (-not $Quiet) {
        Show-InstallProgress "environment configuration"
    }

    # Get current user PATH
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")

    # Check if install directory is already in PATH
    if ($userPath -notlike "*$InstallDir*") {
        # Add to user PATH
        $newPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Print-Success "Added $InstallDir to user PATH"
    } else {
        Print-Info "Install directory already in PATH"
    }

    # Run govman init for additional setup
    try {
        $initOutput = & $govmanBinary init --force 2>&1
        if ($LASTEXITCODE -eq 0) {
            Print-Success "Shell configuration completed successfully"
            if ($initOutput -and -not $Quiet) {
                Write-Host $initOutput
            }
        } else {
            Print-Warning "Shell configuration had issues. You may need to run 'govman init' manually."
            if ($initOutput) {
                Write-Host $initOutput
            }
        }
    }
    catch {
        Print-Warning "Could not run 'govman init'. Please run it manually after installation."
    }
}

# Show system information
function Show-SystemInfo {
    param(
        [string]$Platform,
        [string]$Version,
        [string]$InstallDir
    )

    if ($Quiet) { return }

    Print-Separator "┄"
    Write-Host "$($Colors.Bold)$($Colors.White)System Information:$($Colors.Reset)"
    Print-Separator "┄"

    $parts = $Platform -split "/"
    $os = $parts[0]
    $arch = $parts[1]

    Write-Host "$($Colors.Green) $($Icons.Checkmark)$($Colors.Reset) Operating System: $($Colors.Bold)Windows$($Colors.Reset)"
    Write-Host "$($Colors.Green) $($Icons.Checkmark)$($Colors.Reset) Architecture: $($Colors.Bold)$arch$($Colors.Reset)"
    Write-Host "$($Colors.Green) $($Icons.Checkmark)$($Colors.Reset) Version: $($Colors.Bold)$Version$($Colors.Reset)"
    Write-Host "$($Colors.Blue) $($Icons.Info)$($Colors.Reset) Install Directory: $($Colors.Bold)$InstallDir$($Colors.Reset)"

    Print-Separator "┄"
    Write-Host ""
}

# Show completion message
function Show-Completion {
    param([string]$Version)

    Write-Host ""
    Print-Separator "═"
    Write-Host ""
    Write-Host "$($Colors.Green)$($Colors.Bold) $($Icons.Rocket)  INSTALLATION SUCCESSFUL!$($Colors.Reset)"
    Write-Host ""
    Print-Separator "┄"
    Write-Host "$($Colors.Bold)$($Colors.White)What was installed:$($Colors.Reset)"
    Write-Host " • govman binary and executable"
    Write-Host " • Windows PATH configuration"
    Write-Host " • Environment setup complete"
    Print-Separator "┄"
    Write-Host "$($Colors.Bold)$($Colors.White)Next Steps:$($Colors.Reset)"
    Write-Host " 1. Restart your PowerShell/Command Prompt"
    Write-Host " 2. Verify with 'govman --version'"
    Write-Host " 3. Get started with 'govman --help'"
    Print-Separator "┄"
    Write-Host "$($Colors.Bold)$($Colors.White)Quick Commands:$($Colors.Reset)"
    Write-Host " • govman list         - List available Go versions"
    Write-Host " • govman install 1.25 - Install Go 1.25"
    Write-Host " • govman use 1.25     - Switch to Go 1.25"
    Print-Separator "┄"
    Write-Host "Welcome to govman! 🎉"
    Print-Separator "═"
    Write-Host ""
}

# Check if govman is already installed
function Test-ExistingInstallation {
    $installDir = Join-Path $env:USERPROFILE ".govman\bin"
    $govmanDir = Join-Path $env:USERPROFILE ".govman"
    $binaryFound = Test-Path (Join-Path $installDir "govman.exe")
    $commandFound = $null -ne (Get-Command govman -ErrorAction SilentlyContinue)

    Print-Step "Checking for existing installation..."

    if ($binaryFound -or $commandFound) {
        Write-Host ""
        Print-Separator "┄"
        Write-Host "$($Colors.Bold)$($Colors.White)Existing Installation Detected:$($Colors.Reset)"
        Print-Separator "┄"

        if ($binaryFound) {
            Write-Host "$($Colors.Green) $($Icons.Checkmark)$($Colors.Reset) Binary found: $($Colors.Bold)$(Join-Path $installDir 'govman.exe')$($Colors.Reset)"
        }

        if ($commandFound) {
            try {
                $version = & govman --version 2>$null | Select-Object -First 1
            } catch {
                $version = "unknown"
            }
            Write-Host "$($Colors.Green) $($Icons.Checkmark)$($Colors.Reset) Command available: $($Colors.Bold)govman$($Colors.Reset) $($Colors.Dim)($version)$($Colors.Reset)"
        }

        if (Test-Path $govmanDir) {
            $dirSize = "{0:N2} MB" -f ((Get-ChildItem $govmanDir -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB)
            Write-Host "$($Colors.Blue) $($Icons.Info)$($Colors.Reset) Data directory: $($Colors.Bold)$govmanDir$($Colors.Reset) $($Colors.Dim)($dirSize)$($Colors.Reset)"
        }

        Print-Separator "┄"
        Write-Host ""
        Print-Warning "govman is already installed on this system!"
        Write-Host ""
        Print-Separator "┄"
        Write-Host "$($Colors.Bold)$($Colors.White)What you can do:$($Colors.Reset)"
        Write-Host " • Run 'govman --version' to check current version"
        Write-Host " • Run 'govman --help' to see available commands"
        Write-Host " • Use the uninstaller script first if you need to reinstall"
        Write-Host " • Check 'govman list' to see available Go versions"
        Print-Separator "┄"
        Write-Host ""
        Print-Separator "═"
        Write-Host "$($Colors.Dim)$($Colors.Gray)Installation cancelled - govman already exists$($Colors.Reset)"
        Print-Separator "═"
        Write-Host ""
        exit 0
    } else {
        Print-Success "No existing installation found - proceeding with fresh install"
        Write-Host ""
    }
}

# Main installation function
function Main {
    # Handle help parameter
    if ($Help) {
        Show-Help
        exit 0
    }

    # Show header
    Print-Header

    Print-Info "Starting govman installation process..."
    Write-Host ""

    # Check for existing installation first
    Test-ExistingInstallation

    # Detect platform
    Print-Step "Detecting system platform..."
    $platform = Get-Platform
    Print-Success "Detected platform: $($Colors.Bold)$platform$($Colors.Reset)"
    Write-Host ""

    # Get latest version
    Print-Step "Fetching latest version information..."
    $version = Get-LatestVersion
    Print-Success "Latest version: $($Colors.Bold)$version$($Colors.Reset)"
    Write-Host ""

    # Set installation directory
    $installDir = Join-Path $env:USERPROFILE ".govman\bin"
    Print-Info "Installation directory: $($Colors.Bold)$installDir$($Colors.Reset)"
    Write-Host ""

    # Show system info
    Show-SystemInfo $platform $version $installDir

    # Download binary
    $binaryPath = Download-Binary $version $platform $installDir
    Write-Host ""

    # Add to PATH
    Add-ToPath $installDir
    Write-Host ""

    # Verify installation
    Print-Step "Verifying installation..."
    try {
        $null = & $binaryPath --version 2>$null
        $installedVersion = & $binaryPath --version 2>$null | Select-Object -First 1
        Print-Success "Installation verified: $($Colors.Bold)$installedVersion$($Colors.Reset)"
        Show-Completion $version
    }
    catch {
        Print-Warning "Installation completed, but verification failed"
        Write-Host ""
        Print-Separator "┄"
        Write-Host "$($Colors.Bold)$($Colors.White)Manual Steps Required:$($Colors.Reset)"
        Write-Host " 1. Restart your PowerShell/Command Prompt"
        Write-Host " 2. Try running 'govman --version'"
        Write-Host " 3. If issues persist, run 'govman init' manually"
        Print-Separator "┄"
        Write-Host ""
    }
}

# Trap for clean exit
trap {
    Write-Host ""
    Print-Error "Installation interrupted. Partial installation may have occurred."
    exit 1
}

# Run main function
Main