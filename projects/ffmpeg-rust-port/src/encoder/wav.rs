use crate::core::{AudioFrame, Channels};
use crate::error::{AudioError, AudioResult};
use hound::{WavWriter, WavSpec};
use std::path::Path;

/// WAV audio encoder
pub struct WavEncoder {
    writer: Option<WavWriter<std::io::BufWriter<std::fs::File>>>,
    sample_rate: u32,
    channels: Channels,
}

impl WavEncoder {
    /// Create a new WAV encoder to file
    pub fn new<P: AsRef<Path>>(
        path: P,
        sample_rate: u32,
        channels: Channels,
    ) -> AudioResult<Self> {
        let spec = WavSpec {
            channels: channels.count() as u16,
            sample_rate,
            bits_per_sample: 32,
            sample_format: hound::SampleFormat::Float,
        };

        let writer = WavWriter::create(path, spec)
            .map_err(|e| AudioError::EncodeError(e.to_string()))?;

        Ok(WavEncoder {
            writer: Some(writer),
            sample_rate,
            channels,
        })
    }

    /// Get the sample rate
    pub fn sample_rate(&self) -> u32 {
        self.sample_rate
    }

    /// Get the channel configuration
    pub fn channels(&self) -> Channels {
        self.channels
    }

    /// Get the number of frames written
    pub fn frames_written(&self) -> u32 {
        self.writer.as_ref().map(|w| w.len()).unwrap_or(0)
    }
}

impl super::Encoder for WavEncoder {
    fn encode(&mut self, frame: &AudioFrame) -> AudioResult<()> {
        if frame.sample_rate() != self.sample_rate {
            return Err(AudioError::InvalidSampleRate {
                rate: frame.sample_rate(),
            });
        }

        if frame.channels() != self.channels {
            return Err(AudioError::InvalidChannels {
                expected: self.channels.count(),
                got: frame.channels().count(),
            });
        }

        let writer = self.writer.as_mut()
            .ok_or_else(|| AudioError::ProcessingError("Encoder already finalized".to_string()))?;

        // Write each sample to the WAV file
        for &sample in frame.samples() {
            writer
                .write_sample(sample)
                .map_err(|e| AudioError::EncodeError(e.to_string()))?;
        }

        Ok(())
    }

    fn finalize(&mut self) -> AudioResult<()> {
        if let Some(writer) = self.writer.take() {
            writer
                .finalize()
                .map_err(|e| AudioError::EncodeError(e.to_string()))?;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::encoder::Encoder;
    use tempfile::NamedTempFile;

    #[test]
    fn test_wav_encoder_creation() {
        let temp_file = NamedTempFile::new().unwrap();
        let encoder = WavEncoder::new(temp_file.path(), 44100, Channels::Stereo);
        assert!(encoder.is_ok());
    }

    #[test]
    fn test_wav_encoder_write() {
        let temp_file = NamedTempFile::new().unwrap();
        let mut encoder = WavEncoder::new(temp_file.path(), 44100, Channels::Mono).unwrap();

        // Create a test frame
        let samples = vec![0.0, 0.1, -0.1, 0.5];
        let frame = AudioFrame::new(samples, 44100, Channels::Mono, 0).unwrap();

        // Write the frame
        let result = encoder.encode(&frame);
        assert!(result.is_ok());

        // Verify frame count
        assert_eq!(encoder.frames_written(), 4);

        // Finalize
        assert!(encoder.finalize().is_ok());
    }

    #[test]
    fn test_wav_encoder_invalid_sample_rate() {
        let temp_file = NamedTempFile::new().unwrap();
        let mut encoder = WavEncoder::new(temp_file.path(), 44100, Channels::Mono).unwrap();

        // Try to write with wrong sample rate
        let samples = vec![0.0, 0.1];
        let frame = AudioFrame::new(samples, 48000, Channels::Mono, 0).unwrap();

        let result = encoder.encode(&frame);
        assert!(result.is_err());
    }

    #[test]
    fn test_wav_encoder_invalid_channels() {
        let temp_file = NamedTempFile::new().unwrap();
        let mut encoder = WavEncoder::new(temp_file.path(), 44100, Channels::Mono).unwrap();

        // Try to write stereo data to mono encoder
        let samples = vec![0.0, 0.1, 0.2, 0.3];
        let frame = AudioFrame::new(samples, 44100, Channels::Stereo, 0).unwrap();

        let result = encoder.encode(&frame);
        assert!(result.is_err());
    }
}
