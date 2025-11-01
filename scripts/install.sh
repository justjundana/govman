#!/usr/bin/env bash
 # govman installation script
# This script installs govman to $HOME/.govman/bin and adds it to PATH
 set -e
 # Global flags
QUIET_MODE=false
SPECIFIC_VERSION=""
 # Parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --quiet|-q)
                QUIET_MODE=true
                shift
                ;;
            --version|-v)
                SPECIFIC_VERSION="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}
 # Show help information
show_help() {
    echo "govman installer - Go Version Manager Installation Script"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  --quiet, -q         Run in quiet mode (minimal output)"
    echo "  --version, -v VER   Install specific version (e.g., v1.0.0)"
    echo "  --help, -h          Show this help message"
    echo
    echo "Examples:"
    echo "  $0                  # Install latest version"
    echo "  $0 --quiet          # Install quietly"
    echo "  $0 --version v1.0.0 # Install specific version"
}
 # Colors and styles
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
GRAY='\033[0;90m'
NC='\033[0m'
 # Style effects
BOLD='\033[1m'
DIM='\033[2m'
UNDERLINE='\033[4m'
BLINK='\033[5m'
 # Unicode characters for better UI
CHECKMARK="✓"
CROSSMARK="✗"
ARROW="→"
DOWNLOAD="⬇"
WARNING="⚠"
INSTALL="📦"
INFO="ℹ"
ROCKET="🚀"
GEAR="⚙"
 # Terminal width detection
TERM_WIDTH=$(tput cols 2>/dev/null || echo 80)
 # Print separator line
print_separator() {
    local char="${1:--}"
    printf "%*s\n" "$TERM_WIDTH" | tr ' ' "$char"
}
 # Print fancy header
print_header() {
    [[ "$QUIET_MODE" == "true" ]] && return
    clear
    print_separator "═"
    echo
    echo
    echo '    ██╗███╗   ██╗███████╗████████╗ █████╗ ██╗     ██╗     ███████╗██████╗'
    echo '    ██║████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██║     ██║     ██╔════╝██╔══██╗'
    echo '    ██║██╔██╗ ██║███████╗   ██║   ███████║██║     ██║     █████╗  ██████╔╝'
    echo '    ██║██║╚██╗██║╚════██║   ██║   ██╔══██║██║     ██║     ██╔══╝  ██╔══██╗'
    echo '    ██║██║ ╚████║███████║   ██║   ██║  ██║███████╗███████╗███████╗██║  ██║'
    echo '    ╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝'
    echo
    echo
    echo -e "${BOLD}${WHITE}                        Go Version Manager Installer${NC}"
    echo -e "${DIM}${GRAY}                    Fast and secure installation process${NC}"
    echo
    print_separator "═"
    echo
}
 # Print functions with icons and styling
print_info() {
    [[ "$QUIET_MODE" == "true" ]] && return
    echo -e "${BLUE}${BOLD} ${INFO}  INFO${NC} ${GRAY}│${NC} $1"
}
 print_success() {
    [[ "$QUIET_MODE" == "true" ]] && return
    echo -e "${GREEN}${BOLD} ${CHECKMARK}  SUCCESS${NC} ${GRAY}│${NC} $1"
}
 print_warning() {
    echo -e "${YELLOW}${BOLD} ${WARNING}  WARNING${NC} ${GRAY}│${NC} $1"
}
 print_error() {
    echo -e "${RED}${BOLD} ${CROSSMARK}  ERROR${NC} ${GRAY}│${NC} $1"
}
 print_step() {
    [[ "$QUIET_MODE" == "true" ]] && return
    echo -e "${PURPLE}${BOLD} ${ARROW}  STEP${NC} ${GRAY}│${NC} $1"
}
 print_install() {
    [[ "$QUIET_MODE" == "true" ]] && return
    echo -e "${CYAN}${BOLD} ${INSTALL}  INSTALLING${NC} ${GRAY}│${NC} $1"
}
 # Check if running on Windows (Git Bash)
is_windows() {
    [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]
}
 # Detect current shell and return appropriate config file
