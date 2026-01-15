use crate::error::{AudioError, AudioResult};
use std::time::Duration;

/// Channel configuration for audio
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[allow(dead_code)]
pub enum Channels {
    /// Mono (1 channel)
    Mono = 1,
    /// Stereo (2 channels)
    Stereo = 2,
    /// Quad (4 channels)
    Quad = 4,
    /// 5.1 surround sound
    SurroundFivePointOne = 6,
    /// 7.1 surround sound
    SurroundSevenPointOne = 8,
}

impl Channels {
    /// Create Channels from channel count
    pub fn from_count(count: u32) -> AudioResult<Self> {
        match count {
            1 => Ok(Channels::Mono),
            2 => Ok(Channels::Stereo),
            4 => Ok(Channels::Quad),
            6 => Ok(Channels::SurroundFivePointOne),
            8 => Ok(Channels::SurroundSevenPointOne),
            n => Err(AudioError::InvalidChannels {
                expected: 1,
                got: n,
            }),
        }
    }

    /// Get the number of channels
    pub fn count(&self) -> u32 {
        *self as u32
    }

    /// Get channel layout name
    pub fn name(&self) -> &'static str {
        match self {
            Channels::Mono => "Mono",
            Channels::Stereo => "Stereo",
            Channels::Quad => "Quad",
            Channels::SurroundFivePointOne => "5.1 Surround",
            Channels::SurroundSevenPointOne => "7.1 Surround",
        }
    }
}

/// Bit depth for audio samples
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[allow(dead_code)]
pub enum BitDepth {
    /// 8-bit unsigned (0-255)
    I8,
    /// 16-bit signed (-32768 to 32767)
    I16,
    /// 24-bit signed
    I24,
    /// 32-bit signed
    I32,
    /// 32-bit floating point (internal standard)
    F32,
    /// 64-bit floating point
    F64,
}

impl BitDepth {
    /// Get bytes per sample
    pub fn bytes_per_sample(&self) -> usize {
        match self {
            BitDepth::I8 => 1,
            BitDepth::I16 => 2,
            BitDepth::I24 => 3,
            BitDepth::I32 => 4,
            BitDepth::F32 => 4,
            BitDepth::F64 => 8,
        }
    }
}

/// Audio frame containing samples and metadata
#[derive(Debug, Clone)]
pub struct AudioFrame {
    /// Audio samples (interleaved for multiple channels, f32 from -1.0 to 1.0)
    samples: Vec<f32>,
    /// Sample rate in Hz (e.g., 44100, 48000, 16000)
    sample_rate: u32,
    /// Number of channels
    channels: Channels,
    /// Frame number in the audio stream
    frame_number: u64,
    /// Timestamp in the audio stream
    timestamp: Duration,
}

impl AudioFrame {
    /// Create a new audio frame
    pub fn new(
        samples: Vec<f32>,
        sample_rate: u32,
        channels: Channels,
        frame_number: u64,
    ) -> AudioResult<Self> {
        if sample_rate == 0 {
            return Err(AudioError::InvalidSampleRate { rate: sample_rate });
        }

        let expected_samples = (samples.len() as u32 / channels.count()) as usize;
        if samples.len() % channels.count() as usize != 0 {
            return Err(AudioError::BufferError(
                "Sample count not divisible by channel count".to_string(),
            ));
        }

        let timestamp = Duration::from_secs_f64(expected_samples as f64 / sample_rate as f64);

        Ok(AudioFrame {
            samples,
            sample_rate,
            channels,
            frame_number,
            timestamp,
        })
    }

    /// Get reference to the samples
    pub fn samples(&self) -> &[f32] {
        &self.samples
    }

    /// Get mutable reference to the samples
    pub fn samples_mut(&mut self) -> &mut [f32] {
        &mut self.samples
    }

    /// Get owned samples (consumes frame)
    pub fn into_samples(self) -> Vec<f32> {
        self.samples
    }

    /// Get sample rate in Hz
    pub fn sample_rate(&self) -> u32 {
        self.sample_rate
    }

    /// Get channel configuration
    pub fn channels(&self) -> Channels {
        self.channels
    }

