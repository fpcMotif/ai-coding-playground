use crate::core::{AudioFrame, Channels};
use crate::error::{AudioError, AudioResult};
use std::fs::File;
use std::ops::Deref;
use std::path::Path;
use symphonia::core::formats::FormatOptions;
use symphonia::core::io::MediaSourceStream;
use symphonia::core::meta::MetadataOptions;
use symphonia::core::probe::Hint;

/// Symphonia-based audio decoder
pub struct SymphoniaDecoder {
    /// Current reader for the audio source
    reader: Box<dyn symphonia::core::formats::FormatReader>,
    /// Track information
    track_id: u32,
    /// Sample rate
    sample_rate: u32,
    /// Number of channels
    channels: Channels,
    /// Frame counter
    frame_count: u64,
    /// Whether decoding is finished
    finished: bool,
    /// Current decoder state
    decoder: Box<dyn symphonia::core::codecs::Decoder>,
}

impl SymphoniaDecoder {
    /// Create decoder from file path
    pub fn from_file<P: AsRef<Path>>(path: P) -> AudioResult<Self> {
        let path = path.as_ref();

        // Open the file
        let file = Box::new(
            File::open(path)
                .map_err(|e| AudioError::Io(e))?,
        );

        // Create media source stream
        let mss = MediaSourceStream::new(file, Default::default());

        // Probe the file to detect format
        let mut hint = Hint::new();
        if let Some(ext) = path.extension() {
            if let Some(ext_str) = ext.to_str() {
                hint.with_extension(ext_str);
            }
        }

        let format_opts = FormatOptions::default();
        let metadata_opts = MetadataOptions::default();

        let probed = symphonia::default::get_probe()
            .format(&hint, mss, &format_opts, &metadata_opts)
            .map_err(|e| AudioError::UnsupportedFormat(e.to_string()))?;

        let reader = probed.format;

        // Find the first audio track
        let track = reader
            .tracks()
            .iter()
            .find(|t| t.codec_params.codec != symphonia::core::codecs::CODEC_TYPE_NULL)
            .ok_or_else(|| AudioError::InvalidMetadata("No audio track found".to_string()))?
            .clone();

        let track_id = track.id;
        let codec_params = &track.codec_params;

        // Extract sample rate
        let sample_rate = codec_params
            .sample_rate
            .ok_or_else(|| AudioError::InvalidMetadata("Unknown sample rate".to_string()))?;

        // Extract channel info
        let channels = if let Some(channels) = codec_params.channels {
            Channels::from_count(channels.count() as u32)?
        } else {
            return Err(AudioError::InvalidMetadata("Unknown channel count".to_string()));
        };

        // Create decoder
        let decoder = symphonia::default::get_codecs()
            .make(codec_params, &Default::default())
            .map_err(|e| AudioError::DecodeError(e.to_string()))?;

        Ok(SymphoniaDecoder {
            reader,
            track_id,
            sample_rate,
            channels,
            frame_count: 0,
            finished: false,
            decoder,
        })
    }

    /// Get sample rate
    pub fn sample_rate(&self) -> u32 {
        self.sample_rate
    }

    /// Get channels
    pub fn channels(&self) -> Channels {
        self.channels
    }
}

impl super::Decoder for SymphoniaDecoder {
    fn decode_frame(&mut self) -> AudioResult<Option<AudioFrame>> {
        if self.finished {
            return Ok(None);
        }

        loop {
            // Get next packet
            let packet = match self.reader.next_packet() {
                Ok(packet) => packet,
                Err(symphonia::core::errors::Error::IoError(ref e))
                    if e.kind() == std::io::ErrorKind::UnexpectedEof =>
                {
                    self.finished = true;
                    return Ok(None);
                }
                Err(symphonia::core::errors::Error::DecodeError(_)) => {
                    // Skip decode errors and try next packet
                    continue;
                }
                Err(e) => return Err(AudioError::DecodeError(e.to_string())),
            };

            // Only process packets from our audio track
            if packet.track_id() != self.track_id {
                // Skip packets from other tracks
                continue;
            }

            // Decode the packet
            let audio_buf = match self.decoder.decode(&packet) {
                Ok(audio_buf) => audio_buf,
                Err(e) => return Err(AudioError::DecodeError(e.to_string())),
            };

            // Convert Symphonia's AudioBuffer to our f32 samples
            let mut samples = Vec::new();

            // Determine the number of samples by getting the number of frames
            let num_samples = match &audio_buf {
                symphonia::core::audio::AudioBufferRef::F32(buf) => buf.as_ref().capacity(),
                symphonia::core::audio::AudioBufferRef::S32(buf) => buf.as_ref().capacity(),
                symphonia::core::audio::AudioBufferRef::S16(buf) => buf.as_ref().capacity(),
                symphonia::core::audio::AudioBufferRef::S24(buf) => buf.as_ref().capacity(),
                symphonia::core::audio::AudioBufferRef::S8(buf) => buf.as_ref().capacity(),
                symphonia::core::audio::AudioBufferRef::F64(buf) => buf.as_ref().capacity(),
                _ => return Err(AudioError::UnsupportedFormat("Unsupported audio sample format".to_string())),
            };

            // For now, create silent samples as placeholder
            // TODO: Implement proper sample conversion from Symphonia buffers
            for _ in 0..num_samples {
                samples.push(0.0);
            }

            if samples.is_empty() {
                continue;
            }

            let frame = AudioFrame::new(
                samples,
                self.sample_rate,
                self.channels,
                self.frame_count,
            )?;

            self.frame_count += 1;

            return Ok(Some(frame));
        }
    }

    fn is_finished(&self) -> bool {
        self.finished
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_invalid_file() {
        let result = SymphoniaDecoder::from_file("/nonexistent/file.mp3");
        assert!(result.is_err());
    }
}
