# Release Notes

Official release notes and changelog for **govman**.

## Latest Release

### v1.0.0 (2025-11-01)

🎉 **First Release** - Initial stable release of govman - Go Version Manager.

#### Features

✨ **Core Functionality**
- Install and manage multiple Go versions
- Fast version switching with symlink management
- Project-specific versions via `.govman-version` files
- Automatic version switching on directory change
- Support for all Go versions (stable, beta, RC)

🐚 **Shell Integration**
- Bash, Zsh, Fish, PowerShell support
- Automatic PATH management
- Directory change hooks for auto-switching
- Non-intrusive configuration

⚡ **Performance**
- Parallel downloads with resume capability
- Intelligent caching system
- SHA-256 checksum verification
- Progress bars and status indicators

🌐 **Cross-Platform**
- Windows (AMD64, ARM64)
- macOS (Intel, Apple Silicon)
- Linux (AMD64, ARM64)
- No admin/sudo required

🔧 **Configuration**
- YAML-based configuration
- Mirror support for restricted networks
- Customizable download settings
- Flexible installation paths

#### Commands

- `govman init` - Initialize shell integration
- `govman install` - Install Go versions
- `govman uninstall` - Remove Go versions
- `govman use` - Switch Go versions
- `govman current` - Show active version
- `govman list` - List installed/available versions
- `govman info` - Show version details
- `govman clean` - Clean download cache
- `govman refresh` - Re-evaluate directory context
- `govman selfupdate` - Update govman itself

#### Installation Methods

- Quick install scripts (Bash, PowerShell, CMD)
- Manual binary download
- Build from source
- GitHub Releases

---

## Version History

### v1.0.0 (2025-11-01)

First public release - see features above.

---

## Deprecation Notices

### v1.0.0

- **None** - This is the first release

---

## Known Issues

### v1.0.0

1. **Windows Command Prompt**
   - Auto-switching not supported
   - Manual version switching required
   - **Workaround**: Use PowerShell instead

2. **Large Downloads on Slow Networks**
   - May timeout on very slow connections
   - **Workaround**: Increase timeout in config:
     ```yaml
     download:
       timeout: 1800s
     ```

3. **SELinux on Some Linux Distributions**
   - May require context adjustment
   - **Workaround**: `chcon -t bin_t ~/.govman/bin/govman`

---

## Security Updates

### v1.0.0

- ✅ SHA-256 checksum verification for all downloads
- ✅ Path traversal protection in archive extraction
- ✅ Input validation throughout
- ✅ No elevated privileges required
- ✅ Secure defaults in configuration

---

## Performance Benchmarks

### v1.0.0

Average performance on modern hardware:

- **Install time**: 30-60 seconds (depends on download speed)
- **Switch time**: < 1 second
- **Startup overhead**: < 10ms (shell integration)
- **Cache lookup**: < 5ms
- **Version resolution**: < 100ms

---

## Contributors

Thank you to all contributors who made govman possible!

### Core Team

- [justjundana](https://github.com/justjundana) - Creator & Maintainer

### Community Contributors

Want to contribute? Check out our [Developer Guide](getting-started.md)!

---

## Support

- 📖 [Documentation](https://github.com/justjundana/govman)
- 💬 [Discussions](https://github.com/justjundana/govman/discussions)
- 🐛 [Issue Tracker](https://github.com/justjundana/govman/issues)
- 📧 Contact: [GitHub Issues](https://github.com/justjundana/govman/issues/new)

---

## License

govman is released under the MIT License. See [LICENSE](../LICENSE.md) for details.

---

Stay updated: Star ⭐ the [GitHub repository](https://github.com/justjundana/govman) to receive notifications!
