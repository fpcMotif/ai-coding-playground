use std::io;
use thiserror::Error;

/// Result type for audio operations
pub type AudioResult<T> = Result<T, AudioError>;

/// Comprehensive error types for audio processing
#[derive(Error, Debug)]
pub enum AudioError {
    /// IO error (file operations, disk access)
    #[error("IO error: {0}")]
    Io(#[from] io::Error),

    /// Unsupported audio format
    #[error("Unsupported audio format: {0}")]
    UnsupportedFormat(String),

    /// Invalid audio metadata
    #[error("Invalid audio metadata: {0}")]
    InvalidMetadata(String),

    /// Decoding failed
    #[error("Decode error: {0}")]
    DecodeError(String),

    /// Encoding failed
    #[error("Encode error: {0}")]
    EncodeError(String),

    /// Resampling operation failed
    #[error("Resampling error: {0}")]
    ResamplingError(String),

    /// Invalid channel configuration
    #[error("Invalid channel configuration: expected {expected}, got {got}")]
    InvalidChannels {
        /// Expected number of channels
        expected: u32,
        /// Got number of channels
        got: u32,
    },

    /// Invalid sample rate
    #[error("Invalid sample rate: {rate}")]
    InvalidSampleRate {
        /// The invalid sample rate
        rate: u32,
    },

    /// Buffer-related error
    #[error("Buffer error: {0}")]
    BufferError(String),

    /// Segmentation operation failed
    #[error("Segmentation error: {0}")]
    SegmentationError(String),

    /// Configuration error
    #[error("Configuration error: {0}")]
    ConfigError(String),

    /// Audio processing error
    #[error("Processing error: {0}")]
    ProcessingError(String),
}

impl From<symphonia::core::errors::Error> for AudioError {
    fn from(err: symphonia::core::errors::Error) -> Self {
        AudioError::DecodeError(err.to_string())
    }
}

impl From<hound::Error> for AudioError {
    fn from(err: hound::Error) -> Self {
        match err {
            hound::Error::IoError(e) => AudioError::Io(e),
            e => AudioError::EncodeError(e.to_string()),
        }
    }
}
