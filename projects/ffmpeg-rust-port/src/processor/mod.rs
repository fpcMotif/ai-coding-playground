//! Audio processing pipeline implementations

pub mod segment;

pub use segment::Segment;

/// Audio processing pipeline result
#[derive(Debug)]
pub struct ProcessingStats {
    /// Total frames processed
    pub frames_processed: u64,
    /// Total samples processed
    pub samples_processed: u64,
}
