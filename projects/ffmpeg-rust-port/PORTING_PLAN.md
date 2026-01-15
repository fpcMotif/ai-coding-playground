# FFmpeg to Rust Port - Comprehensive Plan

## Project Overview
Porting FFmpeg (audio/video processing library) to pure Rust, starting with **core audio processing** functionality.

**Location:** `/Users/f/ffmpeg-rust-port/`
**FFmpeg Source:** `/Users/f/ffmpeg-rust-port/ffmpeg-source/` (being cloned)

---

## Architecture & Core Design

### Technology Stack
- **Language:** Rust 2021 edition
- **Audio Decoding:** Symphonia (pure Rust, supports MP3, FLAC, WAV, OGG, AAC, Opus)
- **WAV I/O:** Hound
- **Resampling:** Rubato (for sample rate conversion)
- **Error Handling:** Thiserror
- **CLI:** Clap
- **Zero C dependencies** - Complete isolation from FFmpeg C code

### Project Structure
```
ffmpeg-rs/
├── src/
│   ├── lib.rs              # Library root
│   ├── main.rs             # CLI binary
│   ├── core/
│   │   ├── mod.rs
│   │   ├── audio.rs        # Audio frame/sample handling
│   │   ├── codec.rs        # Codec information
│   │   └── metadata.rs     # Duration, sample rate, channels
│   ├── decoder/
│   │   ├── mod.rs
│   │   └── symphonia.rs    # Symphonia-based decoder
│   ├── encoder/
│   │   ├── mod.rs
│   │   ├── wav.rs          # WAV encoder
│   │   └── flac.rs         # FLAC encoder (future)
│   ├── filter/
│   │   ├── mod.rs
│   │   ├── resample.rs     # Resampling
│   │   ├── remix.rs        # Channel manipulation
│   │   └── normalize.rs    # Volume normalization
│   ├── processor/
│   │   ├── mod.rs
│   │   ├── stream.rs       # Stream-based processing
│   │   └── segment.rs      # Audio segmentation
│   ├── error.rs            # Error types
│   └── cli/
│       ├── mod.rs
│       └── commands.rs     # CLI command handlers
├── Cargo.toml
├── PORTING_PLAN.md
└── ffmpeg-source/          # Official FFmpeg C source (reference)
```

---

## Implementation Phases

### Phase 1: Core Infrastructure (Foundation)
**Goal:** Establish basic audio frame handling and error types

**Tasks:**
1. Define error types with proper context (thiserror)
2. Create `AudioFrame` struct (samples, sample_rate, channels, timestamp)
3. Create `AudioMetadata` struct (duration, codec, format)
4. Implement trait system for extensibility
5. Setup logging infrastructure

**Files to Create:**
- `src/error.rs` - Custom error types
- `src/core/mod.rs` - Core module exports
- `src/core/audio.rs` - AudioFrame, AudioBuffer types
- `src/core/metadata.rs` - Metadata structures

**Testing:** Unit tests for frame creation and metadata

---

### Phase 2: Decoding (Read Audio)
**Goal:** Read audio from files (MP3, FLAC, WAV, etc.)

**Tasks:**
1. Create `Decoder` trait defining decode interface
2. Implement Symphonia-based decoder using libmetadata + libcore
3. Handle format detection (based on file extension + magic bytes)
4. Support streaming frames from file (lazy loading for large files)
5. Extract metadata (duration, codec info, sample rate, channels)

**Key Features:**
- Non-blocking frame iteration
- Proper error handling for corrupt files
- Memory-efficient streaming

**Files to Create:**
- `src/decoder/mod.rs` - Decoder trait
- `src/decoder/symphonia.rs` - Symphonia implementation
- `src/core/codec.rs` - Codec information

**Testing:** Test with various audio formats (MP3, FLAC, WAV, AAC, OGG)

---

### Phase 3: Filtering (Process Audio)
**Goal:** Implement common audio transformations

**Tasks:**
1. **Resampling** - Convert 44.1kHz → 16kHz (use Rubato)
2. **Remixing** - Stereo → Mono, Mono → Stereo, channel extraction
3. **Normalization** - Adjust volume levels
4. **Trimming** - Cut audio by time range

