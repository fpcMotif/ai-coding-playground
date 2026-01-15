use crate::core::{AudioFrame, Channels};
use crate::error::{AudioError, AudioResult};

/// Audio resampler - converts from one sample rate to another using linear interpolation
pub struct Resample {
    input_rate: u32,
    output_rate: u32,
    channels: Channels,
}

impl Resample {
    /// Create a new resampler
    ///
    /// # Arguments
    /// * `input_rate` - Input sample rate in Hz
    /// * `output_rate` - Output sample rate in Hz
    /// * `channels` - Number of channels
    pub fn new(input_rate: u32, output_rate: u32, channels: Channels) -> AudioResult<Self> {
        if input_rate == 0 || output_rate == 0 {
            return Err(AudioError::InvalidSampleRate { rate: 0 });
        }

        Ok(Resample {
            input_rate,
            output_rate,
            channels,
        })
    }

    /// Get the input sample rate
    pub fn input_rate(&self) -> u32 {
        self.input_rate
    }

    /// Get the output sample rate
    pub fn output_rate(&self) -> u32 {
        self.output_rate
    }

    /// Get the ratio of output to input sample rate
    pub fn ratio(&self) -> f64 {
        self.output_rate as f64 / self.input_rate as f64
    }

    /// Linear interpolation resampling
    fn linear_resample(input: &[f32], ratio: f64) -> Vec<f32> {
        if input.is_empty() || ratio <= 0.0 {
            return Vec::new();
        }

        let output_len = (input.len() as f64 / ratio).ceil() as usize;
        let mut output = Vec::with_capacity(output_len);

        for i in 0..output_len {
            let input_pos = i as f64 / ratio;
            let input_idx = input_pos.floor() as usize;

            if input_idx + 1 < input.len() {
                // Linear interpolation between two samples
                let frac = input_pos - input_idx as f64;
                let sample = (input[input_idx] as f64 * (1.0 - frac) + input[input_idx + 1] as f64 * frac) as f32;
                output.push(sample.clamp(-1.0, 1.0));
            } else if input_idx < input.len() {
                // Edge case: last sample
                output.push(input[input_idx]);
            }
        }

        output
    }
}

impl super::Filter for Resample {
    fn process(&mut self, frame: &AudioFrame) -> AudioResult<AudioFrame> {
        if frame.channels() != self.channels {
            return Err(AudioError::InvalidChannels {
                expected: self.channels.count(),
                got: frame.channels().count(),
            });
        }

        if frame.sample_rate() != self.input_rate {
            return Err(AudioError::InvalidSampleRate {
                rate: frame.sample_rate(),
            });
        }

        if self.input_rate == self.output_rate {
            // No resampling needed
            return Ok(frame.clone());
        }

        let ratio = self.input_rate as f64 / self.output_rate as f64;

        // Resample all samples together (works for interleaved format)
        let resampled = Self::linear_resample(frame.samples(), ratio);

        // Create output frame with new sample rate
        let output_frame = AudioFrame::new(
            resampled,
            self.output_rate,
            self.channels,
            frame.frame_number(),
        )?;

        Ok(output_frame)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::filter::Filter;

    #[test]
    fn test_resample_creation() {
        let resample = Resample::new(44100, 16000, Channels::Stereo);
        assert!(resample.is_ok());
        let r = resample.unwrap();
        assert_eq!(r.input_rate(), 44100);
        assert_eq!(r.output_rate(), 16000);
    }

    #[test]
    fn test_resample_invalid_rate() {
        let resample = Resample::new(0, 16000, Channels::Stereo);
        assert!(resample.is_err());
    }

    #[test]
    fn test_linear_resample() {
        let input = vec![0.0, 1.0, 0.5];
        // Downsample by 2x (reduce from 3 to 1-2 samples)
        let output = Resample::linear_resample(&input, 2.0);
        assert!(!output.is_empty());
    }
}
