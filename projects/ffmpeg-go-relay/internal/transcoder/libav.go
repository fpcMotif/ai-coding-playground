//go:build libav && cgo

package transcoder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asticode/go-astiav"

	"ffmpeg-go-relay/internal/config"
	"ffmpeg-go-relay/internal/logger"
)

const libavIOBufferSize = 4096

var libavLogOnce sync.Once

type libavBackend struct {
	writer *io.PipeWriter
	done   chan error
}

func newLibAVBackend(ctx context.Context, cfg config.TranscodeConfig, upstream string, log *logger.Logger) (Backend, error) {
	reader, writer := io.Pipe()
	backend := &libavBackend{
		writer: writer,
		done:   make(chan error, 1),
	}

	go func() {
		backend.done <- runLibAV(ctx, cfg, upstream, reader, log)
	}()

	return backend, nil
}

func (b *libavBackend) Write(p []byte) (int, error) {
	return b.writer.Write(p)
}

func (b *libavBackend) Close() error {
	_ = b.writer.Close()
	return <-b.done
}

type libavCleanup struct {
	fns []func()
}

func (c *libavCleanup) Add(fn func()) {
	c.fns = append(c.fns, fn)
}

func (c *libavCleanup) AddWithError(fn func() error) {
	c.Add(func() { _ = fn() })
}

func (c *libavCleanup) Close() {
	for i := len(c.fns) - 1; i >= 0; i-- {
		c.fns[i]()
	}
}

type streamMode int

const (
	streamModeCopy streamMode = iota
	streamModeTranscode
)

type libavStream struct {
	mode              streamMode
	inputStream       *astiav.Stream
	outputStream      *astiav.Stream
	decCodecContext   *astiav.CodecContext
	encCodecContext   *astiav.CodecContext
	buffersrcContext  *astiav.BuffersrcFilterContext
	buffersinkContext *astiav.BuffersinkFilterContext
	filterGraph       *astiav.FilterGraph
	decFrame          *astiav.Frame
	filterFrame       *astiav.Frame
	encPkt            *astiav.Packet
	decLastPTS        *int64
}

