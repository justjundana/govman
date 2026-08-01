#!/usr/bin/env bash
 # govman installation script
# This script installs govman to $HOME/.govman/bin and adds it to PATH
set -euo pipefail
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
				if [[ $# -lt 2 ]]; then
					print_error "Missing value for $1"
					exit 1
				fi
                if [[ "$2" == -* ]]; then
                    print_error "Missing value for $1"
                    exit 1
                fi
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
 # Unicode characters for better UI
CHECKMARK="✓"
CROSSMARK="✗"
ARROW="→"
WARNING="⚠"
INSTALL="📦"
INFO="ℹ"
ROCKET="🚀"
 # Terminal width detection
TERM_WIDTH=$(tput cols 2>/dev/null || echo 80)
 # Print separator line
print_separator() {
    local char="${1:--}"
    printf "%*s\n" "$TERM_WIDTH" "" | tr ' ' "$char"
}
 # Print fancy header
print_header() {
    [[ "$QUIET_MODE" == "true" ]] && return
    if [[ -t 1 && "${TERM:-}" != "dumb" ]]; then
        clear 2>/dev/null || true
    fi
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
    [[ "${OSTYPE:-}" == msys* || "${OSTYPE:-}" == cygwin* || "${OSTYPE:-}" == win32* ]]
}
 # Detect current shell and return appropriate config file
# The ~ below is deliberate: config_file is only ever echoed back to the user as
# something they will type (e.g. "source ~/.zshrc"), never used to open a file.
# shellcheck disable=SC2088
detect_shell_config() {
    local shell_name=""
    local config_file=""
         # Debug: Print environment variables for troubleshooting
    # echo "DEBUG: SHELL=$SHELL, ZSH_VERSION=$ZSH_VERSION, BASH_VERSION=$BASH_VERSION" >&2
         # First priority: Check the actual running shell from $SHELL
    case "$(basename "${SHELL:-}")" in
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
            if [ -n "${ZSH_VERSION:-}" ]; then
                shell_name="zsh"
                config_file="~/.zshrc"
            elif [ -n "$BASH_VERSION" ]; then
                shell_name="bash"
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    config_file="~/.bash_profile"
                else
                    config_file="~/.bashrc"
                fi
            elif [ -n "${FISH_VERSION:-}" ]; then
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
    local shell_name config_file
    shell_name=$(echo "$shell_info" | cut -d':' -f1)
    config_file=$(echo "$shell_info" | cut -d':' -f2)
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
		CYGWIN*)    os=windows;;
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
	         echo "${os}/${arch}"
}

validate_release_version() {
	local version="$1"
	if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)[0-9]*)?$ ]]; then
		print_error "Invalid release version: $version"
		return 1
	fi
}
# Get shell configuration files without serializing paths through word splitting.
SHELL_CONFIGS=()
get_shell_configs() {
    SHELL_CONFIGS=(
        "$HOME/.bashrc"
        "$HOME/.bash_profile"
        "$HOME/.profile"
        "$HOME/.zshrc"
        "$HOME/.config/fish/config.fish"
    )
}
 # Get the latest release version from GitHub
