@echo off
setlocal enabledelayedexpansion

REM govman uninstallation script for Windows Command Prompt
REM This script removes govman from %USERPROFILE%\.govman\bin and removes it from PATH

REM Parse command line arguments
set SHOW_HELP=0

:parse_args
if "%~1"=="" goto :args_done
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
set "TRASH=DEL"
set "WARNING=!"
set "QUESTION=?"
set "STOP=STOP"
set "CLEAN=CLN"
set "SHIELD=KEEP"
set "INFO=i"

REM Main execution
call :print_header
call :print_info "Starting govman uninstallation process..."
echo.

call :check_govman_installation
set "INSTALLATION_FOUND=!errorlevel!"

if !INSTALLATION_FOUND!==0 (
    call :print_warning "govman does not appear to be installed on this system"
    echo.
    call :print_separator "-"
    echo %BOLD%%WHITE%No govman installation found!%RESET%
    call :print_separator "-"
    echo It looks like govman is not installed or has already been removed.
    echo Common reasons:
    echo  • govman was never installed
    echo  • govman was already uninstalled
    echo  • govman was installed in a different location
    echo  • Installation was incomplete or corrupted
    call :print_separator "-"
    echo.
    set /p "RESPONSE=Do you want to clean any remaining traces? (y/N): "
    if /i "!RESPONSE!" neq "y" (
        echo.
        call :print_info "Exiting without making changes"
        call :print_separator "="
        echo %DIM%%GRAY%No changes were made to your system.%RESET%
        call :print_separator "="
        echo.
        exit /b 0
    )
    echo.
    call :print_info "Proceeding with cleanup of any remaining traces..."
    echo.
) else (
    call :print_success "govman installation detected"
    echo.
)

call :show_uninstall_options

set /p "RESPONSE=Choose an option (1/2/3): "
echo.

if "!RESPONSE!"=="1" (
    call :minimal_removal
) else if "!RESPONSE!"=="2" (
    call :complete_removal
) else (
    echo.
    call :print_info "Uninstallation cancelled by user"
    call :print_separator "="
    echo %DIM%%GRAY%No changes were made to your system.%RESET%
    call :print_separator "="
    echo.
)

goto :eof

REM Functions start here

:show_help
echo govman uninstaller - Go Version Manager Uninstallation Script for Windows
echo.
echo Usage: %~nx0 [OPTIONS]
echo.
echo Options:
echo   --help, -h          Show this help message
echo.
echo Examples:
echo   %~nx0               # Run interactive uninstaller
echo   %~nx0 --help        # Show help
goto :eof

:print_header
cls
call :print_separator "="
echo.
echo.
echo     ██╗   ██╗███╗   ██╗██╗███╗   ██╗███████╗████████╗ █████╗ ██╗     ██╗
echo     ██║   ██║████╗  ██║██║████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██║     ██║
echo     ██║   ██║██╔██╗ ██║██║██╔██╗ ██║███████╗   ██║   ███████║██║     ██║
echo     ██║   ██║██║╚██╗██║██║██║╚██╗██║╚════██║   ██║   ██╔══██║██║     ██║
echo     ╚██████╔╝██║ ╚████║██║██║ ╚████║███████║   ██║   ██║  ██║███████╗███████╗
echo      ╚═════╝ ╚═╝  ╚═══╝╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝
echo.
echo.
echo %BOLD%%WHITE%                        Go Version Manager Uninstaller%RESET%
echo %DIM%%GRAY%                  Safe and complete uninstallation process%RESET%
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
echo %BLUE%%BOLD% %INFO%  INFO%RESET% %GRAY%^|%RESET% %~1
goto :eof

:print_success
echo %GREEN%%BOLD% %CHECKMARK%  SUCCESS%RESET% %GRAY%^|%RESET% %~1
goto :eof

:print_warning
echo %YELLOW%%BOLD% %WARNING%  WARNING%RESET% %GRAY%^|%RESET% %~1
goto :eof

:print_error
echo %RED%%BOLD% %CROSSMARK%  ERROR%RESET% %GRAY%^|%RESET% %~1
goto :eof

:print_step
echo %PURPLE%%BOLD% %ARROW%  STEP%RESET% %GRAY%^|%RESET% %~1
goto :eof

:print_clean
echo %CYAN%%BOLD% %CLEAN%  CLEANING%RESET% %GRAY%^|%RESET% %~1
goto :eof

:print_question
echo %YELLOW%%BOLD% %QUESTION%  QUESTION%RESET% %GRAY%^|%RESET% %~1
goto :eof

:check_govman_installation
call :print_step "Checking govman installation..."

set "BINARY_FOUND=0"
set "PATH_FOUND=0"
set "COMMAND_FOUND=0"
set "DATA_FOUND=0"

REM Check binary directory
if exist "%USERPROFILE%\.govman\bin\govman.exe" set "BINARY_FOUND=1"

REM Check if govman is in PATH
for /f "tokens=2*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USER_PATH=%%B"
echo "!USER_PATH!" | find "%USERPROFILE%\.govman\bin" >nul
if !errorlevel!==0 set "PATH_FOUND=1"