detect_shell_config() {
    local shell_name=""
    local config_file=""
         # Debug: Print environment variables for troubleshooting
    # echo "DEBUG: SHELL=$SHELL, ZSH_VERSION=$ZSH_VERSION, BASH_VERSION=$BASH_VERSION" >&2
         # First priority: Check the actual running shell from $SHELL
    case "$(basename "$SHELL")" in
        zsh)
            shell_name="zsh"
            config_file="~/.zshrc"
            ;;
        bash)
            shell_name="bash"
            if [[ "$OSTYPE" == "darwin"* ]]; then
                config_file="~/.bash_profile"
            else
                config_file="~/.bashrc"
            fi
            ;;
        fish)
            shell_name="fish"
            config_file="~/.config/fish/config.fish"
            ;;
        *)
            # Fallback: Check version variables (less reliable when running bash script in zsh)
            if [ -n "$ZSH_VERSION" ]; then
                shell_name="zsh"
                config_file="~/.zshrc"
            elif [ -n "$BASH_VERSION" ]; then
                shell_name="bash"
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    config_file="~/.bash_profile"
                else
                    config_file="~/.bashrc"
                fi
            elif [ -n "$FISH_VERSION" ]; then
                shell_name="fish"
                config_file="~/.config/fish/config.fish"
            else
                shell_name="shell"
                config_file="your shell's configuration file"
            fi
            ;;
    esac
         echo "${shell_name}:${config_file}"
}
 # Get restart instruction based on OS and shell
get_restart_instruction() {
    local shell_info
    shell_info=$(detect_shell_config)
    local shell_name=$(echo "$shell_info" | cut -d':' -f1)
    local config_file=$(echo "$shell_info" | cut -d':' -f2)
         if is_windows; then
        echo "Please restart your terminal or PowerShell window"
    else
        case "$shell_name" in
            bash|zsh)
                echo "Please restart your terminal or run 'source $config_file'"
                ;;
            fish)
                echo "Please restart your terminal or run 'source $config_file'"
                ;;
            *)
                echo "Please restart your terminal or reload your shell configuration"
                ;;
        esac
    fi
}
 # Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""
         case "$(uname -s)" in
        Linux*)     os=linux;;
        Darwin*)    os=darwin;;
        MINGW*)     os=windows;;
        MSYS*)      os=windows;;
        *)          print_error "Unsupported operating system"; exit 1;;
    esac
         case "$(uname -m)" in
        x86_64)     arch=amd64;;
        aarch64)    arch=arm64;;
        arm64)      arch=arm64;;
        armv7l)     arch=arm;;
        i386|i686) arch=386;;
        *)          print_error "Unsupported architecture"; exit 1;;
    esac
         # Special case for Windows
    if [[ "$os" == "windows" ]]; then
        arch=amd64  # Default to amd64 for Windows
    fi
         echo "${os}/${arch}"
}
 # Get shell configuration files
get_shell_configs() {
    local configs=("$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.zshrc")
    [[ -f "$HOME/.config/fish/config.fish" ]] && configs+=("$HOME/.config/fish/config.fish")
    printf '%s ' "${configs[@]}"
}
 # Get the latest release version from GitHub
get_latest_version() {
    # If specific version is requested, use it
    if [[ -n "$SPECIFIC_VERSION" ]]; then
        echo "$SPECIFIC_VERSION"
        return
    fi
     local version=""
    if command -v curl >/dev/null 2>&1; then
        version=$(curl -s https://api.github.com/repos/justjundana/govman/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- https://api.github.com/repos/justjundana/govman/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        print_error "Either curl or wget is required to download govman"
        exit 1
    fi
     if [[ -z "$version" ]]; then
        print_error "Failed to get latest version information"
        exit 1
    fi
     echo "$version"
}
 # Verify binary checksum (basic validation)
verify_binary() {
    local binary_path="$1"
    local binary_name="$(basename "$binary_path")"
     # Basic file validation
    if [[ ! -f "$binary_path" ]]; then
        print_error "Binary file not found: $binary_path"
        return 1
    fi
     # Check if file is executable
    if [[ ! -x "$binary_path" ]]; then
        print_error "Binary is not executable: $binary_path"
        return 1
    fi
     # Check file size (should be > 1MB for a Go binary)
    local file_size
    if command -v stat >/dev/null 2>&1; then
        case "$(uname -s)" in
            Darwin*) file_size=$(stat -f%z "$binary_path") ;;
            *) file_size=$(stat -c%s "$binary_path") ;;
        esac
         if [[ $file_size -lt 1048576 ]]; then  # Less than 1MB
            print_warning "Binary file seems unusually small ($file_size bytes)"
        fi
    fi
     # Try to get version to ensure it's a valid govman binary
    if ! "$binary_path" --version >/dev/null 2>&1; then
        print_error "Downloaded binary appears to be corrupted or invalid"
        return 1
    fi
     print_success "Binary validation completed"
    return 0
}
 # Animated loading for download process