func runLibAV(ctx context.Context, cfg config.TranscodeConfig, upstream string, reader *io.PipeReader, log *logger.Logger) error {
	setupLibAVLogger(log)

	cleanup := &libavCleanup{}
	defer cleanup.Close()
	defer func() { _ = reader.Close() }()

	interrupter := astiav.NewIOInterrupter()
	cleanup.Add(interrupter.Free)
	go func() {
		<-ctx.Done()
		interrupter.Interrupt()
	}()

	inputFormatContext := astiav.AllocFormatContext()
	if inputFormatContext == nil {
		return errors.New("input format context is nil")
	}
	cleanup.Add(inputFormatContext.Free)

	inputIOContext, err := astiav.AllocIOContext(libavIOBufferSize, false, reader.Read, nil, nil)
	if err != nil {
		return fmt.Errorf("allocate input io context: %w", err)
	}
	cleanup.Add(inputIOContext.Free)

	inputFormatContext.SetPb(inputIOContext)
	inputFormatContext.SetIOInterrupter(interrupter)

	if err := inputFormatContext.OpenInput("", nil, nil); err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	cleanup.Add(inputFormatContext.CloseInput)

	if err := inputFormatContext.FindStreamInfo(nil); err != nil {
		return fmt.Errorf("find stream info: %w", err)
	}

	outputFormatContext, err := astiav.AllocOutputFormatContext(nil, "flv", upstream)
	if err != nil {
		return fmt.Errorf("allocate output format context: %w", err)
	}
	if outputFormatContext == nil {
		return errors.New("output format context is nil")
	}
	cleanup.Add(outputFormatContext.Free)
	outputFormatContext.SetIOInterrupter(interrupter)

	if !outputFormatContext.OutputFormat().Flags().Has(astiav.IOFormatFlagNofile) {
		outputIOContext, err := astiav.OpenIOContext(upstream, astiav.NewIOContextFlags(astiav.IOContextFlagWrite), interrupter, nil)
		if err != nil {
			return fmt.Errorf("open output io context: %w", err)
		}
		cleanup.AddWithError(outputIOContext.Close)
		outputFormatContext.SetPb(outputIOContext)
	}

	streams := map[int]*libavStream{}
	videoCodec := normalizeCodecName(cfg.VideoCodec, "libx264")
	audioCodec := normalizeCodecName(cfg.AudioCodec, "aac")

	for _, is := range inputFormatContext.Streams() {
		mediaType := is.CodecParameters().MediaType()
		if mediaType != astiav.MediaTypeAudio && mediaType != astiav.MediaTypeVideo {
			continue
		}

		codecName := videoCodec
		if mediaType == astiav.MediaTypeAudio {
			codecName = audioCodec
		}

		s := &libavStream{inputStream: is}
		if isCopyCodec(codecName) {
			s.mode = streamModeCopy
			outputStream := outputFormatContext.NewStream(nil)
			if outputStream == nil {
				return errors.New("output stream is nil")
			}
			if err := is.CodecParameters().Copy(outputStream.CodecParameters()); err != nil {
				return fmt.Errorf("copy codec parameters: %w", err)
			}
			outputStream.CodecParameters().SetCodecTag(0)
			outputStream.SetTimeBase(is.TimeBase())
			s.outputStream = outputStream
			streams[is.Index()] = s
			continue
		}

		s.mode = streamModeTranscode
		if err := initTranscodeStream(s, inputFormatContext, outputFormatContext, codecName, cfg, log, cleanup); err != nil {
			return err
		}
		streams[is.Index()] = s
	}

	if len(streams) == 0 {
		return errors.New("no audio or video streams found")
	}

	if err := outputFormatContext.WriteHeader(nil); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	pkt := astiav.AllocPacket()
	if pkt == nil {
		return errors.New("packet is nil")
	}
	cleanup.Add(pkt.Free)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := inputFormatContext.ReadFrame(pkt); err != nil {
			if errors.Is(err, astiav.ErrEof) {
				break
			}
			return fmt.Errorf("read frame: %w", err)
		}

		s, ok := streams[pkt.StreamIndex()]
		if !ok {
			pkt.Unref()
			continue
		}

		if s.mode == streamModeCopy {
			if err := writeCopyPacket(pkt, s, outputFormatContext); err != nil {
				return err
			}
			pkt.Unref()
			continue
		}

		if err := transcodePacket(pkt, s, outputFormatContext); err != nil {
			return err
		}
		pkt.Unref()
	}

	for _, s := range streams {
		if s.mode != streamModeTranscode {
			continue
		}
		if err := flushDecoder(s, outputFormatContext); err != nil {
			return err
		}
		if err := filterEncodeWriteFrame(nil, s, outputFormatContext); err != nil {
			return err
		}
		if err := encodeWriteFrame(nil, s, outputFormatContext); err != nil {
			return err
		}
	}

	if err := outputFormatContext.WriteTrailer(); err != nil {
		return fmt.Errorf("write trailer: %w", err)
	}

	return nil
}

func setupLibAVLogger(log *logger.Logger) {
	if log == nil {
		return
	}
	libavLogOnce.Do(func() {
		astiav.SetLogLevel(astiav.LogLevelError)
		astiav.SetLogCallback(func(c astiav.Classer, l astiav.LogLevel, format, msg string) {
			message := strings.TrimSpace(msg)
			if message == "" {
				return
			}
			log.Debug("libav log", "message", message, "level", int(l))
		})
	})
}

func normalizeCodecName(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func isCopyCodec(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "copy")
}