**Files to Create:**
- `src/filter/mod.rs` - Filter trait
- `src/filter/resample.rs` - Resampling using Rubato
- `src/filter/remix.rs` - Channel manipulation
- `src/filter/normalize.rs` - Loudness adjustment

**Testing:** Test combinations (decode → resample → remix → encode)

---

### Phase 4: Encoding (Write Audio)
**Goal:** Write audio to files

**Tasks:**
1. Create `Encoder` trait defining encode interface
2. Implement WAV encoder (Hound-based)
3. Implement FLAC encoder (optional, via metaflac + flac crate)
4. Support writing to files and in-memory buffers
5. Preserve metadata (sample rate, channels, bit depth)

**Files to Create:**
- `src/encoder/mod.rs` - Encoder trait
- `src/encoder/wav.rs` - WAV encoding
- `src/encoder/flac.rs` - FLAC encoding (optional)

**Testing:** Test encoding to WAV, verify bit-perfect round-trip (decode → encode)

---

### Phase 5: Processing Pipelines (Advanced Operations)
**Goal:** Combine decoders, filters, encoders into processing chains

**Tasks:**
1. Create `StreamProcessor` for memory-efficient processing
2. Implement audio segmentation (split by duration, number of chunks)
3. Implement transcode operation (input format → output format)
4. Support chaining filters (fluent builder API)
5. Batch processing for multiple files

**Files to Create:**
- `src/processor/mod.rs` - Processor trait
- `src/processor/stream.rs` - StreamProcessor
- `src/processor/segment.rs` - Segmentation logic

**Example API:**
```rust
let processor = StreamProcessor::new()
    .source("input.mp3")?
    .resample(16_000)?
    .remix(Channels::Mono)?
    .normalize()?
    .segment(Duration::from_secs(900))? // 15 min chunks
    .encode_wav("output.wav")?;
```

**Testing:** Test with large files, verify memory usage, segment count accuracy

---

### Phase 6: CLI Interface
**Goal:** Create ffmpeg-rs command-line tool

**Tasks:**
1. Implement subcommands (decode, encode, transcode, segment, probe)
2. Add progress reporting for long operations
3. Support batch processing flags
4. Add verbose/debug output modes
5. Create help documentation

**Example Commands:**
```bash
ffmpeg -i input.mp3 output.wav
ffmpeg -i input.mp3 -ar 16000 -ac 1 output.wav
ffmpeg -i input.mp3 -f segment -segment_time 900 segment_%03d.wav
ffmpeg -i input.mp3 # Probe/metadata
```

**Files to Create:**
- `src/cli/mod.rs`
- `src/cli/commands.rs` - Command implementations
- `src/main.rs` - Entry point

**Testing:** Test all CLI commands with various inputs

---

### Phase 7: Testing & Documentation
**Goal:** Comprehensive test coverage and API documentation

**Tasks:**
1. Integration tests for common workflows
2. Performance benchmarks against FFmpeg
3. Fuzz testing with malformed audio files
4. API documentation (doc comments)
5. Example code in repository
6. CHANGELOG for migration guide

**Test Coverage:**
- Unit tests in each module (inline)
- Integration tests: `tests/integration/`
- Benchmarks: `benches/`

**Testing:** Run `cargo test --all` with coverage target

---

## Reference: FFmpeg C Code Mapping

When porting, reference these FFmpeg C files:

| C Module | Rust Equivalent | Purpose |
|----------|-----------------|---------|
| `libavformat/` | `src/decoder/` | Container format detection & demuxing |
| `libavcodec/` | Symphonia crate | Audio codec decoding |
| `libswresample/` | `src/filter/resample.rs` | Resampling/sample rate conversion |
| `libavfilter/` | `src/filter/` | Audio filters |
| `libavutil/` | `src/core/` | Common utilities & data structures |
| `fftools/ffmpeg.c` | `src/cli/` | Command-line interface |

---

## Key Design Decisions

### ✅ Why Pure Rust?
- **No C dependencies** - Easier deployment, cross-compilation, security
- **Memory safety** - Rust's ownership prevents buffer overflows
- **Better error handling** - Result types instead of error codes
- **Performance** - Comparable to C with modern optimization

