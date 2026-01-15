use crate::core::AudioFrame;
use crate::error::{AudioError, AudioResult};

/// Audio normalization filter - adjusts volume levels
#[derive(Clone, Debug)]
pub struct Normalize {
    /// Target peak level (0.0 to 1.0)
    target_peak: f32,
    /// Whether to use loudness normalization (true) or peak normalization (false)
    use_loudness: bool,
}

impl Normalize {
    /// Create a peak normalizer (normalizes to target peak level)
    pub fn peak(target_peak: f32) -> AudioResult<Self> {
        if target_peak <= 0.0 || target_peak > 1.0 {
            return Err(AudioError::ConfigError(format!(
                "Target peak must be between 0.0 and 1.0, got {}",
                target_peak
            )));
        }

        Ok(Normalize {
            target_peak,
            use_loudness: false,
        })
    }

    /// Create a loudness normalizer
    /// Uses RMS (Root Mean Square) instead of peak
    pub fn loudness(target_loudness: f32) -> AudioResult<Self> {
        if target_loudness <= 0.0 || target_loudness > 1.0 {
            return Err(AudioError::ConfigError(format!(
                "Target loudness must be between 0.0 and 1.0, got {}",
                target_loudness
            )));
        }

        Ok(Normalize {
            target_peak: target_loudness,
            use_loudness: true,
        })
    }

    /// Calculate peak level of audio samples
    fn calculate_peak(samples: &[f32]) -> f32 {
        samples
            .iter()
            .map(|&s| s.abs())
            .fold(0.0f32, |a, b| a.max(b))
    }

    /// Calculate RMS (loudness) of audio samples
    fn calculate_rms(samples: &[f32]) -> f32 {
        if samples.is_empty() {
            return 0.0;
        }

        let sum_squared: f32 = samples.iter().map(|&s| s * s).sum();
        (sum_squared / samples.len() as f32).sqrt()
    }

    /// Apply gain to all samples
    fn apply_gain(samples: &[f32], gain: f32) -> Vec<f32> {
        samples.iter().map(|&s| (s * gain).clamp(-1.0, 1.0)).collect()
    }
}

impl super::Filter for Normalize {
    fn process(&mut self, frame: &AudioFrame) -> AudioResult<AudioFrame> {
        let samples = frame.samples();

        if samples.is_empty() {
            return Ok(frame.clone());
        }

        let current_level = if self.use_loudness {
            Self::calculate_rms(samples)
        } else {
            Self::calculate_peak(samples)
        };

        if current_level == 0.0 {
            return Ok(frame.clone());
        }

        // Calculate gain needed to reach target
        let gain = self.target_peak / current_level;

        // Apply gain (with clipping to prevent distortion)
        let normalized_samples = Self::apply_gain(samples, gain);

        // Create output frame
        let output_frame = AudioFrame::new(
            normalized_samples,
            frame.sample_rate(),
            frame.channels(),
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
    fn test_peak_normalization() {
        // Create a normalizer targeting 0.8
        let mut normalizer = Normalize::peak(0.8).unwrap();

        // Create a frame with peak of 0.5
        let samples = vec![0.0, 0.25, 0.5, -0.3];
        let frame = AudioFrame::new(
            samples,
            44100,
            crate::core::Channels::Mono,
            0,
        ).unwrap();

        let result = normalizer.process(&frame).unwrap();

        // Peak should now be 0.8
        let peak = Normalize::calculate_peak(result.samples());
        assert!((peak - 0.8).abs() < 0.001);
    }

    #[test]
    fn test_rms_normalization() {
        let mut normalizer = Normalize::loudness(0.5).unwrap();

        let samples = vec![0.1, 0.2, -0.15, 0.1];
        let frame = AudioFrame::new(
            samples.clone(),
            44100,
            crate::core::Channels::Mono,
            0,
        ).unwrap();

        let result = normalizer.process(&frame).unwrap();

        // Calculate the RMS of the original
        let original_rms = Normalize::calculate_rms(&samples);
        let new_rms = Normalize::calculate_rms(result.samples());

        // New RMS should be approximately the target
        assert!((new_rms - 0.5).abs() < 0.01);
    }

    #[test]
    fn test_silence_handling() {
        let mut normalizer = Normalize::peak(0.8).unwrap();

        let samples = vec![0.0, 0.0, 0.0];
        let frame = AudioFrame::new(
            samples.clone(),
            44100,
            crate::core::Channels::Mono,
            0,
        ).unwrap();

        // Should handle silence gracefully (no division by zero)
        let result = normalizer.process(&frame).unwrap();
        assert_eq!(result.samples(), &[0.0, 0.0, 0.0]);
    }
}
