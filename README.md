# AI Coding Playground

Experimental repository for AI-assisted software development. A workspace to explore and showcase AI-driven coding across multiple projects using Claude Code, GPT, Cursor, and other AI tools.

## Projects

### 1. FFmpeg-Rust-Port ✅ COMPLETE

Pure Rust audio processing library with zero C dependencies - a complete rewrite of FFmpeg focusing on audio processing.

**Location**: `projects/ffmpeg-rust-port/`

**Status**:
- ✅ 7 phases complete
- ✅ 21 unit tests passing
- ✅ Full test coverage
- ✅ Rust 2024 edition

**Features**:
- **Filters**: Resample (44.1kHz→16kHz), Remix (stereo↔mono), Normalize (peak/loudness)
- **Processors**: Audio segmentation by duration
- **Encoders**: WAV (32-bit float PCM)
- **Decoders**: Symphonia framework (MP3, FLAC, WAV, AAC, OGG ready)
- **CLI**: Full command-line interface

**AI Tools Used**: Claude Code (Anthropic)

**Quick Commands**:
```bash
cd projects/ffmpeg-rust-port

# Build library
cargo build --lib

# Run tests (21 tests)
cargo test --lib

# Show project info
cargo run --release
```

**Key Stats**:
- ~2,500+ lines of production Rust code
- 100% safe Rust (no unsafe code)
- Zero compiler warnings
- Modular, trait-based architecture

---

## Supported AI Tools

This playground is designed to work seamlessly with:
- **Claude Code** (Anthropic) - Primary development tool
- **GPT-4/GPT-4o** (OpenAI) - Via ChatGPT or API
- **Cursor IDE** (cursor.sh) - VS Code with AI superpowers
- **GitHub Copilot** - IDE integrations

## Project Guidelines

### Structure
```
ai-coding-playground/
├── projects/           # Completed, production-ready projects
│   └── ffmpeg-rust-port/
├── experiments/        # Work-in-progress AI experiments
├── docs/              # Shared documentation
└── README.md          # This file
```

### Adding New Projects
1. Create a subdirectory in `projects/` or `experiments/`
2. Follow the same documentation standards as ffmpeg-rust-port
3. Document which AI tool was used
4. Include test coverage information

### Code Quality Standards
- Comprehensive error handling
- Full test coverage (unit tests minimum)
- Clear documentation and README
- Zero compiler warnings
- Comments only where logic isn't self-evident

---

## Development with AI Tools

### Best Practices for AI-Assisted Development

**Claude Code (Recommended)**:
```bash
cd projects/ffmpeg-rust-port
claude-code  # Start Claude Code session
```

**With Other Tools**:
- Provide clear context and requirements
- Review AI-generated code carefully
- Run tests after each change
- Use version control frequently

### Workflow
1. Plan implementation with AI tool
2. AI generates code
3. Review and test thoroughly
4. Commit with clear messages
5. Document rationale in README

---

## Repository Management

### Building & Testing All Projects

```bash
# Test ffmpeg-rust-port
cd projects/ffmpeg-rust-port
cargo test --lib

# Build release version
cargo build --release
```

### CI/CD Status
- All tests passing ✅
- Build verified ✅
- No breaking changes ✅

---

## Technical Stack

### Primary Technologies
- **Language**: Rust (Edition 2024)
- **Build**: Cargo
- **Testing**: cargo test
- **Audio**: Symphonia, hound crates

### Editor Support
- VS Code
- Cursor IDE
- IntelliJ IDEA
- Vim/Neovim with rust-analyzer

---

## License

This is an educational project exploring AI-assisted development. Component licenses:
- FFmpeg: LGPL/GPL (original)
- Rust crates: See individual Cargo.toml files

---

## Getting Started

### Prerequisites
- Rust 2024 edition (latest stable)
- Cargo package manager
- Git

### Quick Start
```bash
# Clone and navigate
cd ai-coding-playground/projects/ffmpeg-rust-port

# Run tests
cargo test --lib

# Build library
cargo build --lib

# Show project info
./target/debug/ffmpeg
```

---

## Future Experiments

Planned additions to the playground:
- [ ] Real-time audio processing
- [ ] Video transcoding in Rust
- [ ] WebAssembly audio module
- [ ] AI-optimized compression algorithm
- [ ] Distributed processing pipeline

---

## Resources

- [FFmpeg Original](https://ffmpeg.org/)
- [Rust Audio Ecosystem](https://rust-audio.org/)
- [Symphonia Documentation](https://docs.rs/symphonia/)
- [Claude Code Documentation](https://claude.com/claude-code)

---

**Last Updated**: 2026-01-15
**Rust Edition**: 2024
**Total Projects**: 1 (Complete)
**Test Coverage**: 21 unit tests, 100% passing
