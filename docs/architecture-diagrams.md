# Diagrams

Visual representations of govman's architecture and workflows.

## System Overview

```mermaid
graph TB
    User[User] --> CLI[CLI Commands]
    CLI --> Manager[Manager]
    Manager --> Config[Config]
    Manager --> Downloader[Downloader]
    Manager --> Shell[Shell Integration]
    Manager --> Golang[Go Releases API]
    
    Downloader --> HTTP[HTTP Client]
    Downloader --> Progress[Progress Bar]
    Golang --> GoDevAPI[go.dev API]
    
    HTTP --> Cache[Download Cache]
    HTTP --> Install[Install Directory]
    
    Config --> YAML[config.yaml]
    Shell --> ShellRC[Shell Config Files]
    
    style User fill:#e1f5ff
    style Manager fill:#fff3cd
    style Downloader fill:#d4edda
    style Golang fill:#d4edda
```

## Installation Workflow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Manager
    participant Golang
    participant Downloader
    participant Filesystem
    
    User->>CLI: govman install 1.25.1
    CLI->>Manager: Install("1.25.1")
    Manager->>Golang: GetDownloadURL("1.25.1")
    Golang->>Golang: Fetch from go.dev API
    Golang-->>Manager: URL + Checksum
    Manager->>Downloader: Download(url, installDir)
    Downloader->>Downloader: Check cache
    Downloader->>Filesystem: Download to cache
    Downloader->>Downloader: Verify checksum
    Downloader->>Filesystem: Extract to install dir
    Downloader-->>Manager: Success
    Manager-->>CLI: Success
    CLI-->>User: ✓ Installed Go 1.25.1
```

## Version Switching Workflow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Manager
    participant Config
    participant Symlink
    participant Shell
    
    User->>CLI: govman use 1.25.1 --default
    CLI->>Manager: Use("1.25.1", default=true)
    Manager->>Manager: Verify version installed
    Manager->>Config: Set DefaultVersion
    Config->>Config: Save to config.yaml
    Manager->>Symlink: Create symlink
    Symlink->>Symlink: ~/.govman/bin/go → versions/go1.25.1/bin/go
    Manager->>Shell: Generate PATH command
    Shell-->>Manager: export PATH=...
    Manager-->>CLI: PATH command
    CLI-->>User: export PATH=... (for shell wrapper)
    User->>User: Shell wrapper evals PATH
```

## Auto-Switch Workflow

```mermaid
flowchart TD
    Start[User: cd /project] --> Hook{Shell Hook Triggered}
    Hook -->|Yes| CheckConfig{Auto-switch enabled?}
    CheckConfig -->|No| End[No Action]
    CheckConfig -->|Yes| CheckFile{.govman-goversion exists?}
    CheckFile -->|No| UseDefault[Use default version]
    CheckFile -->|Yes| ReadFile[Read required version]
    ReadFile --> CheckCurrent{Current == Required?}
    CheckCurrent -->|Yes| End
    CheckCurrent -->|No| Switch[govman use required-version]
    Switch --> UpdatePATH[Update PATH]
    UpdatePATH --> End
    UseDefault --> End
```

## Component Dependencies

```mermaid
graph TD
    CLI[internal/cli] --> Manager[internal/manager]
    CLI --> Logger[internal/logger]
    CLI --> Shell[internal/shell]
    
    Manager --> Config[internal/config]
    Manager --> Downloader[internal/downloader]
    Manager --> Golang[internal/golang]
    Manager --> Logger
    Manager --> Shell
    Manager --> Symlink[internal/symlink]
    
    Downloader --> Progress[internal/progress]
    Downloader --> Logger
    Downloader --> Golang
    Downloader --> Util[internal/util]
    
    Golang --> Logger
    
    Config --> Viper[github.com/spf13/viper]
    CLI --> Cobra[github.com/spf13/cobra]
    
    style CLI fill:#e1f5ff
    style Manager fill:#fff3cd
    style Config fill:#d4edda
    style Downloader fill:#d4edda
    style Golang fill:#d4edda
    style Logger fill:#f8d7da
    style Shell fill:#d4edda
```

## File System Organization

```mermaid
graph TB
    Home[~/.govman/] --> Bin[bin/]
    Home --> Versions[versions/]
    Home --> Cache[cache/]
    Home --> ConfigFile[config.yaml]
    
    Bin --> GoSymlink[go → versions/go1.25.1/bin/go]
    
    Versions --> V1[go1.25.1/]
    Versions --> V2[go1.24.0/]
    Versions --> V3[go1.23.5/]
    
    V1 --> V1Bin[bin/]
    V1 --> V1Pkg[pkg/]
    V1 --> V1Src[src/]
    
    Cache --> Archive1[go1.25.1.linux-amd64.tar.gz]
    Cache --> Archive2[go1.24.0.linux-amd64.tar.gz]
    
    style Home fill:#e1f5ff
    style Bin fill:#fff3cd
    style Versions fill:#d4edda
    style Cache fill:#f8d7da
```

## State Machine: Version Activation

```mermaid
stateDiagram-v2
    [*] --> NoVersion: Fresh install
    NoVersion --> SessionActive: govman use X
    NoVersion --> DefaultSet: govman use X --default
    NoVersion --> LocalSet: govman use X --local
    
    SessionActive --> SessionActive: govman use Y (in same session)
    SessionActive --> DefaultSet: govman use Y --default
    SessionActive --> LocalSet: govman use Y --local
    SessionActive --> NoVersion: Shell restart
    
    DefaultSet --> SessionActive: govman use Y (session-only)
    DefaultSet --> DefaultSet: govman use Y --default
    DefaultSet --> LocalSet: govman use Y --local (in project dir)
    DefaultSet --> DefaultSet: Shell restart (persists)
    
    LocalSet --> SessionActive: govman use Y (session-only)
    LocalSet --> DefaultSet: govman use Y --default
    LocalSet --> LocalSet: cd to different project dir
```