func initTranscodeStream(
	s *libavStream,
	inputFormatContext *astiav.FormatContext,
	outputFormatContext *astiav.FormatContext,
	codecName string,
	cfg config.TranscodeConfig,
	log *logger.Logger,
	cleanup *libavCleanup,
) error {
	if s.inputStream == nil {
		return errors.New("input stream is nil")
	}

	decCodec := astiav.FindDecoder(s.inputStream.CodecParameters().CodecID())
	if decCodec == nil {
		return errors.New("decoder codec is nil")
	}

	s.decCodecContext = astiav.AllocCodecContext(decCodec)
	if s.decCodecContext == nil {
		return errors.New("decoder codec context is nil")
	}
	cleanup.Add(s.decCodecContext.Free)

	if err := s.inputStream.CodecParameters().ToCodecContext(s.decCodecContext); err != nil {
		return fmt.Errorf("update decoder context: %w", err)
	}

	if s.inputStream.CodecParameters().MediaType() == astiav.MediaTypeVideo {
		s.decCodecContext.SetFramerate(inputFormatContext.GuessFrameRate(s.inputStream, nil))
	}

	if err := s.decCodecContext.Open(decCodec, nil); err != nil {
		return fmt.Errorf("open decoder: %w", err)
	}

	s.decCodecContext.SetTimeBase(s.inputStream.TimeBase())

	s.decFrame = astiav.AllocFrame()
	if s.decFrame == nil {
		return errors.New("decoder frame is nil")
	}
	cleanup.Add(s.decFrame.Free)

	encCodec := astiav.FindEncoderByName(codecName)
	if encCodec == nil {
		return fmt.Errorf("encoder codec %q is nil", codecName)
	}

	s.encCodecContext = astiav.AllocCodecContext(encCodec)
	if s.encCodecContext == nil {
		return errors.New("encoder codec context is nil")
	}
	cleanup.Add(s.encCodecContext.Free)

	if s.inputStream.CodecParameters().MediaType() == astiav.MediaTypeAudio {
		if layouts := encCodec.SupportedChannelLayouts(); len(layouts) > 0 {
			s.encCodecContext.SetChannelLayout(layouts[0])
		} else {
			s.encCodecContext.SetChannelLayout(s.decCodecContext.ChannelLayout())
		}
		s.encCodecContext.SetSampleRate(s.decCodecContext.SampleRate())
		if formats := encCodec.SupportedSampleFormats(); len(formats) > 0 {
			s.encCodecContext.SetSampleFormat(formats[0])
		} else {
			s.encCodecContext.SetSampleFormat(s.decCodecContext.SampleFormat())
		}
		s.encCodecContext.SetTimeBase(astiav.NewRational(1, s.encCodecContext.SampleRate()))
	} else {
		s.encCodecContext.SetHeight(s.decCodecContext.Height())
		s.encCodecContext.SetWidth(s.decCodecContext.Width())
		if formats := encCodec.SupportedPixelFormats(); len(formats) > 0 {
			s.encCodecContext.SetPixelFormat(formats[0])
		} else {
			s.encCodecContext.SetPixelFormat(s.decCodecContext.PixelFormat())
		}
		s.encCodecContext.SetSampleAspectRatio(s.decCodecContext.SampleAspectRatio())
		s.encCodecContext.SetTimeBase(s.decCodecContext.TimeBase())
		s.encCodecContext.SetFramerate(s.decCodecContext.Framerate())

		if gopSize := parseGop(cfg.GOP, s.decCodecContext.Framerate(), log); gopSize > 0 {
			s.encCodecContext.SetGopSize(gopSize)
		}
	}

	if outputFormatContext.OutputFormat().Flags().Has(astiav.IOFormatFlagGlobalheader) {
		s.encCodecContext.SetFlags(s.encCodecContext.Flags().Add(astiav.CodecContextFlagGlobalHeader))
	}

	options := encoderOptions(cfg, s.inputStream.CodecParameters().MediaType())
	if err := s.encCodecContext.Open(encCodec, options); err != nil {
		if options != nil {
			options.Free()
		}
		return fmt.Errorf("open encoder: %w", err)
	}
	if options != nil {
		options.Free()
	}

	s.outputStream = outputFormatContext.NewStream(nil)
	if s.outputStream == nil {
		return errors.New("output stream is nil")
	}

	if err := s.outputStream.CodecParameters().FromCodecContext(s.encCodecContext); err != nil {
		return fmt.Errorf("update output codec parameters: %w", err)
	}
	s.outputStream.SetTimeBase(s.encCodecContext.TimeBase())

	if err := initFilters(s, cleanup); err != nil {
		return err
	}

	return nil
}

func encoderOptions(cfg config.TranscodeConfig, mediaType astiav.MediaType) *astiav.Dictionary {
	if mediaType != astiav.MediaTypeVideo {
		return nil
	}

	var hasOptions bool
	options := astiav.NewDictionary()
	if cfg.Preset != "" {
		_ = options.Set("preset", cfg.Preset, astiav.NewDictionaryFlags())
		hasOptions = true
	}
	if cfg.CRF > 0 {
		_ = options.Set("crf", strconv.Itoa(cfg.CRF), astiav.NewDictionaryFlags())
		hasOptions = true
	}

	if !hasOptions {
		options.Free()
		return nil
	}

	return options
}