REM Check if govman command works
govman --version >nul 2>&1
if !errorlevel!==0 set "COMMAND_FOUND=1"

REM Check data directory
if exist "%USERPROFILE%\.govman" set "DATA_FOUND=1"

echo.
call :print_separator "-"
echo %BOLD%%WHITE%Installation Status:%RESET%
call :print_separator "-"

if !BINARY_FOUND!==1 (
    echo %GREEN% %CHECKMARK%%RESET% Binary directory: %BOLD%%USERPROFILE%\.govman\bin%RESET%
) else (
    echo %GRAY% %CROSSMARK%%RESET% Binary directory: %DIM%%USERPROFILE%\.govman\bin (not found)%RESET%
)

if !PATH_FOUND!==1 (
    echo %GREEN% %CHECKMARK%%RESET% PATH configuration: %BOLD%Found in user PATH%RESET%
) else (
    echo %GRAY% %CROSSMARK%%RESET% PATH configuration: %DIM%No govman PATH found%RESET%
)

if !COMMAND_FOUND!==1 (
    for /f "tokens=*" %%i in ('govman --version 2^>nul') do set "VERSION=%%i"
    echo %GREEN% %CHECKMARK%%RESET% Command available: %BOLD%govman%RESET% %DIM%(!VERSION!)%RESET%
) else (
    echo %GRAY% %CROSSMARK%%RESET% Command available: %DIM%govman (not found)%RESET%
)

if !DATA_FOUND!==1 (
    echo %BLUE% %INFO%%RESET% Data directory: %BOLD%%USERPROFILE%\.govman%RESET%
) else (
    echo %GRAY% %CROSSMARK%%RESET% Data directory: %DIM%%USERPROFILE%\.govman (not found)%RESET%
)

call :print_separator "-"
echo.

REM Return 1 if something to uninstall, 0 if nothing found
if !BINARY_FOUND!==1 exit /b 1
if !PATH_FOUND!==1 exit /b 1
if !DATA_FOUND!==1 exit /b 1
exit /b 0

:show_uninstall_options
call :print_separator "="
echo %BOLD%%WHITE% %QUESTION%  UNINSTALLATION OPTIONS%RESET%
call :print_separator "="
echo.
echo %CYAN%%BOLD%1)%RESET% %WHITE%Minimal Removal%RESET% %DIM%(Recommended)%RESET%
echo    • Remove govman binary and executable
echo    • Clean PATH configuration
echo    • %GREEN%Keep%RESET% downloaded Go versions for future use
echo.
echo %RED%%BOLD%2)%RESET% %WHITE%Complete Removal%RESET% %DIM%(Permanent)%RESET%
echo    • Remove govman binary and executable
echo    • Clean PATH configuration
echo    • %RED%Delete%RESET% all downloaded Go versions and data
echo    • %RED%Delete%RESET% entire .govman directory
echo.
echo %GRAY%%BOLD%3)%RESET% %WHITE%Cancel%RESET%
echo    • Exit without making any changes
echo.
call :print_separator "-"
goto :eof

:show_removal_preview
set "OPTION=%~1"

echo %BOLD%%WHITE%Removal Preview:%RESET%
call :print_separator "-"

REM Check binary
if exist "%USERPROFILE%\.govman\bin\govman.exe" (
    echo %RED% %TRASH%%RESET% Binary directory: %BOLD%%USERPROFILE%\.govman\bin%RESET%
) else (
    echo %GRAY% %CROSSMARK%%RESET% Binary directory: %DIM%%USERPROFILE%\.govman\bin (not found)%RESET%
)

REM Check PATH configuration
for /f "tokens=2*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USER_PATH=%%B"
echo "!USER_PATH!" | find "%USERPROFILE%\.govman\bin" >nul
if !errorlevel!==0 (
    echo %RED% %TRASH%%RESET% PATH configuration: %BOLD%User PATH entry%RESET%
) else (
    echo %GRAY% %CROSSMARK%%RESET% PATH configuration: %DIM%No govman PATH found%RESET%
)

REM Show data directory based on option
if exist "%USERPROFILE%\.govman" (
    if "%OPTION%"=="complete" (
        echo %RED% %TRASH%%RESET% Data directory: %BOLD%%USERPROFILE%\.govman%RESET%
    ) else (
        echo %GREEN% %SHIELD%%RESET% Data directory: %BOLD%%USERPROFILE%\.govman%RESET% %DIM%(will be kept)%RESET%
    )
) else (
    echo %GRAY% %CROSSMARK%%RESET% Data directory: %DIM%%USERPROFILE%\.govman (not found)%RESET%
)

call :print_separator "-"
echo.
goto :eof

:minimal_removal
call :print_info "Proceeding with minimal removal..."
echo.
call :show_removal_preview "minimal"

call :print_separator "-"
echo %YELLOW%%BOLD% %STOP%  FINAL CONFIRMATION%RESET%
call :print_separator "-"
set /p "CONFIRM=Proceed with minimal removal? (y/N): "