show_download_progress() {
    [[ "$QUIET_MODE" == "true" ]] && return
    local item="$1"
    local delay=0.1
    local spinstr='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    local temp
     echo -n "   ${DIM}Downloading $item... ${NC}"
    for i in {1..15}; do
        temp=${spinstr#?}
        printf "\r   ${DIM}Downloading $item... ${CYAN}%c${NC} " "$spinstr"
        spinstr=$temp${spinstr%"$temp"}
        sleep $delay
    done
    printf "\r   ${GREEN}${CHECKMARK}${NC} Downloaded $item successfully.      \n"
}
 # Animated loading for installation process
show_install_progress() {
    [[ "$QUIET_MODE" == "true" ]] && return
    local item="$1"
    local delay=0.1
    local spinstr='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    local temp
     echo -n "   ${DIM}Installing $item... ${NC}"
    for i in {1..10}; do
        temp=${spinstr#?}
        printf "\r   ${DIM}Installing $item... ${PURPLE}%c${NC} " "$spinstr"
        spinstr=$temp${spinstr%"$temp"}
        sleep $delay
    done
    printf "\r   ${GREEN}${CHECKMARK}${NC} Installed $item successfully.      \n"
}
 # Download the binary
download_binary() {
    local version="$1"
    local platform="$2"
    local install_dir="$3"
         local os=$(echo "$platform" | cut -d'/' -f1)
    local arch=$(echo "$platform" | cut -d'/' -f2)
         # Construct binary name
    local binary_name="govman"
    if [[ "$os" == "windows" ]]; then
        binary_name="govman.exe"
    fi
         # Construct download URL
    local download_url="https://github.com/justjundana/govman/releases/download/${version}/govman-${os}-${arch}"
    if [[ "$os" == "windows" ]]; then
        download_url="${download_url}.exe"
    fi
         print_step "Downloading govman ${version} for ${platform}..."
    print_info "Download URL: $download_url"
         # Create install directory
    mkdir -p "$install_dir"
         # Download binary
    [[ "$QUIET_MODE" == "false" ]] && show_download_progress "govman binary"
     if command -v curl >/dev/null 2>&1; then
        if [[ "$QUIET_MODE" == "true" ]]; then
            curl -sSL -o "${install_dir}/${binary_name}" "$download_url"
        else
            curl -sSL -o "${install_dir}/${binary_name}" "$download_url"
        fi
    elif command -v wget >/dev/null 2>&1; then
        if [[ "$QUIET_MODE" == "true" ]]; then
            wget -qO "${install_dir}/${binary_name}" "$download_url"
        else
            wget -qO "${install_dir}/${binary_name}" "$download_url"
        fi
    else
        print_error "Either curl or wget is required to download govman"
        exit 1
    fi
     # Check if download was successful
    if [[ ! -f "${install_dir}/${binary_name}" ]]; then
        print_error "Failed to download govman binary"
        exit 1
    fi
     # Make binary executable
    chmod +x "${install_dir}/${binary_name}"
     # Validate the downloaded binary
    if ! verify_binary "${install_dir}/${binary_name}"; then
        print_error "Binary validation failed"
        rm -f "${install_dir}/${binary_name}"
        exit 1
    fi
         print_success "Downloaded govman binary to ${install_dir}/${binary_name}"
}
 # Add to PATH and initialize shell configuration
add_to_path() {
    local install_dir="$1"
    local govman_binary="${install_dir}/govman"
     # Ensure the govman binary is executable
    if [ ! -x "$govman_binary" ]; then
        print_error "govman binary not found or not executable at $govman_binary"
        exit 1
    fi
     print_step "Configuring shell environment..."
     # Show install progress animation
    [[ "$QUIET_MODE" == "false" ]] && show_install_progress "shell configuration"
     # Run `govman init` and capture its output
    # The `init` command will automatically detect the shell and provide setup instructions
    # We use `--force` to ensure it overwrites any existing configuration
    local init_output
    if init_output=$("$govman_binary" init --force 2>&1); then
        print_success "Shell configuration completed successfully"
        # The output of `govman init` will guide the user if manual steps are needed
        # For Unix-like shells, it will automatically update the config file
        # For PowerShell/cmd, it will print instructions
        if [[ -n "$init_output" ]]; then
            echo "$init_output"
        fi
    else
        print_error "Shell configuration failed. Please check the output below for details:"
        echo "$init_output"
        print_warning "You may need to run 'govman init' manually to complete the setup."
    fi
}
 # Show system information
show_system_info() {
    local platform="$1"
    local version="$2"
    local install_dir="$3"
         print_separator "┄"
    echo -e "${BOLD}${WHITE}System Information:${NC}"
    print_separator "┄"
         local os=$(echo "$platform" | cut -d'/' -f1)
    local arch=$(echo "$platform" | cut -d'/' -f2)
         # Capitalize first letter (compatible with older bash versions)
    local os_capitalized=$(echo "$os" | sed 's/./\U&/')
         echo -e "${GREEN} ${CHECKMARK}${NC} Operating System: ${BOLD}${os_capitalized}${NC}"
    echo -e "${GREEN} ${CHECKMARK}${NC} Architecture: ${BOLD}${arch}${NC}"
    echo -e "${GREEN} ${CHECKMARK}${NC} Version: ${BOLD}${version}${NC}"
    echo -e "${BLUE} ${INFO}${NC} Install Directory: ${BOLD}${install_dir}${NC}"
         print_separator "┄"
    echo
}
 # Show completion message
show_completion() {
    local version="$1"
    local restart_instruction="$2"
         echo
    print_separator "═"
    echo
    echo -e "${GREEN}${BOLD} ${ROCKET}  INSTALLATION SUCCESSFUL!${NC}"
    echo
    print_separator "┄"
    echo -e "${BOLD}${WHITE}What was installed:${NC}"
    echo " • govman binary and executable"
    echo " • Shell PATH configurations"
    echo " • Environment setup complete"
    print_separator "┄"
    echo -e "${BOLD}${WHITE}Next Steps:${NC}"
    echo " 1. $restart_instruction"
    echo " 2. Verify with 'govman --version'"
    echo " 3. Get started with 'govman --help'"
    print_separator "┄"
    echo -e "${BOLD}${WHITE}Quick Commands:${NC}"
    echo " • govman list         - List available Go versions"
    echo " • govman install 1.25 - Install Go 1.25"
    echo " • govman use 1.25     - Switch to Go 1.25"
    print_separator "┄"
    echo "Welcome to govman! 🎉"
    print_separator "═"
    echo
}
 # Check if govman is already installed
check_existing_installation() {
    local install_dir="$HOME/.govman/bin"
    local govman_dir="$HOME/.govman"
    local shell_configs_str
    shell_configs_str=$(get_shell_configs)
    local shell_configs=($shell_configs_str)
    local binary_found=false
    local config_found=false
    local command_found=false
         print_step "Checking for existing installation..."
         # Check binary directory
    if [[ -f "$install_dir/govman" ]]; then
        binary_found=true
    fi
         # Check shell configurations
    for shell_config in "${shell_configs[@]}"; do
        if [[ -f "$shell_config" ]] && grep -q "# GOVMAN - Go Version Manager" "$shell_config" 2>/dev/null; then
            config_found=true
            break
        fi
    done
         # Check if govman command is available in PATH
    if command -v govman >/dev/null 2>&1; then
        command_found=true
    fi
         # If any installation traces found, show details and exit
    if [[ "$binary_found" == true || "$config_found" == true || "$command_found" == true ]]; then
        echo
        print_separator "┄"
        echo -e "${BOLD}${WHITE}Existing Installation Detected:${NC}"
        print_separator "┄"
                 if [[ "$binary_found" == true ]]; then
            echo -e "${GREEN} ${CHECKMARK}${NC} Binary found: ${BOLD}$install_dir/govman${NC}"
        fi
                 if [[ "$config_found" == true ]]; then
            echo -e "${GREEN} ${CHECKMARK}${NC} Shell configuration: ${BOLD}Found in PATH${NC}"
        fi
                 if [[ "$command_found" == true ]]; then
            local version=$(govman --version 2>/dev/null | head -1 || echo "unknown")
            echo -e "${GREEN} ${CHECKMARK}${NC} Command available: ${BOLD}govman${NC} ${DIM}($version)${NC}"
        fi
                 if [[ -d "$govman_dir" ]]; then
            local dir_size=$(du -sh "$govman_dir" 2>/dev/null | cut -f1 || echo "unknown")
            echo -e "${BLUE} ${INFO}${NC} Data directory: ${BOLD}$govman_dir${NC} ${DIM}($dir_size)${NC}"
        fi
                 print_separator "┄"
        echo
        print_warning "govman is already installed on this system!"
        echo
        print_separator "┄"
        echo -e "${BOLD}${WHITE}What you can do:${NC}"
        echo " • Run 'govman --version' to check current version"
        echo " • Run 'govman --help' to see available commands"
        echo " • Use the uninstaller script first if you need to reinstall"
        echo " • Check 'govman list' to see available Go versions"
        print_separator "┄"
        echo
        print_separator "═"
        echo -e "${DIM}${GRAY}Installation cancelled - govman already exists${NC}"
        print_separator "═"
        echo
        exit 0
    else
        print_success "No existing installation found - proceeding with fresh install"
        echo
    fi
}
 # Main installation function
main() {
    # Parse command line arguments
    parse_arguments "$@"
     # Show header
    print_header
         print_info "Starting govman installation process..."
    echo
         # Check for existing installation first
    check_existing_installation
         # Detect platform
    print_step "Detecting system platform..."
    local platform
    platform=$(detect_platform)
    print_success "Detected platform: ${BOLD}$platform${NC}"
    echo
         # Get latest version
    print_step "Fetching latest version information..."
    local version
    version=$(get_latest_version)
    print_success "Latest version: ${BOLD}$version${NC}"
    echo
         # Set installation directory
    local install_dir="$HOME/.govman/bin"
    print_info "Installation directory: ${BOLD}$install_dir${NC}"
    echo
         # Show system info
    show_system_info "$platform" "$version" "$install_dir"
         # Download binary
    download_binary "$version" "$platform" "$install_dir"
    echo
         # Add to PATH
    add_to_path "$install_dir"
    echo
         # Verify installation
    print_step "Verifying installation..."
    if "$install_dir/govman" --version >/dev/null 2>&1; then
        local installed_version=$("$install_dir/govman" --version 2>/dev/null | head -1 || echo "unknown")
        print_success "Installation verified: ${BOLD}$installed_version${NC}"
                 # Dynamic restart instruction
        local restart_instruction
        restart_instruction=$(get_restart_instruction)
                 show_completion "$version" "$restart_instruction"
    else
        print_warning "Installation completed, but verification failed"
        echo
        print_separator "┄"
        echo -e "${BOLD}${WHITE}Manual Steps Required:${NC}"
        echo " 1. Restart your terminal"
        echo " 2. Try running 'govman --version'"
        echo " 3. If issues persist, run 'govman init' manually"
        print_separator "┄"
        echo
    fi
}
 # Trap to ensure clean exit
trap 'echo -e "\n${RED}Installation interrupted. Partial installation may have occurred.${NC}"; exit 1' INT TERM
 # Run main function
main "$@"