    /// Get number of samples per channel
    pub fn samples_per_channel(&self) -> usize {
        self.samples.len() / self.channels.count() as usize
    }

    /// Get frame number
    pub fn frame_number(&self) -> u64 {
        self.frame_number
    }

    /// Get timestamp of this frame
    pub fn timestamp(&self) -> Duration {
        self.timestamp
    }

    /// Get duration of this frame
    pub fn duration(&self) -> Duration {
        Duration::from_secs_f64(self.samples_per_channel() as f64 / self.sample_rate as f64)
    }

    /// Check if frame is empty
    pub fn is_empty(&self) -> bool {
        self.samples.is_empty()
    }
}

/// Audio metadata/information
#[derive(Debug, Clone)]
pub struct AudioMetadata {
    /// Total duration of the audio
    pub duration: Option<Duration>,
    /// Sample rate in Hz
    pub sample_rate: u32,
    /// Number of channels
    pub channels: Channels,
    /// Codec name (e.g., "MP3", "FLAC", "AAC")
    pub codec: String,
    /// Bit depth if known
    pub bit_depth: Option<BitDepth>,
    /// Bitrate in bits per second if known
    pub bitrate: Option<u32>,
}

impl AudioMetadata {
    /// Create new metadata
    pub fn new(sample_rate: u32, channels: Channels, codec: String) -> AudioResult<Self> {
        if sample_rate == 0 {
            return Err(AudioError::InvalidSampleRate { rate: sample_rate });
        }

        Ok(AudioMetadata {
            duration: None,
            sample_rate,
            channels,
            codec,
            bit_depth: None,
            bitrate: None,
        })
    }

    /// Set duration
    pub fn with_duration(mut self, duration: Duration) -> Self {
        self.duration = Some(duration);
        self
    }

    /// Set bit depth
    pub fn with_bit_depth(mut self, bit_depth: BitDepth) -> Self {
        self.bit_depth = Some(bit_depth);
        self
    }

    /// Set bitrate
    pub fn with_bitrate(mut self, bitrate: u32) -> Self {
        self.bitrate = Some(bitrate);
        self
    }

    /// Get duration in seconds
    pub fn duration_secs(&self) -> Option<f64> {
        self.duration.map(|d| d.as_secs_f64())
    }

    /// Get bitrate in kbps
    pub fn bitrate_kbps(&self) -> Option<u32> {
        self.bitrate.map(|b| b / 1000)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_channels_from_count() {
        assert_eq!(Channels::from_count(1).unwrap(), Channels::Mono);
        assert_eq!(Channels::from_count(2).unwrap(), Channels::Stereo);
        assert!(Channels::from_count(0).is_err());
        assert!(Channels::from_count(3).is_err());
    }

    #[test]
    fn test_channels_count() {
        assert_eq!(Channels::Mono.count(), 1);
        assert_eq!(Channels::Stereo.count(), 2);
        assert_eq!(Channels::Quad.count(), 4);
    }

    #[test]
    fn test_audio_frame_creation() {
        let samples = vec![0.1, 0.2, 0.3, 0.4];
        let frame = AudioFrame::new(samples, 44100, Channels::Stereo, 0).unwrap();

        assert_eq!(frame.sample_rate(), 44100);
        assert_eq!(frame.channels(), Channels::Stereo);
        assert_eq!(frame.samples_per_channel(), 2);
        assert_eq!(frame.frame_number(), 0);
    }

    #[test]
    fn test_audio_frame_invalid_samples() {
        // Odd number of samples for stereo should fail
        let samples = vec![0.1, 0.2, 0.3];
        let result = AudioFrame::new(samples, 44100, Channels::Stereo, 0);
        assert!(result.is_err());
    }

    #[test]
    fn test_audio_metadata() {
        let metadata = AudioMetadata::new(48000, Channels::Stereo, "MP3".to_string())
            .unwrap()
            .with_duration(Duration::from_secs(60))
            .with_bitrate(192000);

        assert_eq!(metadata.sample_rate, 48000);
        assert_eq!(metadata.channels, Channels::Stereo);
        assert_eq!(metadata.duration_secs(), Some(60.0));
        assert_eq!(metadata.bitrate_kbps(), Some(192));
    }
}
