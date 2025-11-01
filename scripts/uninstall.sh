#!/usr/bin/env bash
 # govman uninstallation script
# This script removes govman from $HOME/.govman/bin and removes it from PATH
 set -e
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
TRASH="🗑"
WARNING="⚠"
QUESTION="❓"
STOP="🛑"
CLEAN="🧹"
SHIELD="🛡"
INFO="ℹ"
 # Terminal width detection
TERM_WIDTH=$(tput cols 2>/dev/null || echo 80)
 # Print separator line
print_separator() {
    local char="${1:--}"
    printf "%*s\n" "$TERM_WIDTH" | tr ' ' "$char"
}
 # Print fancy header
print_header() {
    clear
    print_separator "═"
    echo
    echo
    echo '    ██╗   ██╗███╗   ██╗██╗███╗   ██╗███████╗████████╗ █████╗ ██╗     ██╗'
    echo '    ██║   ██║████╗  ██║██║████╗  ██║██╔════╝╚══██╔══╝██╔══██╗██║     ██║'
    echo '    ██║   ██║██╔██╗ ██║██║██╔██╗ ██║███████╗   ██║   ███████║██║     ██║'
    echo '    ██║   ██║██║╚██╗██║██║██║╚██╗██║╚════██║   ██║   ██╔══██║██║     ██║'
    echo '    ╚██████╔╝██║ ╚████║██║██║ ╚████║███████║   ██║   ██║  ██║███████╗███████╗'
    echo '     ╚═════╝ ╚═╝  ╚═══╝╚═╝╚═╝  ╚═══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝'
    echo
    echo
    echo -e "${BOLD}${WHITE}                        Go Version Manager Uninstaller${NC}"
    echo -e "${DIM}${GRAY}                  Safe and complete uninstallation process${NC}"
    echo
    print_separator "═"
    echo
}
 # Print functions with icons and styling
print_info() {
    echo -e "${BLUE}${BOLD} ${INFO}  INFO${NC} ${GRAY}│${NC} $1"
}
 print_success() {
    echo -e "${GREEN}${BOLD} ${CHECKMARK}  SUCCESS${NC} ${GRAY}│${NC} $1"
}
 print_warning() {
    echo -e "${YELLOW}${BOLD} ${WARNING}  WARNING${NC} ${GRAY}│${NC} $1"
}
 print_error() {
    echo -e "${RED}${BOLD} ${CROSSMARK}  ERROR${NC} ${GRAY}│${NC} $1"
}
 print_step() {
    echo -e "${PURPLE}${BOLD} ${ARROW}  STEP${NC} ${GRAY}│${NC} $1"
}
 print_clean() {
    echo -e "${CYAN}${BOLD} ${CLEAN}  CLEANING${NC} ${GRAY}│${NC} $1"
}
 print_question() {
    echo -e "${YELLOW}${BOLD} ${QUESTION}  QUESTION${NC} ${GRAY}│${NC} $1"
}
 # User input function
get_user_input() {
    local prompt="$1"
    local response=""
     # Always read from /dev/tty when available
    if [[ -e /dev/tty ]]; then
        read -r -p "$(echo -e "$prompt")" response </dev/tty
    else
        # Fallback to standard input
        read -r -p "$(echo -e "$prompt")" response
    fi
     echo "$response"
}
 # Get shell configuration files
get_shell_configs() {
    local configs=("$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.zshrc")
    [[ -f "$HOME/.config/fish/config.fish" ]] && configs+=("$HOME/.config/fish/config.fish")
    printf '%s ' "${configs[@]}"
}
 # Check if govman is installed