get_latest_version() {
    # If specific version is requested, use it
    if [[ -n "$SPECIFIC_VERSION" ]]; then
		validate_release_version "$SPECIFIC_VERSION" || exit 1
        echo "$SPECIFIC_VERSION"
        return
    fi
     local version=""
    if command -v curl >/dev/null 2>&1; then
		version=$(curl --fail --silent --show-error --location --max-time 30 \
			-H "Accept: application/vnd.github+json" -H "User-Agent: govman-installer" \
			https://api.github.com/repos/justjundana/govman/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
		version=$(wget --quiet --https-only --timeout=30 -O- \
			--header="Accept: application/vnd.github+json" --header="User-Agent: govman-installer" \
			https://api.github.com/repos/justjundana/govman/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        print_error "Either curl or wget is required to download govman"
        exit 1
    fi
    if [[ -z "$version" ]]; then
        print_error "Failed to get latest version information"
        exit 1
    fi
	validate_release_version "$version" || exit 1
     echo "$version"
}

calculate_sha256() {
	local file_path="$1"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$file_path" | awk '{print $1}'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$file_path" | awk '{print $1}'
	elif command -v openssl >/dev/null 2>&1; then
		openssl dgst -sha256 "$file_path" | awk '{print $NF}'
	else
		print_error "sha256sum, shasum, or openssl is required to verify govman"
		return 1
	fi
}

 verify_binary() {
    local binary_path="$1"
	local expected_version="$2"
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
	local version_output
	if ! version_output=$("$binary_path" --version 2>&1); then
        print_error "Downloaded binary appears to be corrupted or invalid"
        return 1
    fi
	local normalized_version="${expected_version#v}"
	local escaped_version="${normalized_version//./\\.}"
	if ! printf '%s\n' "$version_output" | grep -Eq "(^|[^0-9])v?${escaped_version}([^0-9]|$)"; then
		print_error "Downloaded binary reports the wrong version"
		return 1
	fi
     print_success "Binary validation completed"
    return 0
}
 # Download the binary
download_binary() {
    local version="$1"
    local platform="$2"
    local install_dir="$3"
	local os
	local arch
	os=$(echo "$platform" | cut -d'/' -f1)
	arch=$(echo "$platform" | cut -d'/' -f2)
	local asset_name="govman-${os}-${arch}"
    local binary_name="govman"
    if [[ "$os" == "windows" ]]; then
		asset_name="${asset_name}.exe"
        binary_name="govman.exe"
    fi
	local release_base="https://github.com/justjundana/govman/releases/download/${version}"
	local download_url="${release_base}/${asset_name}"
	local checksum_url="${release_base}/checksums.txt"
	print_step "Downloading govman ${version} for ${platform}..."
    print_info "Download URL: $download_url"
    mkdir -p "$install_dir"

	local temp_binary
	local temp_checksums
	temp_binary=$(mktemp "${install_dir}/.govman-download.XXXXXX") || exit 1
	if [[ "$os" == "windows" ]]; then
		mv "$temp_binary" "${temp_binary}.exe"
		temp_binary="${temp_binary}.exe"
	fi
	temp_checksums=$(mktemp "${install_dir}/.govman-checksums.XXXXXX") || {
		rm -f "$temp_binary"
		exit 1
	}

	if command -v curl >/dev/null 2>&1; then
		if ! curl --fail --silent --show-error --location --max-time 120 -o "$temp_binary" "$download_url" ||
			! curl --fail --silent --show-error --location --max-time 30 -o "$temp_checksums" "$checksum_url"; then
			rm -f "$temp_binary" "$temp_checksums"
			print_error "Failed to download govman binary or checksum manifest"
			exit 1
		fi
    elif command -v wget >/dev/null 2>&1; then
		if ! wget --quiet --https-only --timeout=120 -O "$temp_binary" "$download_url" ||
			! wget --quiet --https-only --timeout=30 -O "$temp_checksums" "$checksum_url"; then
			rm -f "$temp_binary" "$temp_checksums"
			print_error "Failed to download govman binary or checksum manifest"
			exit 1
		fi
    else
		rm -f "$temp_binary" "$temp_checksums"
        print_error "Either curl or wget is required to download govman"
        exit 1
    fi

	local checksum_matches
	checksum_matches=$(awk -v name="$asset_name" '$2 == name || $2 == "*" name { print $1 }' "$temp_checksums")
	rm -f "$temp_checksums"
	local checksum_count
	checksum_count=$(printf '%s\n' "$checksum_matches" | awk 'NF { count++ } END { print count+0 }')
	if [[ "$checksum_count" -ne 1 ]]; then
		rm -f "$temp_binary"
		print_error "Checksum manifest must contain exactly one entry for $asset_name"
		exit 1
	fi
	local expected_checksum
	expected_checksum=$(printf '%s\n' "$checksum_matches" | awk 'NF { print tolower($1) }')
	if [[ ! "$expected_checksum" =~ ^[0-9a-f]{64}$ ]]; then
		rm -f "$temp_binary"
		print_error "Checksum manifest contains an invalid SHA-256 value"
		exit 1
	fi
	local actual_checksum
	actual_checksum=$(calculate_sha256 "$temp_binary") || {
		rm -f "$temp_binary"
		exit 1
	}
	actual_checksum=$(printf '%s' "$actual_checksum" | tr '[:upper:]' '[:lower:]')
	if [[ "$actual_checksum" != "$expected_checksum" ]]; then
		rm -f "$temp_binary"
		print_error "Checksum verification failed for $asset_name"
		exit 1
	fi

	chmod 0755 "$temp_binary"
	if ! verify_binary "$temp_binary" "$version"; then
        print_error "Binary validation failed"
		rm -f "$temp_binary"
        exit 1
    fi

	local binary_path="${install_dir}/${binary_name}"
	local backup_path="${binary_path}.bak.$$"
	if [[ -e "$binary_path" ]]; then
		mv "$binary_path" "$backup_path"
	fi
	if ! mv "$temp_binary" "$binary_path"; then
		[[ -e "$backup_path" ]] && mv "$backup_path" "$binary_path"
		print_error "Failed to install govman binary; previous binary was restored"
		exit 1
	fi
	rm -f "$backup_path"
	print_success "Downloaded and verified govman binary at ${binary_path}"
}
 # Add to PATH and initialize shell configuration
add_to_path() {
    local install_dir="$1"
    local govman_binary="${install_dir}/govman"
	if is_windows; then
		govman_binary="${govman_binary}.exe"
	fi
     # Ensure the govman binary is executable
    if [ ! -x "$govman_binary" ]; then
        print_error "govman binary not found or not executable at $govman_binary"
        exit 1
    fi
     print_step "Configuring shell environment..."
     # Run `govman init` and capture its output
    # The `init` command will automatically detect the shell and provide setup instructions
    # We use `--force` to ensure it overwrites any existing configuration
    local init_output shell_info init_shell
    shell_info=$(detect_shell_config)
    init_shell=${shell_info%%:*}
    case "$init_shell" in
        bash|zsh|fish)
            if init_output=$("$govman_binary" init --force --shell "$init_shell" 2>&1); then
                :
            else
                print_error "Shell configuration failed; the verified binary remains installed at $govman_binary"
                printf '%s\n' "$init_output" >&2
                print_warning "Run '$govman_binary init --force --shell $init_shell' after correcting the shell config error."
                return 1
            fi
            ;;
        *)
            if init_output=$("$govman_binary" init --force 2>&1); then
                :
            else
                print_error "Shell configuration failed; the verified binary remains installed at $govman_binary"
                printf '%s\n' "$init_output" >&2
                print_warning "Run '$govman_binary init --force' after correcting the shell config error."
                return 1
            fi
            ;;
    esac

    if [[ -n "$init_output" ]]; then
        print_success "Shell configuration completed successfully"
        [[ "$QUIET_MODE" == "true" ]] || printf '%s\n' "$init_output"
    else
        print_success "Shell configuration completed successfully"
    fi
}
 # Show system information
