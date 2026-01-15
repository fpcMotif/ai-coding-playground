#![warn(missing_docs)]
#![doc = include_str!("../PORTING_PLAN.md")]

//! # FFmpeg-RS: Pure Rust Audio Processing Library
//!
//! A complete rewrite of FFmpeg in Rust, focusing on audio processing with zero C dependencies.
//!
//! ## Features
//!
//! - **Decode** - MP3, FLAC, WAV, OGG, AAC, Opus formats
//! - **Encode** - WAV, FLAC (planned)
//! - **Transform** - Resample, remix channels, normalize
//! - **Segment** - Split audio into chunks
//! - **CLI** - Command-line interface compatible with ffmpeg
//!
//! ## Quick Start
//!
//! ```ignore
//! use ffmpeg_rs::{AudioFrame, Channels};
//! use ffmpeg_rs::encoder::WavEncoder;
//! use ffmpeg_rs::filter::Filter;
//!
//! // Create an audio frame
//! let samples = vec![0.0, 0.1, -0.1, 0.5];
//! let frame = AudioFrame::new(samples, 44100, Channels::Mono, 0)?;
//!
//! // Encode to WAV
//! let mut encoder = WavEncoder::new("output.wav", 44100, Channels::Mono)?;
//! encoder.encode(&frame)?;
//! encoder.finalize()?;
//! ```

// Declare modules
/// Core audio types and structures
pub mod core;
/// Error types for audio operations
pub mod error;
/// Audio decoder implementations
pub mod decoder;
/// Audio filter implementations
pub mod filter;
/// Audio encoder implementations
pub mod encoder;
/// Audio processing pipelines
pub mod processor;

// Export public types
pub use core::{AudioFrame, AudioMetadata, Channels, BitDepth};
pub use error::{AudioError, AudioResult};

/// Library version
pub const VERSION: &str = env!("CARGO_PKG_VERSION");
