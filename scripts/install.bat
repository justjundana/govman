@echo off
setlocal enabledelayedexpansion

REM govman installation script for Windows Command Prompt
REM This script installs govman to %USERPROFILE%\.govman\bin and adds it to PATH

REM Parse command line arguments
set QUIET_MODE=0
set SPECIFIC_VERSION=
set SHOW_HELP=0

:parse_args
if "%~1"=="" goto :args_done
if /i "%~1"=="--quiet" set QUIET_MODE=1 & shift & goto :parse_args
if /i "%~1"=="-q" set QUIET_MODE=1 & shift & goto :parse_args
if /i "%~1"=="--version" if "%~2"=="" echo Missing value for --version & exit /b 1
if /i "%~1"=="--version" set SPECIFIC_VERSION=%~2 & shift & shift & goto :parse_args
if /i "%~1"=="-v" if "%~2"=="" echo Missing value for -v & exit /b 1
if /i "%~1"=="-v" set SPECIFIC_VERSION=%~2 & shift & shift & goto :parse_args
if /i "%~1"=="--help" set SHOW_HELP=1 & shift & goto :parse_args
if /i "%~1"=="-h" set SHOW_HELP=1 & shift & goto :parse_args
echo Unknown option: %~1
call :show_help
exit /b 1

:args_done

REM Show help if requested
if %SHOW_HELP%==1 (
    call :show_help
    exit /b 0
)

REM ANSI color codes (for Windows 10+ terminals)
REM Keep output portable. The previous values omitted the ESC character and
REM printed literal control-code fragments instead of colors.
set "RED="
set "GREEN="
set "YELLOW="
set "BLUE="
set "PURPLE="
set "CYAN="
set "WHITE="
set "GRAY="
set "RESET="
set "BOLD="
set "DIM="

REM Unicode characters (will fallback to ASCII on older systems)
set "CHECKMARK=v"
set "CROSSMARK=x"
set "ARROW=->"
set "INFO=i"
set "WARNING=!"
set "INSTALL=+"

REM Main execution
call :print_header
call :print_info "Starting govman installation process..."
echo.

call :check_existing_installation
if !errorlevel! neq 0 exit /b !errorlevel!
if "!ALREADY_INSTALLED!"=="1" exit /b 0

call :detect_platform
if !errorlevel! neq 0 exit /b !errorlevel!

call :get_latest_version
if !errorlevel! neq 0 exit /b !errorlevel!

set "INSTALL_DIR=%USERPROFILE%\.govman\bin"
call :print_info "Installation directory: %INSTALL_DIR%"
echo.

call :show_system_info
call :download_binary
if !errorlevel! neq 0 exit /b !errorlevel!

call :add_to_path
if !errorlevel! neq 0 exit /b !errorlevel!

call :verify_installation
if !errorlevel! neq 0 exit /b !errorlevel!
call :show_completion

goto :eof

REM Functions start here

:show_help
echo govman installer - Go Version Manager Installation Script for Windows
echo.
echo Usage: %~nx0 [OPTIONS]
echo.
echo Options:
echo   --quiet, -q         Run in quiet mode (minimal output)
echo   --version, -v VER   Install specific version (e.g., v1.0.0)
echo   --help, -h          Show this help message
echo.
echo Examples:
echo   %~nx0                  # Install latest version
echo   %~nx0 --quiet          # Install quietly
echo   %~nx0 --version v1.0.0 # Install specific version
goto :eof

:print_header
if %QUIET_MODE%==1 goto :eof
cls
call :print_separator "="
echo.
echo.
echo     ██╗███╗   ██╗███████╗████████╗ █████╗ ██╗     ██╗     ███████╗██████╗
echo     ██║████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██║     ██║     ██╔════╝██╔══██╗
echo     ██║██╔██╗ ██║███████╗   ██║   ███████║██║     ██║     █████╗  ██████╔╝
echo     ██║██║╚██╗██║╚════██║   ██║   ██╔══██║██║     ██║     ██╔══╝  ██╔══██╗
echo     ██║██║ ╚████║███████║   ██║   ██║  ██║███████╗███████╗███████╗██║  ██║
echo     ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝
echo.
echo.
echo %BOLD%%WHITE%                        Go Version Manager Installer%RESET%
echo %DIM%%GRAY%                    Fast and secure installation process%RESET%
echo.
call :print_separator "="
echo.
goto :eof

