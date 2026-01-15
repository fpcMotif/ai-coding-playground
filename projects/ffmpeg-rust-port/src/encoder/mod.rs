//! Audio encoder implementations

pub mod wav;

pub use wav::WavEncoder;

use crate::core::AudioFrame;
use crate::error::AudioResult;

/// Trait for audio encoders
pub trait Encoder {
    /// Encode an audio frame to output
    fn encode(&mut self, frame: &AudioFrame) -> AudioResult<()>;

    /// Finalize encoding (flush any remaining data)
    fn finalize(&mut self) -> AudioResult<()> {
        Ok(())
    }
}