### ✅ Why Symphonia?
- **Pure Rust** - No C dependencies
- **Actively maintained** - Catches security issues quickly
- **Good format support** - MP3, FLAC, WAV, OGG, AAC, Opus
- **Streaming API** - Efficient for large files

### ✅ Builder Pattern
Use builder pattern for complex operations:
```rust
let result = Transcode::new("input.mp3")?
    .output_sample_rate(16_000)
    .output_channels(Channels::Mono)
    .resample_quality(Quality::High)
    .output_file("output.wav")?;
```

### ✅ Error Handling
- Use `thiserror` for context-rich errors
- All I/O operations return `Result<T, AudioError>`
- Distinguishing between format errors and I/O errors

---

## Critical Implementation Notes

### 1. Memory Management
- Use `Vec<f32>` for sample buffers (zero-copy when possible)
- Implement frame pooling for high-throughput scenarios
- Stream large files (don't load into memory)

### 2. Audio Sample Format
- **Internal:** 32-bit floating point (f32, -1.0 to 1.0)
- **Input:** Decode to f32 automatically
- **Output:** Convert to target bit depth (16-bit i16 for WAV, etc.)

### 3. Thread Safety
- Mark thread-safe types with `Send + Sync`
- Use `Arc<Mutex<T>>` for shared state if needed
- No global state (functional approach)

### 4. Security
- Validate file headers (prevent zip bombs, false formats)
- Bounds-check all buffer operations
- Limit recursion depth for nested formats
- Sanitize file paths in CLI

### 5. Performance
- Use `SmallVec` for temporary allocations
- Lazy frame decoding (don't decode if filtering unnecessary)
- Parallel processing for batch operations (using rayon)

---

## Testing Strategy

### Unit Tests
```rust
#[cfg(test)]
mod tests {
    #[test]
    fn test_resample_16khz_to_48khz() { ... }

    #[test]
    fn test_stereo_to_mono_remix() { ... }
}
```

### Integration Tests
```rust
// tests/integration/transcode_test.rs
#[test]
fn test_full_workflow_mp3_to_wav() {
    let processor = StreamProcessor::new()
        .source("test_audio/input.mp3")?
        .resample(16_000)?
        .encode_wav("test_audio/output.wav")?;

    let output_metadata = Metadata::from_file("test_audio/output.wav")?;
    assert_eq!(output_metadata.sample_rate(), 16_000);
}
```

### Benchmarks
```rust
// benches/decode_bench.rs
criterion::criterion_group!(benches, decode_mp3_1min, decode_flac_1min);
```

---

## Timeline & Milestones

⚠️ **No time estimates given** - Focus on quality over speed

**Milestones:**
1. Phase 1-2: Basic decode infrastructure working ✓
2. Phase 3-4: Encode/decode round-trip functional ✓
3. Phase 5: Streaming processor tested
4. Phase 6: CLI with all major commands working
5. Phase 7: Full test coverage & documentation
6. Phase 8: Performance optimization & deployment

---

## Verification Checklist

### Code Quality
- ✅ Zero compiler warnings (`cargo clippy`)
- ✅ All tests pass (`cargo test --all`)
- ✅ Code formatting consistent (`cargo fmt`)
- ✅ Documentation complete (`cargo doc`)

### Functional Verification
- ✅ Can decode MP3, FLAC, WAV, OGG files
- ✅ Can encode WAV files
- ✅ Can resample from any rate to 16kHz
- ✅ Can remix stereo to mono
- ✅ Can segment long audio into chunks
- ✅ CLI works with common ffmpeg commands

### Performance Verification
- ✅ Processing speed comparable to ffmpeg
- ✅ Memory usage reasonable for large files
- ✅ No memory leaks (Miri testing)

### Security Verification
- ✅ Handles malformed files gracefully
- ✅ No panic on invalid input
- ✅ No buffer overflows possible (Rust guarantees)

---

## References & Resources

- [Symphonia Audio Format Library](https://github.com/pdx-westernsunrise/symphonia)
- [FFmpeg Documentation](https://ffmpeg.org/documentation.html)
- [Rust Audio Ecosystem](https://rust-audio.org/)
- [Hound WAV Library](https://github.com/ruuda/hound)
- [Rubato Resampling](https://github.com/HEnquist/rubato)

---

**Status:** Ready for Phase 1 Implementation
**Last Updated:** 2025-01-15