:print_separator
set "char=%~1"
if "%char%"=="" set "char=-"
for /l %%i in (1,1,79) do echo|set /p="!char!"
echo.
goto :eof

:print_info
if %QUIET_MODE%==1 goto :eof
echo %BLUE%%BOLD% %INFO%  INFO%RESET% %GRAY%^|%RESET% %~1
goto :eof

:print_success
if %QUIET_MODE%==1 goto :eof
echo %GREEN%%BOLD% %CHECKMARK%  SUCCESS%RESET% %GRAY%^|%RESET% %~1
goto :eof

:print_warning
echo %YELLOW%%BOLD% %WARNING%  WARNING%RESET% %GRAY%^|%RESET% %~1
goto :eof

:print_error
echo %RED%%BOLD% %CROSSMARK%  ERROR%RESET% %GRAY%^|%RESET% %~1
goto :eof

:print_step
if %QUIET_MODE%==1 goto :eof
echo %PURPLE%%BOLD% %ARROW%  STEP%RESET% %GRAY%^|%RESET% %~1
goto :eof

:check_existing_installation
call :print_step "Checking for existing installation..."

set "ALREADY_INSTALLED=0"
set "BINARY_FOUND=0"
set "COMMAND_FOUND=0"
set "DATA_FOUND=0"

if exist "%USERPROFILE%\.govman\bin\govman-real.exe" set "BINARY_FOUND=1"
if exist "%USERPROFILE%\.govman\bin\govman.exe" set "BINARY_FOUND=1"
if exist "%USERPROFILE%\.govman" set "DATA_FOUND=1"

REM Check if govman is in PATH
govman --version >nul 2>&1
if !errorlevel!==0 set "COMMAND_FOUND=1"

if !BINARY_FOUND!==1 (
    echo.
    call :print_separator "-"
    echo %BOLD%%WHITE%Existing Installation Detected:%RESET%
    call :print_separator "-"
    echo %GREEN% %CHECKMARK%%RESET% Binary found in: %BOLD%%USERPROFILE%\.govman\bin%RESET%

    if !COMMAND_FOUND!==1 (
        for /f "tokens=*" %%i in ('govman --version 2^>nul') do set "VERSION=%%i"
        echo %GREEN% %CHECKMARK%%RESET% Command available: %BOLD%govman%RESET% %DIM%(!VERSION!)%RESET%
    )

    if !DATA_FOUND!==1 (
        echo %BLUE% %INFO%%RESET% Data directory: %BOLD%%USERPROFILE%\.govman%RESET%
    )

    call :print_separator "-"
    echo.
    call :print_warning "govman is already installed on this system!"
    echo.
    call :print_separator "-"
    echo %BOLD%%WHITE%What you can do:%RESET%
    echo  • Run 'govman --version' to check current version
    echo  • Run 'govman --help' to see available commands
    echo  • Use the uninstaller script first if you need to reinstall
    echo  • Check 'govman list' to see available Go versions
    call :print_separator "-"
    echo.
    call :print_separator "="
    echo %DIM%%GRAY%Installation cancelled - govman already exists%RESET%
    call :print_separator "="
    echo.
    set "ALREADY_INSTALLED=1"
    exit /b 0
)

call :print_success "No existing installation found - proceeding with fresh install"
echo.
exit /b 0

:detect_platform
call :print_step "Detecting system platform..."

