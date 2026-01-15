use crate::core::{AudioFrame, Channels};
use crate::error::{AudioError, AudioResult};

/// Audio channel remixer - converts between channel layouts
pub struct Remix {
    input_channels: Channels,
    output_channels: Channels,
}

impl Remix {
    /// Create a new channel remixer
    pub fn new(input_channels: Channels, output_channels: Channels) -> AudioResult<Self> {
        Ok(Remix {
            input_channels,
            output_channels,
        })
    }

    /// Remix stereo to mono by averaging channels
    fn stereo_to_mono(input: &[f32]) -> Vec<f32> {
        let mut output = Vec::new();
        for i in (0..input.len()).step_by(2) {
            if i + 1 < input.len() {
                let avg = (input[i] + input[i + 1]) / 2.0;
                output.push(avg);
            }
        }
        output
    }

    /// Remix mono to stereo by duplicating the channel
    fn mono_to_stereo(input: &[f32]) -> Vec<f32> {
        let mut output = Vec::new();
        for &sample in input {
            output.push(sample);
            output.push(sample); // Duplicate to both channels
        }
        output
    }

    /// Extract left channel from stereo
    fn stereo_left(input: &[f32]) -> Vec<f32> {
        let mut output = Vec::new();
        for i in (0..input.len()).step_by(2) {
            output.push(input[i]);
        }
        output
    }

    /// Extract right channel from stereo
    fn stereo_right(input: &[f32]) -> Vec<f32> {
        let mut output = Vec::new();
        for i in (0..input.len()).step_by(2) {
            if i + 1 < input.len() {
                output.push(input[i + 1]);
            }
        }
        output
    }
}

impl super::Filter for Remix {
    fn process(&mut self, frame: &AudioFrame) -> AudioResult<AudioFrame> {
        if frame.channels() != self.input_channels {
            return Err(AudioError::InvalidChannels {
                expected: self.input_channels.count(),
                got: frame.channels().count(),
            });
        }

        let samples = frame.samples();

        // Handle common remixing operations
        let output_samples = match (self.input_channels, self.output_channels) {
            // Pass through same channel count
            (src, dst) if src == dst => samples.to_vec(),

            // Stereo to Mono
            (Channels::Stereo, Channels::Mono) => Self::stereo_to_mono(samples),

            // Mono to Stereo
            (Channels::Mono, Channels::Stereo) => Self::mono_to_stereo(samples),

            // Quad to Stereo (average all channels)
            (Channels::Quad, Channels::Stereo) => {
                let mut output = Vec::new();
                // Assuming quad is FLRR (Front-Left, Front-Right, Rear-Left, Rear-Right)
                for i in (0..samples.len()).step_by(4) {
                    if i + 3 < samples.len() {
                        let left = (samples[i] + samples[i + 2]) / 2.0; // FL + RL
                        let right = (samples[i + 1] + samples[i + 3]) / 2.0; // FR + RR
                        output.push(left);
                        output.push(right);
                    }
                }
                output
            }

            // Stereo to Left Only
            (Channels::Stereo, other) if other == Channels::Mono => Self::stereo_left(samples),

            _ => {
                return Err(AudioError::ProcessingError(format!(
                    "Remix from {} to {} not yet supported",
                    self.input_channels.name(),
                    self.output_channels.name()
                )))
            }
        };

        // Create output frame
        let output_frame = AudioFrame::new(
            output_samples,
            frame.sample_rate(),
            self.output_channels,
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
    fn test_remix_stereo_to_mono() {
        // Create test stereo samples: [L1, R1, L2, R2]
        let input = vec![0.0, 1.0, 0.5, 0.5];
        let output = Remix::stereo_to_mono(&input);

        // Expected: [(0+1)/2, (0.5+0.5)/2] = [0.5, 0.5]
        assert_eq!(output.len(), 2);
        assert!((output[0] - 0.5).abs() < 0.001);
        assert!((output[1] - 0.5).abs() < 0.001);
    }

    #[test]
    fn test_remix_mono_to_stereo() {
        let input = vec![0.5, 0.8];
        let output = Remix::mono_to_stereo(&input);

        // Expected: [0.5, 0.5, 0.8, 0.8]
        assert_eq!(output.len(), 4);
        assert_eq!(output[0], 0.5);
        assert_eq!(output[1], 0.5);
        assert_eq!(output[2], 0.8);
        assert_eq!(output[3], 0.8);
    }
}
