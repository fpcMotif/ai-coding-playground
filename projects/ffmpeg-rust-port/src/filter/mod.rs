//! Audio filter implementations

pub mod resample;
pub mod remix;
pub mod normalize;

pub use resample::Resample;
pub use remix::Remix;
pub use normalize::Normalize;

use crate::core::AudioFrame;
use crate::error::AudioResult;

/// Trait for audio filters
pub trait Filter {
    /// Process an audio frame through this filter
    fn process(&mut self, frame: &AudioFrame) -> AudioResult<AudioFrame>;

    /// Flush any remaining audio from the filter
    fn flush(&mut self) -> AudioResult<Option<AudioFrame>> {
        Ok(None)
    }
}