REM Detect architecture
set "ARCH=amd64"
if /i "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "ARCH=arm64"
if /i "%PROCESSOR_ARCHITEW6432%"=="ARM64" set "ARCH=arm64"
if /i "%PROCESSOR_ARCHITECTURE%"=="x86" if "%PROCESSOR_ARCHITEW6432%"=="" set "ARCH=386"

set "PLATFORM=windows/!ARCH!"
call :print_success "Detected platform: %BOLD%!PLATFORM!%RESET%"
echo.
exit /b 0

:get_latest_version
call :print_step "Fetching latest version information..."

if defined SPECIFIC_VERSION (
    set "VERSION=!SPECIFIC_VERSION!"
	call :validate_version
	if !errorlevel! neq 0 exit /b !errorlevel!
    call :print_success "Using specified version: %BOLD%!VERSION!%RESET%"
    echo.
    exit /b 0
)

REM Try to get latest version using curl or PowerShell
curl --version >nul 2>&1
if !errorlevel!==0 (
    REM Use curl if available
    for /f "delims=" %%i in ('curl --fail --silent --show-error --location --max-time 30 -H "Accept: application/vnd.github+json" -H "User-Agent: govman-installer" https://api.github.com/repos/justjundana/govman/releases/latest ^| findstr "tag_name" ^| for /f "tokens=2 delims=:, " %%j in ^("%%i"^) do echo %%~j') do set "VERSION=%%~i"
) else (
    REM Fallback to PowerShell
	for /f "delims=" %%i in ('powershell -NoProfile -Command "$h=@{'User-Agent'='govman-installer';'Accept'='application/vnd.github+json'}; (Invoke-RestMethod -Headers $h -TimeoutSec 30 https://api.github.com/repos/justjundana/govman/releases/latest).tag_name" 2^>nul') do set "VERSION=%%i"
)

if "!VERSION!"=="" (
    call :print_error "Failed to get latest version information"
    call :print_info "Please check your internet connection or use --version flag"
    exit /b 1
)

call :validate_version
if !errorlevel! neq 0 exit /b !errorlevel!

call :print_success "Latest version: %BOLD%!VERSION!%RESET%"
echo.
exit /b 0

:validate_version
powershell -NoProfile -Command "if ($env:VERSION -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)[0-9]*)?$') { exit 1 }" >nul 2>&1
if !errorlevel! neq 0 (
	call :print_error "Invalid release version: !VERSION!"
	exit /b 1
)
exit /b 0

:show_system_info
if %QUIET_MODE%==1 goto :eof
call :print_separator "-"
echo %BOLD%%WHITE%System Information:%RESET%
call :print_separator "-"
echo %GREEN% %CHECKMARK%%RESET% Operating System: %BOLD%Windows%RESET%
echo %GREEN% %CHECKMARK%%RESET% Architecture: %BOLD%!ARCH!%RESET%
echo %GREEN% %CHECKMARK%%RESET% Version: %BOLD%!VERSION!%RESET%
echo %BLUE% %INFO%%RESET% Install Directory: %BOLD%!INSTALL_DIR!%RESET%
call :print_separator "-"
echo.
goto :eof

:download_binary
call :print_step "Downloading govman !VERSION! for !PLATFORM!..."

set "ASSET_NAME=govman-windows-!ARCH!.exe"
set "RELEASE_BASE=https://github.com/justjundana/govman/releases/download/!VERSION!"
set "DOWNLOAD_URL=!RELEASE_BASE!/!ASSET_NAME!"
set "CHECKSUM_URL=!RELEASE_BASE!/checksums.txt"
set "BINARY_PATH=!INSTALL_DIR!\govman-real.exe"
set "TEMP_BINARY=!INSTALL_DIR!\.govman-download-!RANDOM!-!RANDOM!.exe"
set "TEMP_CHECKSUMS=!INSTALL_DIR!\.govman-checksums-!RANDOM!-!RANDOM!.txt"
set "BACKUP_PATH=!BINARY_PATH!.bak.!RANDOM!"