func parseGop(value string, frameRate astiav.Rational, log *logger.Logger) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	if gopSize, err := strconv.Atoi(trimmed); err == nil {
		return gopSize
	}
	if duration, err := time.ParseDuration(trimmed); err == nil {
		fps := frameRate.Float64()
		if fps <= 0 {
			if log != nil {
				log.Warn("unable to derive GOP from duration; missing frame rate", "gop", trimmed)
			}
			return 0
		}
		return int(math.Round(duration.Seconds() * fps))
	}
	if log != nil {
		log.Warn("unable to parse GOP setting", "gop", trimmed)
	}
	return 0
}

func initFilters(s *libavStream, cleanup *libavCleanup) error {
	s.filterGraph = astiav.AllocFilterGraph()
	if s.filterGraph == nil {
		return errors.New("filter graph is nil")
	}
	cleanup.Add(s.filterGraph.Free)

	outputs := astiav.AllocFilterInOut()
	if outputs == nil {
		return errors.New("filter outputs is nil")
	}
	cleanup.Add(outputs.Free)

	inputs := astiav.AllocFilterInOut()
	if inputs == nil {
		return errors.New("filter inputs is nil")
	}
	cleanup.Add(inputs.Free)

	buffersrcContextParameters := astiav.AllocBuffersrcFilterContextParameters()
	if buffersrcContextParameters == nil {
		return errors.New("buffersrc context parameters is nil")
	}
	defer buffersrcContextParameters.Free()

	var buffersrc *astiav.Filter
	var buffersink *astiav.Filter
	var content string
	if s.decCodecContext.MediaType() == astiav.MediaTypeAudio {
		buffersrc = astiav.FindFilterByName("abuffer")
		buffersrcContextParameters.SetChannelLayout(s.decCodecContext.ChannelLayout())
		buffersrcContextParameters.SetSampleFormat(s.decCodecContext.SampleFormat())
		buffersrcContextParameters.SetSampleRate(s.decCodecContext.SampleRate())
		buffersrcContextParameters.SetTimeBase(s.decCodecContext.TimeBase())
		buffersink = astiav.FindFilterByName("abuffersink")
		content = fmt.Sprintf(
			"aformat=sample_fmts=%s:channel_layouts=%s",
			s.encCodecContext.SampleFormat().Name(),
			s.encCodecContext.ChannelLayout().String(),
		)
	} else {
		buffersrc = astiav.FindFilterByName("buffer")
		buffersrcContextParameters.SetHeight(s.decCodecContext.Height())
		buffersrcContextParameters.SetPixelFormat(s.decCodecContext.PixelFormat())
		buffersrcContextParameters.SetSampleAspectRatio(s.decCodecContext.SampleAspectRatio())
		buffersrcContextParameters.SetTimeBase(s.inputStream.TimeBase())
		buffersrcContextParameters.SetWidth(s.decCodecContext.Width())
		buffersink = astiav.FindFilterByName("buffersink")
		content = fmt.Sprintf("format=pix_fmts=%s", s.encCodecContext.PixelFormat().Name())
	}

	if buffersrc == nil || buffersink == nil {
		return errors.New("required filters are nil")
	}

	var err error
	if s.buffersrcContext, err = s.filterGraph.NewBuffersrcFilterContext(buffersrc, "in"); err != nil {
		return fmt.Errorf("create buffersrc context: %w", err)
	}
	if s.buffersinkContext, err = s.filterGraph.NewBuffersinkFilterContext(buffersink, "out"); err != nil {
		return fmt.Errorf("create buffersink context: %w", err)
	}

	if err = s.buffersrcContext.SetParameters(buffersrcContextParameters); err != nil {
		return fmt.Errorf("set buffersrc parameters: %w", err)
	}
	if err = s.buffersrcContext.Initialize(nil); err != nil {
		return fmt.Errorf("initialize buffersrc context: %w", err)
	}

	outputs.SetName("in")
	outputs.SetFilterContext(s.buffersrcContext.FilterContext())
	outputs.SetPadIdx(0)
	outputs.SetNext(nil)

	inputs.SetName("out")
	inputs.SetFilterContext(s.buffersinkContext.FilterContext())
	inputs.SetPadIdx(0)
	inputs.SetNext(nil)

	if err = s.filterGraph.Parse(content, inputs, outputs); err != nil {
		return fmt.Errorf("parse filter graph: %w", err)
	}
	if err = s.filterGraph.Configure(); err != nil {
		return fmt.Errorf("configure filter graph: %w", err)
	}

	s.filterFrame = astiav.AllocFrame()
	if s.filterFrame == nil {
		return errors.New("filter frame is nil")
	}
	cleanup.Add(s.filterFrame.Free)

	s.encPkt = astiav.AllocPacket()
	if s.encPkt == nil {
		return errors.New("encoder packet is nil")
	}
	cleanup.Add(s.encPkt.Free)

	return nil
}