show_system_info() {
    local platform="$1"
    local version="$2"
    local install_dir="$3"
    [[ "$QUIET_MODE" == "true" ]] && return
         print_separator "┄"
    echo -e "${BOLD}${WHITE}System Information:${NC}"
    print_separator "┄"
    local os arch os_capitalized
    os=$(echo "$platform" | cut -d'/' -f1)
    arch=$(echo "$platform" | cut -d'/' -f2)
    case "$os" in
        linux) os_capitalized="Linux" ;;
        darwin) os_capitalized="Darwin" ;;
        windows) os_capitalized="Windows" ;;
        *) os_capitalized="$os" ;;
    esac
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
    [[ "$QUIET_MODE" == "true" ]] && return
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
    local shell_config binary_path
    local binary_found=false
    local config_found=false
    local command_found=false
         print_step "Checking for existing installation..."
         # Check binary directory
    binary_path="$install_dir/govman"
    is_windows && binary_path="${binary_path}.exe"
    if [[ -f "$binary_path" ]]; then
        binary_found=true
    fi
         # Check shell configurations
    get_shell_configs
    for shell_config in "${SHELL_CONFIGS[@]}"; do
        if [[ -f "$shell_config" ]] && grep -Fxq "# GOVMAN - Go Version Manager" "$shell_config" 2>/dev/null; then
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
            echo -e "${GREEN} ${CHECKMARK}${NC} Binary found: ${BOLD}$binary_path${NC}"
        fi
                 if [[ "$config_found" == true ]]; then
            echo -e "${GREEN} ${CHECKMARK}${NC} Shell configuration: ${BOLD}Found in PATH${NC}"
        fi
                 if [[ "$command_found" == true ]]; then
            local version
            version=$(govman --version 2>/dev/null | head -1 || echo "unknown")
            echo -e "${GREEN} ${CHECKMARK}${NC} Command available: ${BOLD}govman${NC} ${DIM}($version)${NC}"
        fi
                 if [[ -d "$govman_dir" ]]; then
            local dir_size
            dir_size=$(du -sh "$govman_dir" 2>/dev/null | cut -f1 || echo "unknown")
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
    if ! add_to_path "$install_dir"; then
        print_error "Installation is incomplete because shell initialization failed"
        return 1
    fi
    echo
         # Verify installation
    print_step "Verifying installation..."
    local installed_binary="$install_dir/govman"
    is_windows && installed_binary="${installed_binary}.exe"
    if "$installed_binary" --version >/dev/null 2>&1; then
        local installed_version
        installed_version=$("$installed_binary" --version 2>/dev/null | head -1 || echo "unknown")
        print_success "Installation verified: ${BOLD}$installed_version${NC}"
                 # Dynamic restart instruction
        local restart_instruction
        restart_instruction=$(get_restart_instruction)
                 show_completion "$version" "$restart_instruction"
    else
        print_error "Installation verification failed for $installed_binary"
        return 1
    fi
}
 # Trap to ensure clean exit
trap 'echo -e "\n${RED}Installation interrupted. Partial installation may have occurred.${NC}"; exit 1' INT TERM
 # Run main function
main "$@"