call :print_info "Download URL: !DOWNLOAD_URL!"

REM Create install directory
if not exist "!INSTALL_DIR!" mkdir "!INSTALL_DIR!"

REM Show progress (simplified for batch)
if %QUIET_MODE%==0 echo    Downloading govman binary...

curl --version >nul 2>&1
if !errorlevel!==0 (
	curl --fail --silent --show-error --location --max-time 120 -o "!TEMP_BINARY!" "!DOWNLOAD_URL!"
	if !errorlevel! neq 0 goto :download_failed
	curl --fail --silent --show-error --location --max-time 30 -o "!TEMP_CHECKSUMS!" "!CHECKSUM_URL!"
	if !errorlevel! neq 0 goto :download_failed
) else (
	powershell -NoProfile -Command "$ErrorActionPreference='Stop'; $h=@{'User-Agent'='govman-installer'}; Invoke-WebRequest -UseBasicParsing -Headers $h -TimeoutSec 120 -Uri $env:DOWNLOAD_URL -OutFile $env:TEMP_BINARY; Invoke-WebRequest -UseBasicParsing -Headers $h -TimeoutSec 30 -Uri $env:CHECKSUM_URL -OutFile $env:TEMP_CHECKSUMS" >nul 2>&1
	if !errorlevel! neq 0 goto :download_failed
)

set "EXPECTED_CHECKSUM="
set "CHECKSUM_COUNT=0"
for /f "usebackq tokens=1,2" %%A in ("!TEMP_CHECKSUMS!") do (
	set "MANIFEST_NAME=%%B"
	if "!MANIFEST_NAME:~0,1!"=="*" set "MANIFEST_NAME=!MANIFEST_NAME:~1!"
	if /i "!MANIFEST_NAME!"=="!ASSET_NAME!" (
		set /a CHECKSUM_COUNT+=1
		set "EXPECTED_CHECKSUM=%%A"
	)
)
if not "!CHECKSUM_COUNT!"=="1" (
	call :print_error "Checksum manifest must contain exactly one entry for !ASSET_NAME!"
	goto :download_cleanup_error
)

for /f "delims=" %%H in ('powershell -NoProfile -Command "(Get-FileHash -LiteralPath $env:TEMP_BINARY -Algorithm SHA256).Hash.ToLowerInvariant()"') do set "ACTUAL_CHECKSUM=%%H"
if /i not "!ACTUAL_CHECKSUM!"=="!EXPECTED_CHECKSUM!" (
	call :print_error "Checksum verification failed for !ASSET_NAME!"
	goto :download_cleanup_error
)

powershell -NoProfile -Command "$o=^& $env:TEMP_BINARY --version 2^>^&1 ^| Out-String; $v=[regex]::Escape($env:VERSION.TrimStart('v')); if ($LASTEXITCODE -ne 0 -or $o -notmatch ('(^|[^0-9])v?' + $v + '([^0-9]|$)')) { exit 1 }" >nul 2>&1
if !errorlevel! neq 0 (
	call :print_error "Downloaded binary is invalid or reports the wrong version"
	goto :download_cleanup_error
)

if exist "!BINARY_PATH!" move /y "!BINARY_PATH!" "!BACKUP_PATH!" >nul
move /y "!TEMP_BINARY!" "!BINARY_PATH!" >nul
if !errorlevel! neq 0 (
	if exist "!BACKUP_PATH!" move /y "!BACKUP_PATH!" "!BINARY_PATH!" >nul
	call :print_error "Failed to install govman binary; previous binary was restored"
	goto :download_cleanup_error
)
if exist "!BACKUP_PATH!" del /q "!BACKUP_PATH!"
if exist "!TEMP_CHECKSUMS!" del /q "!TEMP_CHECKSUMS!"
call :print_success "Downloaded and verified govman binary at !BINARY_PATH!"
echo.
exit /b 0

:download_failed
call :print_error "Failed to download govman binary or checksum manifest"

