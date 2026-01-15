use crate::core::{AudioFrame, Channels};
use crate::error::{AudioError, AudioResult};
use std::time::Duration;

/// Audio segmentation - split audio into time-based chunks
#[derive(Debug, Clone)]
pub struct Segment {
    /// Segment duration
    duration: Duration,
    /// Sample rate
    sample_rate: u32,
    /// Segment index counter
    segment_index: u32,
}

impl Segment {
    /// Create a new segmenter
    pub fn new(duration: Duration, sample_rate: u32) -> AudioResult<Self> {
        if sample_rate == 0 {
            return Err(AudioError::InvalidSampleRate { rate: 0 });
        }

        Ok(Segment {
            duration,
            sample_rate,
            segment_index: 0,
        })
    }

    /// Calculate the number of samples per segment
    pub fn samples_per_segment(&self) -> usize {
        (self.duration.as_secs_f64() * self.sample_rate as f64).ceil() as usize
    }

    /// Split frame(s) by duration into segments
    pub fn split_frame(&mut self, frame: &AudioFrame) -> AudioResult<Vec<AudioFrame>> {
        if frame.sample_rate() != self.sample_rate {
            return Err(AudioError::InvalidSampleRate {
                rate: frame.sample_rate(),
            });
        }

        let samples_per_segment = self.samples_per_segment();
        let num_channels = frame.channels().count() as usize;
        let total_samples = frame.samples();
        let samples_per_channel = total_samples.len() / num_channels;

        let mut segments = Vec::new();

        // Split the frame into segments of the specified duration
        for segment_start in (0..samples_per_channel).step_by(samples_per_segment) {
            let segment_end = std::cmp::min(segment_start + samples_per_segment, samples_per_channel);
            let samples_in_segment = segment_end - segment_start;

            if samples_in_segment == 0 {
                break;
            }

            // Extract samples for this segment (all channels)
            let mut segment_samples = Vec::new();
            for ch in 0..num_channels {
                for sample_idx in segment_start..segment_end {
                    let global_idx = sample_idx * num_channels + ch;
                    if global_idx < total_samples.len() {
                        segment_samples.push(total_samples[global_idx]);
                    }
                }
            }

            // Create the segment frame
            let segment_frame = AudioFrame::new(
                segment_samples,
                self.sample_rate,
                frame.channels(),
                self.segment_index as u64,
            )?;

            segments.push(segment_frame);
            self.segment_index += 1;
        }

        Ok(segments)
    }

    /// Get the current segment index
    pub fn segment_index(&self) -> u32 {
        self.segment_index
    }

    /// Reset segment counter
    pub fn reset(&mut self) {
        self.segment_index = 0;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_segment_creation() {
        let segment = Segment::new(Duration::from_secs(1), 44100);
        assert!(segment.is_ok());
        let s = segment.unwrap();
        assert_eq!(s.samples_per_segment(), 44100);
    }

    #[test]
    fn test_segment_invalid_rate() {
        let segment = Segment::new(Duration::from_secs(1), 0);
        assert!(segment.is_err());
    }

    #[test]
    fn test_split_frame() {
        let mut segment = Segment::new(Duration::from_secs(1), 44100).unwrap();

        // Create a test frame with 88200 samples (2 seconds at 44100 Hz)
        let mut samples = Vec::new();
        for _ in 0..88200 {
            samples.push(0.0);
        }
        let frame = AudioFrame::new(samples, 44100, Channels::Mono, 0).unwrap();

        let segments = segment.split_frame(&frame).unwrap();

        // Should split into 2 segments of ~44100 samples each
        assert_eq!(segments.len(), 2);
        assert_eq!(segments[0].samples_per_channel(), 44100);
    }
}