func writeCopyPacket(pkt *astiav.Packet, s *libavStream, outputFormatContext *astiav.FormatContext) error {
	pkt.SetStreamIndex(s.outputStream.Index())
	pkt.RescaleTs(s.inputStream.TimeBase(), s.outputStream.TimeBase())
	pkt.SetPos(-1)
	if err := outputFormatContext.WriteInterleavedFrame(pkt); err != nil {
		return fmt.Errorf("write packet: %w", err)
	}
	return nil
}

func transcodePacket(pkt *astiav.Packet, s *libavStream, outputFormatContext *astiav.FormatContext) error {
	pkt.RescaleTs(s.inputStream.TimeBase(), s.decCodecContext.TimeBase())
	if err := s.decCodecContext.SendPacket(pkt); err != nil {
		return fmt.Errorf("send packet: %w", err)
	}

	for {
		if err := s.decCodecContext.ReceiveFrame(s.decFrame); err != nil {
			if errors.Is(err, astiav.ErrEagain) || errors.Is(err, astiav.ErrEof) {
				return nil
			}
			return fmt.Errorf("receive frame: %w", err)
		}

		if s.decLastPTS != nil && *s.decLastPTS >= s.decFrame.Pts() {
			s.decFrame.Unref()
			continue
		}
		pts := s.decFrame.Pts()
		s.decLastPTS = &pts

		if err := filterEncodeWriteFrame(s.decFrame, s, outputFormatContext); err != nil {
			s.decFrame.Unref()
			return err
		}
		s.decFrame.Unref()
	}
}

func flushDecoder(s *libavStream, outputFormatContext *astiav.FormatContext) error {
	if err := s.decCodecContext.SendPacket(nil); err != nil {
		if !errors.Is(err, astiav.ErrEof) {
			return fmt.Errorf("flush decoder: %w", err)
		}
		return nil
	}

	for {
		if err := s.decCodecContext.ReceiveFrame(s.decFrame); err != nil {
			if errors.Is(err, astiav.ErrEagain) || errors.Is(err, astiav.ErrEof) {
				return nil
			}
			return fmt.Errorf("flush decoder frame: %w", err)
		}
		if err := filterEncodeWriteFrame(s.decFrame, s, outputFormatContext); err != nil {
			s.decFrame.Unref()
			return err
		}
		s.decFrame.Unref()
	}
}

func filterEncodeWriteFrame(f *astiav.Frame, s *libavStream, outputFormatContext *astiav.FormatContext) error {
	if err := s.buffersrcContext.AddFrame(f, astiav.NewBuffersrcFlags(astiav.BuffersrcFlagKeepRef)); err != nil {
		return fmt.Errorf("add frame to filter: %w", err)
	}

	for {
		if err := s.buffersinkContext.GetFrame(s.filterFrame, astiav.NewBuffersinkFlags()); err != nil {
			if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
				return nil
			}
			return fmt.Errorf("get filter frame: %w", err)
		}
		s.filterFrame.SetPictureType(astiav.PictureTypeNone)
		if err := encodeWriteFrame(s.filterFrame, s, outputFormatContext); err != nil {
			s.filterFrame.Unref()
			return err
		}
		s.filterFrame.Unref()
	}
}

func encodeWriteFrame(f *astiav.Frame, s *libavStream, outputFormatContext *astiav.FormatContext) error {
	if err := s.encCodecContext.SendFrame(f); err != nil {
		return fmt.Errorf("send frame: %w", err)
	}

	for {
		if err := s.encCodecContext.ReceivePacket(s.encPkt); err != nil {
			if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
				return nil
			}
			return fmt.Errorf("receive packet: %w", err)
		}
		s.encPkt.SetStreamIndex(s.outputStream.Index())
		s.encPkt.RescaleTs(s.encCodecContext.TimeBase(), s.outputStream.TimeBase())
		if err := outputFormatContext.WriteInterleavedFrame(s.encPkt); err != nil {
			s.encPkt.Unref()
			return fmt.Errorf("write packet: %w", err)
		}
		s.encPkt.Unref()
	}
}