check_govman_installation() {
    local install_dir="$HOME/.govman/bin"
    local govman_dir="$HOME/.govman"
    local shell_configs_str
    shell_configs_str=$(get_shell_configs)
    local shell_configs=($shell_configs_str)
    local binary_found=false
    local config_found=false
    local data_found=false
         print_step "Checking govman installation..."
         # Check binary directory
    if [[ -d "$install_dir" ]]; then
        binary_found=true
    fi
         # Check shell configurations
    for shell_config in "${shell_configs[@]}"; do
        if [[ -f "$shell_config" ]] && grep -q "# GOVMAN - Go Version Manager" "$shell_config" 2>/dev/null; then
            config_found=true
            break
        fi
    done
         # Check data directory
    if [[ -d "$govman_dir" ]]; then
        data_found=true
    fi
         # Check if govman command is available in PATH
    local command_found=false
    if command -v govman >/dev/null 2>&1; then
        command_found=true
    fi
         echo
    print_separator "┄"
    echo -e "${BOLD}${WHITE}Installation Status:${NC}"
    print_separator "┄"
         if [[ "$binary_found" == true ]]; then
        echo -e "${GREEN} ${CHECKMARK}${NC} Binary directory: ${BOLD}$install_dir${NC}"
    else
        echo -e "${GRAY} ${CROSSMARK}${NC} Binary directory: ${DIM}$install_dir (not found)${NC}"
    fi
         if [[ "$config_found" == true ]]; then
        echo -e "${GREEN} ${CHECKMARK}${NC} Shell configuration: ${BOLD}Found in PATH${NC}"
    else
        echo -e "${GRAY} ${CROSSMARK}${NC} Shell configuration: ${DIM}No govman configuration found${NC}"
    fi
         if [[ "$command_found" == true ]]; then
        local version=$(govman --version 2>/dev/null | head -1 || echo "unknown")
        echo -e "${GREEN} ${CHECKMARK}${NC} Command available: ${BOLD}govman${NC} ${DIM}($version)${NC}"
    else
        echo -e "${GRAY} ${CROSSMARK}${NC} Command available: ${DIM}govman (not in PATH)${NC}"
    fi
         if [[ "$data_found" == true ]]; then
        local dir_size=$(du -sh "$govman_dir" 2>/dev/null | cut -f1 || echo "unknown")
        echo -e "${BLUE} ${INFO}${NC} Data directory: ${BOLD}$govman_dir${NC} ${DIM}($dir_size)${NC}"
    else
        echo -e "${GRAY} ${CROSSMARK}${NC} Data directory: ${DIM}$govman_dir (not found)${NC}"
    fi
         print_separator "┄"
    echo
         # Return status: 0 if something to uninstall, 1 if nothing found
    if [[ "$binary_found" == true || "$config_found" == true || "$data_found" == true ]]; then
        return 0
    else
        return 1
    fi
}
 # Show what will be removed based on option
show_removal_preview() {
    local option="$1"
     echo -e "${BOLD}${WHITE}Removal Preview:${NC}"
    print_separator "┄"
     local install_dir="$HOME/.govman/bin"
    local govman_dir="$HOME/.govman"
    local shell_configs_str
    shell_configs_str=$(get_shell_configs)
    local shell_configs=($shell_configs_str)
         # Check binary
    if [[ -d "$install_dir" ]]; then
        echo -e "${RED} ${TRASH}${NC} Binary directory: ${BOLD}$install_dir${NC}"
    else
        echo -e "${GRAY} ${CROSSMARK}${NC} Binary directory: ${DIM}$install_dir (not found)${NC}"
    fi
         # Check shell configurations
    local config_found=false
    for shell_config in "${shell_configs[@]}"; do
        if [[ -f "$shell_config" ]] && grep -q "# GOVMAN - Go Version Manager" "$shell_config" 2>/dev/null; then
            echo -e "${RED} ${TRASH}${NC} Shell config: ${BOLD}$shell_config${NC}"
            config_found=true
        fi
    done
         if [[ "$config_found" == false ]]; then
        echo -e "${GRAY} ${CROSSMARK}${NC} Shell configs: ${DIM}No govman configuration found${NC}"
    fi
         # Show data directory based on option
    if [[ -d "$govman_dir" ]]; then
        local dir_size=$(du -sh "$govman_dir" 2>/dev/null | cut -f1 || echo "unknown")
        if [[ "$option" == "complete" ]]; then
            echo -e "${RED} ${TRASH}${NC} Data directory: ${BOLD}$govman_dir${NC} ${DIM}($dir_size)${NC}"
        else
            echo -e "${GREEN} ${SHIELD}${NC} Data directory: ${BOLD}$govman_dir${NC} ${DIM}($dir_size - will be kept)${NC}"
        fi
    else
        echo -e "${GRAY} ${CROSSMARK}${NC} Data directory: ${DIM}$govman_dir (not found)${NC}"
    fi
         print_separator "┄"
    echo
}
 # Animated loading for removal process