if /i "!CONFIRM!"=="y" (
    echo.
    call :remove_binary
    echo.
    call :remove_from_path
    echo.
    call :show_completion "false"
) else (
    echo.
    call :print_info "Uninstallation cancelled by user"
    call :print_separator "="
    echo %DIM%%GRAY%No changes were made to your system.%RESET%
    call :print_separator "="
    echo.
)
goto :eof

:complete_removal
call :print_info "Proceeding with complete removal..."
echo.
call :show_removal_preview "complete"

call :print_separator "-"
echo %RED%%BOLD% %STOP%  DANGER: COMPLETE REMOVAL%RESET%
call :print_separator "-"
echo %RED%This will permanently delete ALL govman data and cannot be undone!%RESET%
call :print_separator "-"
set /p "CONFIRM=Type 'DELETE' to confirm complete removal: "

if "!CONFIRM!"=="DELETE" (
    echo.
    call :remove_binary
    echo.
    call :remove_from_path
    echo.
    call :remove_govman_dir
    echo.
    call :show_completion "true"
) else (
    echo.
    call :print_info "Complete removal cancelled - confirmation text did not match"
    call :print_separator "="
    echo %DIM%%GRAY%No changes were made to your system.%RESET%
    call :print_separator "="
    echo.
)
goto :eof

:remove_binary
call :print_step "Removing govman binary..."

if exist "%USERPROFILE%\.govman\bin" (
    echo    Removing binary directory...
    rmdir /s /q "%USERPROFILE%\.govman\bin" 2>nul
    if !errorlevel!==0 (
        call :print_success "Removed govman binary from %USERPROFILE%\.govman\bin"
    ) else (
        call :print_error "Failed to remove binary directory"
    )
) else (
    call :print_warning "govman binary directory not found"
)
goto :eof

:remove_from_path
call :print_step "Cleaning PATH configuration..."

REM Get current user PATH
for /f "tokens=2*" %%A in ('reg query "HKCU\Environment" /v PATH 2^>nul') do set "USER_PATH=%%B"

REM Check if govman path exists
echo "!USER_PATH!" | find "%USERPROFILE%\.govman\bin" >nul
if !errorlevel!==0 (
    echo    Removing PATH configuration...

    REM Remove govman path from PATH string
    set "NEW_PATH=!USER_PATH!"
    set "NEW_PATH=!NEW_PATH:;%USERPROFILE%\.govman\bin=!"
    set "NEW_PATH=!NEW_PATH:%USERPROFILE%\.govman\bin;=!"
    set "NEW_PATH=!NEW_PATH:%USERPROFILE%\.govman\bin=!"

    REM Update registry
    reg add "HKCU\Environment" /v PATH /t REG_EXPAND_SZ /d "!NEW_PATH!" /f >nul
    if !errorlevel!==0 (
        call :print_success "Cleaned PATH configuration"
    ) else (
        call :print_error "Failed to update PATH"
    )
) else (
    call :print_info "No govman PATH configuration found"
)
goto :eof

:remove_govman_dir
call :print_step "Removing govman data directory..."

if exist "%USERPROFILE%\.govman" (
    call :print_info "Removing directory: %USERPROFILE%\.govman"
    echo    Removing data directory...
    rmdir /s /q "%USERPROFILE%\.govman" 2>nul
    if !errorlevel!==0 (
        call :print_success "Removed govman data directory"
    ) else (
        call :print_error "Failed to remove data directory"
    )
) else (
    call :print_warning "govman directory not found"
)
goto :eof

:show_completion
set "COMPLETE=%~1"

echo.
call :print_separator "="
echo.
if "%COMPLETE%"=="true" (
    echo %GREEN%%BOLD% %CHECKMARK%  COMPLETE UNINSTALLATION SUCCESSFUL!%RESET%
    echo.
    call :print_separator "-"
    echo %BOLD%%WHITE%What was removed:%RESET%
    echo  • govman binary and executable
    echo  • PATH configuration
    echo  • All downloaded Go versions
    echo  • Complete .govman directory
) else (
    echo %GREEN%%BOLD% %CHECKMARK%  MINIMAL UNINSTALLATION COMPLETE!%RESET%
    echo.
    call :print_separator "-"
    echo %BOLD%%WHITE%What was removed:%RESET%
    echo  • govman binary and executable
    echo  • PATH configuration
    echo.
    echo %BOLD%%WHITE%What was kept:%RESET%
    echo  • Downloaded Go versions in .govman directory
)
call :print_separator "-"
echo %BOLD%%WHITE%Final Steps:%RESET%
echo  1. Restart your Command Prompt to complete the process
echo  2. Verify with 'govman --version' (should show 'not recognized')
if "%COMPLETE%" neq "true" (
    echo  3. Manually remove '.govman' directory if you change your mind later
)
call :print_separator "-"
echo Thank you for using govman!
call :print_separator "="
echo.
goto :eof