:download_cleanup_error
if exist "!TEMP_BINARY!" del /q "!TEMP_BINARY!"
if exist "!TEMP_CHECKSUMS!" del /q "!TEMP_CHECKSUMS!"
exit /b 1

:add_to_path
call :print_step "Configuring Windows environment..."

REM PowerShell reads the registry value directly so cmd delayed expansion never
REM observes or corrupts PATH entries containing !, &, spaces, or %VAR%.
powershell -NoProfile -Command "$ErrorActionPreference='Stop'; function N([string]$p) { if ($null -eq $p) { return '' }; return $p.Trim().TrimEnd([char[]]@('\','/')) }; $entry=$env:INSTALL_DIR; $key=[Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment'); if ($null -eq $key) { throw 'Unable to open HKCU\Environment' }; try { $had=@($key.GetValueNames()) -contains 'Path'; $old=if($had){[string]$key.GetValue('Path','',[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)}else{''}; $kind=if($had){$key.GetValueKind('Path')}else{[Microsoft.Win32.RegistryValueKind]::ExpandString}; $found=$false; foreach($part in $old.Split([char[]]@(';'),[StringSplitOptions]::None)){if([StringComparer]::OrdinalIgnoreCase.Equals((N $part),(N $entry))){$found=$true}}; if(-not $found){$new=if([string]::IsNullOrEmpty($old)){$entry}else{$old+';'+$entry}; $backup='Path.govman-backup-'+$PID+'-'+[Guid]::NewGuid().ToString('N'); $key.SetValue($backup,$old,$kind); try{$key.SetValue('Path',$new,$kind); $actual=[string]$key.GetValue('Path','',[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames); if($actual -cne $new -or $key.GetValueKind('Path') -ne $kind){throw 'PATH verification failed'}}catch{if($had){$key.SetValue('Path',$old,$kind)}else{$key.DeleteValue('Path',$false)};throw}finally{$key.DeleteValue($backup,$false)}} } finally { $key.Dispose() }" >nul 2>&1
if !errorlevel! neq 0 (
    call :print_error "Failed to update PATH; the original registry value was restored"
    exit /b 1
)
call :print_success "User PATH contains an exact govman entry"

REM Try to run govman init
"!BINARY_PATH!" init --force --shell cmd >nul 2>&1
if !errorlevel!==0 (
    call :print_success "Shell configuration completed successfully"
) else (
    call :print_error "Shell configuration failed"
    exit /b 1
)

echo.
exit /b 0

:verify_installation
call :print_step "Verifying installation..."

"!BINARY_PATH!" --version >nul 2>&1
if !errorlevel!==0 (
    for /f "tokens=*" %%i in ('"!BINARY_PATH!" --version 2^>nul') do set "INSTALLED_VERSION=%%i"
    call :print_success "Installation verified: %BOLD%!INSTALLED_VERSION!%RESET%"
) else (
    call :print_error "Installation verification failed"
    exit /b 1
)
echo.
exit /b 0

:show_completion
echo.
call :print_separator "="
echo.
echo %GREEN%%BOLD% INSTALLATION SUCCESSFUL!%RESET%
echo.
call :print_separator "-"
echo %BOLD%%WHITE%What was installed:%RESET%
echo  • govman binary and executable
echo  • Windows PATH configuration
echo  • Environment setup complete
call :print_separator "-"
echo %BOLD%%WHITE%Next Steps:%RESET%
echo  1. Restart your Command Prompt
echo  2. Verify with 'govman --version'
echo  3. Get started with 'govman --help'
call :print_separator "-"
echo %BOLD%%WHITE%Quick Commands:%RESET%
echo  • govman list         - List available Go versions
echo  • govman install 1.25 - Install Go 1.25
echo  • govman use 1.25     - Switch to Go 1.25
call :print_separator "-"
echo Welcome to govman!
call :print_separator "="
echo.
goto :eof