show_removal_progress() {
    local item="$1"
    local delay=0.1
    local spinstr='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    local temp
         echo -n "   ${DIM}Removing $item... ${NC}"
    for i in {1..10}; do
        temp=${spinstr#?}
        printf "\r   ${DIM}Removing $item... ${CYAN}%c${NC} " "$spinstr"
        spinstr=$temp${spinstr%"$temp"}
        sleep $delay
    done
    printf "\r   ${GREEN}${CHECKMARK}${NC} Removed $item successfully.      \n"
}
 # Remove binary with feedback
remove_binary() {
    local install_dir="$HOME/.govman/bin"
         print_step "Removing govman binary..."
         if [[ -d "$install_dir" ]]; then
        show_removal_progress "binary directory"
        rm -rf "$install_dir"
        print_success "Removed govman binary from $install_dir"
    else
        print_warning "govman binary directory not found at $install_dir"
    fi
}
 # Remove from PATH with feedback
remove_from_path() {
    local shell_configs_str
    shell_configs_str=$(get_shell_configs)
    local shell_configs=($shell_configs_str)
    local configs_modified=0
         print_step "Cleaning shell configurations..."
         for shell_config in "${shell_configs[@]}"; do
        if [[ -f "$shell_config" ]]; then
            # Check if govman is configured in this config
            if grep -q "# GOVMAN - Go Version Manager" "$shell_config" 2>/dev/null; then
                show_removal_progress "$(basename "$shell_config") configuration"
                 # More portable sed operation
                sed '/# GOVMAN - Go Version Manager/,/# END GOVMAN/d' "$shell_config" > "${shell_config}.tmp"
                mv "${shell_config}.tmp" "$shell_config"
                 # Clean up extra blank lines that might be left
                awk 'NF || prev_blank {print} {prev_blank = !NF}' "$shell_config" > "${shell_config}.tmp" && mv "${shell_config}.tmp" "$shell_config"
                 print_success "Cleaned PATH configuration in $(basename "$shell_config")"
                ((configs_modified++))
            fi
        fi
    done
         if [[ $configs_modified -eq 0 ]]; then
        print_info "No shell configurations found with govman setup"
    else
        print_success "Cleaned $configs_modified shell configuration(s)"
    fi
}
 # Remove entire govman directory with feedback
remove_govman_dir() {
    local govman_dir="$HOME/.govman"
         print_step "Removing govman data directory..."
         if [[ -d "$govman_dir" ]]; then
        # Show what's being removed
        local dir_size=$(du -sh "$govman_dir" 2>/dev/null | cut -f1 || echo "unknown size")
        print_info "Removing directory: $govman_dir ($dir_size)"
                 show_removal_progress "data directory"
        rm -rf "$govman_dir"
        print_success "Removed govman data directory"
    else
        print_warning "govman directory not found at $govman_dir"
    fi
}
 # Show uninstall options
show_uninstall_options() {
    print_separator "═"
    echo -e "${BOLD}${WHITE} ${QUESTION}  UNINSTALLATION OPTIONS${NC}"
    print_separator "═"
    echo
    echo -e "${CYAN}${BOLD}1)${NC} ${WHITE}Minimal Removal${NC} ${DIM}(Recommended)${NC}"
    echo "   • Remove govman binary and executable"
    echo "   • Clean shell PATH configurations"  
    echo -e "   • ${GREEN}Keep${NC} downloaded Go versions for future use"
    echo
    echo -e "${RED}${BOLD}2)${NC} ${WHITE}Complete Removal${NC} ${DIM}(Permanent)${NC}"
    echo "   • Remove govman binary and executable"
    echo "   • Clean shell PATH configurations"
    echo -e "   • ${RED}Delete${NC} all downloaded Go versions and data"
    echo -e "   • ${RED}Delete${NC} entire ~/.govman directory"
    echo
    echo -e "${GRAY}${BOLD}3)${NC} ${WHITE}Cancel${NC}"
    echo "   • Exit without making any changes"
    echo
    print_separator "┄"
}
 # Show completion message
show_completion() {
    local complete_removal="$1"
         echo
    print_separator "═"
    echo
    if [[ "$complete_removal" == "true" ]]; then
        echo -e "${GREEN}${BOLD} ${CHECKMARK}  COMPLETE UNINSTALLATION SUCCESSFUL!${NC}"
        echo
        print_separator "┄"
        echo -e "${BOLD}${WHITE}What was removed:${NC}"
        echo " • govman binary and executable"
        echo " • Shell PATH configurations"
        echo " • All downloaded Go versions"
        echo " • Complete .govman directory"
    else
        echo -e "${GREEN}${BOLD} ${CHECKMARK}  MINIMAL UNINSTALLATION COMPLETE!${NC}"
        echo
        print_separator "┄"
        echo -e "${BOLD}${WHITE}What was removed:${NC}"
        echo " • govman binary and executable"
        echo " • Shell PATH configurations"
        echo
        echo -e "${BOLD}${WHITE}What was kept:${NC}"
        echo " • Downloaded Go versions in ~/.govman"
    fi
    print_separator "┄"
    echo -e "${BOLD}${WHITE}Final Steps:${NC}"
    echo " 1. Restart your terminal to complete the process"
    echo " 2. Verify with 'govman --version' (should show 'command not found')"
    if [[ "$complete_removal" != "true" ]]; then
        echo " 3. Manually remove '~/.govman' if you change your mind later"
    fi
    print_separator "┄"
    echo "Thank you for using govman!"
    print_separator "═"
    echo
}
 # Main uninstallation function
main() {
    # Show header
    print_header
         print_info "Starting govman uninstallation process..."
    echo
         # Check if govman is installed
    if ! check_govman_installation; then
        print_warning "govman does not appear to be installed on this system"
        echo
        print_separator "┄"
        echo -e "${BOLD}${WHITE}No govman installation found!${NC}"
        print_separator "┄"
        echo "It looks like govman is not installed or has already been removed."
        echo "Common reasons:"
        echo " • govman was never installed"
        echo " • govman was already uninstalled"
        echo " • govman was installed in a different location"
        echo " • Installation was incomplete or corrupted"
        print_separator "┄"
        echo
        local response
        response=$(get_user_input "Do you want to clean any remaining traces? ${DIM}(y/N):${NC} ")
                 if [[ ! "$response" =~ ^[Yy]$ ]]; then
            echo
            print_info "Exiting without making changes"
            print_separator "═"
            echo -e "${DIM}${GRAY}No changes were made to your system.${NC}"
            print_separator "═"
            echo
            exit 0
        fi
                 echo
        print_info "Proceeding with cleanup of any remaining traces..."
        echo
    else
        print_success "govman installation detected"
        echo
    fi
         # Show uninstall options
    show_uninstall_options
         # Get user choice
    local response
    response=$(get_user_input "Choose an option ${DIM}(1/2/3):${NC} ")
         echo
         case "$response" in
        1)
            print_info "Proceeding with minimal removal..."
            echo
            show_removal_preview "minimal"
                         # Final confirmation for minimal removal
            print_separator "┄"
            echo -e "${YELLOW}${BOLD} ${STOP}  FINAL CONFIRMATION${NC}"
            print_separator "┄"
            local confirm
            confirm=$(get_user_input "Proceed with minimal removal? ${DIM}(y/N):${NC} ")
                         if [[ "$confirm" =~ ^[Yy]$ ]]; then
                echo
                remove_binary
                echo
                remove_from_path
                echo
                show_completion "false"
            else
                echo
                print_info "Uninstallation cancelled by user"
                print_separator "═"
                echo -e "${DIM}${GRAY}No changes were made to your system.${NC}"
                print_separator "═"
                echo
            fi
            ;;
                     2)
            print_info "Proceeding with complete removal..."
            echo
            show_removal_preview "complete"
                         # Final confirmation for complete removal
            print_separator "┄"
            echo -e "${RED}${BOLD} ${STOP}  DANGER: COMPLETE REMOVAL${NC}"
            print_separator "┄"
            echo -e "${RED}This will permanently delete ALL govman data and cannot be undone!${NC}"
            print_separator "┄"
            local confirm
            confirm=$(get_user_input "Type 'DELETE' to confirm complete removal: ")
                         if [[ "$confirm" == "DELETE" ]]; then
                echo
                remove_binary
                echo
                remove_from_path
                echo
                remove_govman_dir
                echo
                show_completion "true"
            else
                echo
                print_info "Complete removal cancelled - confirmation text did not match"
                print_separator "═"
                echo -e "${DIM}${GRAY}No changes were made to your system.${NC}"
                print_separator "═"
                echo
            fi
            ;;
                     3|*)
            echo
            print_info "Uninstallation cancelled by user"
            print_separator "═"
            echo -e "${DIM}${GRAY}No changes were made to your system.${NC}"
            print_separator "═"
            echo
            ;;
    esac
}
 # Trap to ensure clean exit
trap 'echo -e "\n${RED}Uninstallation interrupted. Incomplete removal may have occurred.${NC}"; exit 1' INT TERM
 # Run main function
main "$@"