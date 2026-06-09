//! Audio filter implementations

pub mod normalize;
pub mod remix;
pub mod resample;

pub use normalize::Normalize;
pub use remix::Remix;
pub use resample::Resample;

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
