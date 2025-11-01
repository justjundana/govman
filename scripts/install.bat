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
if /i "%~1"=="--version" set SPECIFIC_VERSION=%~2 & shift & shift & goto :parse_args
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
set "RED=[0;31m"
set "GREEN=[0;32m"
set "YELLOW=[1;33m"
set "BLUE=[0;34m"
set "PURPLE=[0;35m"
set "CYAN=[0;36m"
set "WHITE=[1;37m"
set "GRAY=[0;90m"
set "RESET=[0m"
set "BOLD=[1m"
set "DIM=[2m"

REM Check if we're in a terminal that supports ANSI colors
REM For older Windows versions, we'll disable colors
ver | find "Version 10." >nul
if %errorlevel% neq 0 (
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
)

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

set "BINARY_FOUND=0"
set "COMMAND_FOUND=0"
set "DATA_FOUND=0"

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
    echo %GREEN% %CHECKMARK%%RESET% Binary found: %BOLD%%USERPROFILE%\.govman\bin\govman.exe%RESET%

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
    exit /b 1
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

set "PLATFORM=windows/!ARCH!"
call :print_success "Detected platform: %BOLD%!PLATFORM!%RESET%"
echo.
exit /b 0

:get_latest_version
call :print_step "Fetching latest version information..."

if defined SPECIFIC_VERSION (
    set "VERSION=!SPECIFIC_VERSION!"
    call :print_success "Using specified version: %BOLD%!VERSION!%RESET%"
    echo.
    exit /b 0
)

REM Try to get latest version using curl or PowerShell
curl --version >nul 2>&1
if !errorlevel!==0 (
    REM Use curl if available
    for /f "delims=" %%i in ('curl -s https://api.github.com/repos/justjundana/govman/releases/latest ^| findstr "tag_name" ^| for /f "tokens=2 delims=:, " %%j in ^("%%i"^) do echo %%~j') do set "VERSION=%%~i"
) else (
    REM Fallback to PowerShell
    for /f "delims=" %%i in ('powershell -Command "(Invoke-RestMethod https://api.github.com/repos/justjundana/govman/releases/latest).tag_name" 2^>nul') do set "VERSION=%%i"
)

if "!VERSION!"=="" (
    call :print_error "Failed to get latest version information"
    call :print_info "Please check your internet connection or use --version flag"
    exit /b 1
)

call :print_success "Latest version: %BOLD%!VERSION!%RESET%"
echo.
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

set "DOWNLOAD_URL=https://github.com/justjundana/govman/releases/download/!VERSION!/govman-windows-!ARCH!.exe"
set "BINARY_PATH=!INSTALL_DIR!\govman.exe"

call :print_info "Download URL: !DOWNLOAD_URL!"

REM Create install directory
if not exist "!INSTALL_DIR!" mkdir "!INSTALL_DIR!"

REM Show progress (simplified for batch)
if %QUIET_MODE%==0 echo    Downloading govman binary...

REM Download using curl or PowerShell
curl --version >nul 2>&1
if !errorlevel!==0 (
    curl -sSL -o "!BINARY_PATH!" "!DOWNLOAD_URL!"
) else (
    powershell -Command "Invoke-WebRequest -Uri '!DOWNLOAD_URL!' -OutFile '!BINARY_PATH!'" >nul 2>&1
)

if !errorlevel! neq 0 (
    call :print_error "Failed to download govman binary"
    exit /b 1
)

if not exist "!BINARY_PATH!" (
    call :print_error "Failed to download govman binary"
    exit /b 1
)

REM Basic validation - check if file exists and has reasonable size
for %%F in ("!BINARY_PATH!") do set "FILE_SIZE=%%~zF"
if !FILE_SIZE! lss 1048576 (
    call :print_warning "Binary file seems unusually small (!FILE_SIZE! bytes)"
)

call :print_success "Downloaded govman binary to !BINARY_PATH!"
echo.
exit /b 0

:add_to_path
call :print_step "Configuring Windows environment..."

REM Get current user PATH
for /f "tokens=2*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USER_PATH=%%B"

REM Check if install directory is already in PATH
echo "!USER_PATH!" | find "!INSTALL_DIR!" >nul
if !errorlevel!==0 (
    call :print_info "Install directory already in PATH"
) else (
    REM Add to PATH
    if defined USER_PATH (
        set "NEW_PATH=!USER_PATH!;!INSTALL_DIR!"
    ) else (
        set "NEW_PATH=!INSTALL_DIR!"
    )

    reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "!NEW_PATH!" /f >nul
    if !errorlevel!==0 (
        call :print_success "Added !INSTALL_DIR! to user PATH"
    ) else (
        call :print_error "Failed to update PATH"
        exit /b 1
    )
)

REM Try to run govman init
"!BINARY_PATH!" init --force >nul 2>&1
if !errorlevel!==0 (
    call :print_success "Shell configuration completed successfully"
) else (
    call :print_warning "Shell configuration had issues. You may need to run 'govman init' manually."
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
    call :print_warning "Installation completed, but verification failed"
    echo.
    call :print_separator "-"
    echo %BOLD%%WHITE%Manual Steps Required:%RESET%
    echo  1. Restart your Command Prompt
    echo  2. Try running 'govman --version'
    echo  3. If issues persist, run 'govman init' manually
    call :print_separator "-"
)
echo.
goto :eof

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