## Download Flow

```mermaid
flowchart TD
    Start[Request Install] --> Resolve[Resolve Version]
    Resolve --> GetURL[Get Download URL from API]
    GetURL --> CheckCache{File in cache?}
    CheckCache -->|Yes| VerifyCache{Cache valid?}
    CheckCache -->|No| Download[Download from go.dev]
    VerifyCache -->|Yes| UseCache[Use cached file]
    VerifyCache -->|No| Download
    Download --> SaveCache[Save to cache]
    SaveCache --> Checksum[Verify SHA-256]
    UseCache --> Checksum
    Checksum --> Extract[Extract archive]
    Extract --> SetPerms[Set permissions]
    SetPerms --> Success[Installation complete]
```

## Shell Integration Architecture

```mermaid
graph LR
    ShellRC[Shell Config File] --> WrapperFunc[govman wrapper function]
    ShellRC --> AutoSwitch[govman_auto_switch function]
    ShellRC --> Hook[Shell Hook]
    
    Hook -->|bash| PromptCmd[PROMPT_COMMAND]
    Hook -->|zsh| Chpwd[chpwd hook]
    Hook -->|fish| PwdEvent[PWD event]
    Hook -->|powershell| SetLocation[Set-Location override]
    
    PromptCmd --> AutoSwitch
    Chpwd --> AutoSwitch
    PwdEvent --> AutoSwitch
    SetLocation --> AutoSwitch
    
    AutoSwitch --> CheckFile[Check .govman-goversion]
    CheckFile --> WrapperFunc
    
    WrapperFunc --> Use[govman use]
    Use --> UpdatePATH[Update PATH in current shell]
```

## Error Handling Flow

```mermaid
flowchart TD
    Operation[Operation] --> Try{Try}
    Try -->|Success| Return[Return success]
    Try -->|Error| Wrap[Wrap error with context]
    Wrap --> CheckRetry{Retryable?}
    CheckRetry -->|Yes| Retry{Retries left?}
    CheckRetry -->|No| Log[Log error]
    Retry -->|Yes| Wait[Wait retry_delay]
    Retry -->|No| Log
    Wait --> Try
    Log --> Format[Format user message]
    Format --> Help[Add help suggestion]
    Help --> Display[Display to user]
    Display --> Exit[Exit with error code]
```

## Release and Self-Update

```mermaid
sequenceDiagram
    participant User
    participant govman
    participant GitHub
    participant Binary
    
    User->>govman: govman selfupdate
    govman->>GitHub: GET /repos/.../releases/latest
    GitHub-->>govman: Release metadata
    govman->>govman: Compare versions
    alt New version available
        govman->>GitHub: Download new binary
        GitHub-->>govman: Binary file
        govman->>Binary: Backup current binary
        govman->>Binary: Replace with new binary
        govman->>Binary: Set permissions
        govman->>Binary: Verify new version
        alt Verification success
            govman->>Binary: Remove backup
            govman-->>User: ✓ Updated to vX.X.X
        else Verification failed
            govman->>Binary: Restore backup
            govman-->>User: ✗ Update failed, rolled back
        end
    else Already latest
        govman-->>User: Already on latest version
    end
```

## Platform-Specific Binary Selection

```mermaid
flowchart TD
    Start[Download Request] --> DetectOS{Detect OS}
    DetectOS -->|Linux| DetectArchL{Detect Architecture}
    DetectOS -->|macOS| DetectArchM{Detect Architecture}
    DetectOS -->|Windows| DetectArchW{Detect Architecture}
    
    DetectArchL -->|amd64| LinuxAMD64[linux-amd64]
    DetectArchL -->|arm64| LinuxARM64[linux-arm64]
    
    DetectArchM -->|amd64| DarwinAMD64[darwin-amd64]
    DetectArchM -->|arm64| CheckVersion{Go version >= 1.16?}
    CheckVersion -->|Yes| DarwinARM64[darwin-arm64]
    CheckVersion -->|No| DarwinAMD64Rosetta[darwin-amd64 via Rosetta]
    
    DetectArchW -->|amd64| WindowsAMD64[windows-amd64]
    DetectArchW -->|arm64| WindowsARM64[windows-arm64]
    
    LinuxAMD64 --> Download[Download]
    LinuxARM64 --> Download
    DarwinAMD64 --> Download
    DarwinARM64 --> Download
    DarwinAMD64Rosetta --> Download
    WindowsAMD64 --> Download
    WindowsARM64 --> Download
```

## Configuration Loading Sequence

```mermaid
sequenceDiagram
    participant App
    participant Config
    participant Viper
    participant Filesystem
    
    App->>Config: Load()
    Config->>Filesystem: Check ~/.govman/config.yaml exists
    alt Config exists
        Config->>Viper: ReadInConfig()
        Viper->>Filesystem: Read YAML
        Filesystem-->>Viper: YAML content
        Viper->>Viper: Parse YAML
        Viper-->>Config: Parsed config
    else No config
        Config->>Config: setDefaults()
        Config->>Viper: Set default values
    end
    Config->>Config: expandPaths() - resolve ~
    Config->>Config: Validate paths
    Config->>Filesystem: createDirectories()
    Filesystem-->>Config: Directories created
    Config-->>App: Loaded config
```
