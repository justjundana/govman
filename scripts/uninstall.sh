#!/usr/bin/env bash
set -euo pipefail

readonly GOVMAN_MARKER_START="# GOVMAN - Go Version Manager"
readonly GOVMAN_MARKER_END="# END GOVMAN"

SHELL_CONFIGS=()
CONFIGS_TO_CLEAN=()
MARKER_STATE="absent"

log_info() {
    printf 'INFO: %s\n' "$*"
}

log_success() {
    printf 'SUCCESS: %s\n' "$*"
}

log_error() {
    printf 'ERROR: %s\n' "$*" >&2
}

get_shell_configs() {
    SHELL_CONFIGS=(
        "$HOME/.bashrc"
        "$HOME/.bash_profile"
        "$HOME/.profile"
        "$HOME/.zshrc"
        "$HOME/.config/fish/config.fish"
    )
}

has_govman_marker() {
    local config_path="$1"
    [[ -f "$config_path" ]] || return 1
    grep -Fxq "$GOVMAN_MARKER_START" "$config_path" 2>/dev/null \
        || grep -Fxq "$GOVMAN_MARKER_END" "$config_path" 2>/dev/null
}

inspect_marker_block() {
    local config_path="$1"
    local marker_data
    local start_count end_count start_line end_line

    MARKER_STATE="absent"
    [[ -e "$config_path" || -L "$config_path" ]] || return 0

    if [[ -L "$config_path" ]]; then
        if has_govman_marker "$config_path"; then
            log_error "Refusing to modify symlinked shell configuration: $config_path"
            return 1
        fi
        return 0
    fi
    if [[ ! -f "$config_path" ]]; then
        return 0
    fi

    marker_data=$(awk -v start="$GOVMAN_MARKER_START" -v end="$GOVMAN_MARKER_END" '
        $0 == start { start_count++; if (start_line == 0) start_line = NR }
        $0 == end { end_count++; if (end_line == 0) end_line = NR }
        END { printf "%d %d %d %d\n", start_count + 0, end_count + 0, start_line + 0, end_line + 0 }
    ' "$config_path")
    read -r start_count end_count start_line end_line <<<"$marker_data"

    if (( start_count == 0 && end_count == 0 )); then
        return 0
    fi
    if (( start_count != 1 || end_count != 1 || end_line <= start_line )); then
        log_error "Malformed or duplicate govman marker block: $config_path"
        return 1
    fi

    MARKER_STATE="valid"
}

file_mode() {
    local config_path="$1"
    if stat -f '%Lp' "$config_path" >/dev/null 2>&1; then
        stat -f '%Lp' "$config_path"
    else
        stat -c '%a' "$config_path"
    fi
}

remove_govman_block() {
    local config_path="$1"
    local config_dir config_name timestamp backup_path temp_path mode

    inspect_marker_block "$config_path" || return 1
    [[ "$MARKER_STATE" == "valid" ]] || return 0

    config_dir=$(dirname "$config_path")
    config_name=$(basename "$config_path")
    timestamp=$(date -u '+%Y%m%dT%H%M%SZ')
    backup_path="${config_path}.govman-backup.${timestamp}.$$"
    temp_path=$(mktemp "${config_dir}/.${config_name}.govman.XXXXXX") || {
        log_error "Unable to create a temporary file beside $config_path"
        return 1
    }

    if ! cp -p "$config_path" "$backup_path"; then
        rm -f "$temp_path"
        log_error "Unable to back up shell configuration: $config_path"
        return 1
    fi

    if ! awk -v start="$GOVMAN_MARKER_START" -v end="$GOVMAN_MARKER_END" '
        $0 == start { removing = 1; next }
        removing && $0 == end { removing = 0; next }
        !removing { print }
        END { if (removing) exit 2 }
    ' "$config_path" >"$temp_path"; then
        rm -f "$temp_path"
        log_error "Unable to remove govman block from $config_path; backup: $backup_path"
        return 1
    fi

    mode=$(file_mode "$config_path") || {
        rm -f "$temp_path"
        log_error "Unable to read permissions for $config_path; backup: $backup_path"
        return 1
    }
    if ! chmod "$mode" "$temp_path"; then
        rm -f "$temp_path"
        log_error "Unable to preserve permissions for $config_path; backup: $backup_path"
        return 1
    fi
    if [[ -L "$config_path" ]]; then
        rm -f "$temp_path"
        log_error "Shell configuration became a symlink during update: $config_path"
        return 1
    fi
    if ! mv -f "$temp_path" "$config_path"; then
        rm -f "$temp_path"
        log_error "Unable to replace shell configuration: $config_path; backup: $backup_path"
        return 1
    fi

    log_success "Removed govman integration from $config_path"
    log_info "Backup saved to $backup_path"
}

preflight_shell_configs() {
    local config_path

    CONFIGS_TO_CLEAN=()
    get_shell_configs
    for config_path in "${SHELL_CONFIGS[@]}"; do
        inspect_marker_block "$config_path" || return 1
        if [[ "$MARKER_STATE" == "valid" ]]; then
            CONFIGS_TO_CLEAN+=("$config_path")
        fi
    done
}

remove_from_path() {
    local config_path
    local modified=0

    preflight_shell_configs || return 1
    for config_path in "${CONFIGS_TO_CLEAN[@]}"; do
        log_info "Cleaning shell configuration: $config_path"
        remove_govman_block "$config_path" || return 1
        modified=$((modified + 1))
    done

    if (( modified == 0 )); then
        log_info "No complete govman shell integration blocks found"
    else
        log_success "Cleaned $modified shell configuration(s)"
    fi
}

remove_binary() {
    local install_dir="$HOME/.govman/bin"
    if [[ -e "$install_dir" || -L "$install_dir" ]]; then
        log_info "Removing binary directory: $install_dir"
        rm -rf "$install_dir"
        log_success "Removed govman binaries"
    else
        log_info "Binary directory not found: $install_dir"
    fi
}

remove_all_data() {
    local govman_dir="$HOME/.govman"
    if [[ -e "$govman_dir" || -L "$govman_dir" ]]; then
        log_info "Removing data directory: $govman_dir"
        rm -rf "$govman_dir"
        log_success "Removed all govman data"
    else
        log_info "Data directory not found: $govman_dir"
    fi
}

installation_exists() {
    local config_path
    [[ -e "$HOME/.govman" || -L "$HOME/.govman" ]] && return 0
    get_shell_configs
    for config_path in "${SHELL_CONFIGS[@]}"; do
        has_govman_marker "$config_path" && return 0
    done
    return 1
}

read_answer() {
    local prompt="$1"
    local answer=""
    printf '%s' "$prompt" >&2
    IFS= read -r answer || true
    printf '%s' "$answer"
}

main() {
    local choice confirm

    if [[ -t 1 && "${TERM:-}" != "dumb" ]]; then
        clear 2>/dev/null || true
    fi
    printf 'govman uninstaller\n\n'

    if ! installation_exists; then
        log_info "No govman installation or shell integration was found"
        return 0
    fi

    printf '1) Minimal removal (keep downloaded Go versions)\n'
    printf '2) Complete removal (delete all govman data)\n'
    printf '3) Cancel\n'
    choice=$(read_answer 'Choose an option (1/2/3): ')

    case "$choice" in
        1)
            confirm=$(read_answer 'Proceed with minimal removal? (y/N): ')
            [[ "$confirm" =~ ^[Yy]$ ]] || { log_info "Uninstallation cancelled"; return 0; }
            remove_from_path || return 1
            remove_binary
            log_success "Minimal uninstallation completed; downloaded Go versions were retained"
            ;;
        2)
            confirm=$(read_answer "Type 'DELETE' to remove all govman data: ")
            [[ "$confirm" == "DELETE" ]] || { log_info "Uninstallation cancelled"; return 0; }
            remove_from_path || return 1
            remove_all_data
            log_success "Complete uninstallation completed"
            ;;
        *)
            log_info "Uninstallation cancelled"
            ;;
    esac
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    trap 'printf "ERROR: uninstallation interrupted; some state may have been retained\n" >&2; exit 1' INT TERM
    main "$@"
fi
