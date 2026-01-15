//! Audio decoder implementations

pub mod symphonia;

pub use symphonia::SymphoniaDecoder;

use crate::core::AudioFrame;
use crate::error::AudioResult;
use std::path::Path;

/// Trait for audio decoders
pub trait Decoder: Send {
    /// Get next audio frame from the stream
    fn decode_frame(&mut self) -> AudioResult<Option<AudioFrame>>;

    /// Check if decoder is finished
    fn is_finished(&self) -> bool;

    /// Reset decoder to beginning (if supported)
    fn reset(&mut self) -> AudioResult<()> {
        Err(crate::error::AudioError::ProcessingError(
            "Reset not supported for this decoder".to_string(),
        ))
    }
}

/// Create a decoder from a file path
pub fn from_file<P: AsRef<Path>>(path: P) -> AudioResult<Box<dyn Decoder>> {
    let path = path.as_ref();
    SymphoniaDecoder::from_file(path).map(|d| Box::new(d) as Box<dyn Decoder>)
}
