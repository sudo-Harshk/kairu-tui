package soundscape

import (
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"kairu-tui/internal/config"
)

var (
	sr          = beep.SampleRate(44100)
	speakerInit sync.Once
	audioMutex  sync.Mutex
	activeCtrl  *beep.Ctrl
	activeTrack string

	// Synth volume controls (0.0 to 1.0)
	synthVolume      = 0.5
	volumeFadeTicker *time.Ticker
	fadeDone         chan bool
)

// InitSpeaker initializes the Beep speaker once.
func InitSpeaker() error {
	var err error
	speakerInit.Do(func() {
		// Init speaker with 44100Hz and buffer size 2048 (approx 46ms latency)
		err = speaker.Init(sr, 2048)
	})
	return err
}

// WhiteNoiseStreamer generates white noise.
type WhiteNoiseStreamer struct {
	baseVolume float64
}

func (w *WhiteNoiseStreamer) Stream(samples [][2]float64) (int, bool) {
	audioMutex.Lock()
	vol := w.baseVolume * synthVolume
	audioMutex.Unlock()

	for i := range samples {
		val := (rand.Float64()*2 - 1) * vol
		samples[i][0] = val
		samples[i][1] = val
	}
	return len(samples), true
}

func (w *WhiteNoiseStreamer) Err() error {
	return nil
}

// BrownNoiseStreamer generates brown noise using a leaky integrator.
type BrownNoiseStreamer struct {
	lastVal    float64
	baseVolume float64
}

func (b *BrownNoiseStreamer) Stream(samples [][2]float64) (int, bool) {
	audioMutex.Lock()
	vol := b.baseVolume * synthVolume
	audioMutex.Unlock()

	for i := range samples {
		r := rand.Float64()*2 - 1
		// Leaky integrator filter to create 1/f^2 spectrum
		b.lastVal = 0.98*b.lastVal + 0.02*r
		val := b.lastVal * 5.0 * vol // scale up since filtering attenuates signal
		if val > 1.0 {
			val = 1.0
		} else if val < -1.0 {
			val = -1.0
		}
		samples[i][0] = val
		samples[i][1] = val
	}
	return len(samples), true
}

func (b *BrownNoiseStreamer) Err() error {
	return nil
}

// PinkNoiseStreamer generates pink noise using Voss-McCartney algorithm.
type PinkNoiseStreamer struct {
	rows       [16]float64
	sum        float64
	baseVolume float64
}

func (p *PinkNoiseStreamer) Stream(samples [][2]float64) (int, bool) {
	audioMutex.Lock()
	vol := p.baseVolume * synthVolume
	audioMutex.Unlock()

	for i := range samples {
		// Select a random index and update it
		idx := rand.Intn(16)
		p.sum -= p.rows[idx]
		p.rows[idx] = rand.Float64()*2 - 1
		p.sum += p.rows[idx]
		// Voss pink noise value
		val := (p.sum + rand.Float64()*2 - 1) / 17.0 * vol
		if val > 1.0 {
			val = 1.0
		} else if val < -1.0 {
			val = -1.0
		}
		samples[i][0] = val
		samples[i][1] = val
	}
	return len(samples), true
}

func (p *PinkNoiseStreamer) Err() error {
	return nil
}

// BinauralBeatStreamer generates slightly detuned sine waves in left & right ears.
type BinauralBeatStreamer struct {
	freqL      float64
	freqR      float64
	sampleRate float64
	phaseL     float64
	phaseR     float64
	baseVolume float64
}

func (b *BinauralBeatStreamer) Stream(samples [][2]float64) (int, bool) {
	audioMutex.Lock()
	vol := b.baseVolume * synthVolume
	freqL := b.freqL
	freqR := b.freqR
	audioMutex.Unlock()

	stepL := 2.0 * math.Pi * freqL / b.sampleRate
	stepR := 2.0 * math.Pi * freqR / b.sampleRate
	for i := range samples {
		valL := math.Sin(b.phaseL) * vol
		valR := math.Sin(b.phaseR) * vol

		samples[i][0] = valL
		samples[i][1] = valR

		b.phaseL += stepL
		b.phaseR += stepR

		if b.phaseL > 2.0*math.Pi {
			b.phaseL -= 2.0 * math.Pi
		}
		if b.phaseR > 2.0*math.Pi {
			b.phaseR -= 2.0 * math.Pi
		}
	}
	return len(samples), true
}

func (b *BinauralBeatStreamer) Err() error {
	return nil
}

// IsSyntheticTrack returns true if the track is generated natively.
func IsSyntheticTrack(track string) bool {
	return strings.HasPrefix(track, "[Synth]")
}

func getBinauralFrequencies(preset string, customCarrier, customBeat float64) (float64, float64) {
	var carrier, beat float64
	switch strings.ToLower(preset) {
	case "alpha":
		carrier, beat = 120.0, 10.0
	case "beta":
		carrier, beat = 150.0, 20.0
	case "theta":
		carrier, beat = 100.0, 6.0
	case "delta":
		carrier, beat = 70.0, 3.0
	case "custom":
		carrier, beat = customCarrier, customBeat
		if carrier <= 0 {
			carrier = 120.0
		}
		if beat <= 0 {
			beat = 10.0
		}
	default:
		carrier, beat = 120.0, 10.0
	}
	return carrier - beat/2.0, carrier + beat/2.0
}

func UpdateActiveBinauralFrequencies(cfg config.Config) {
	audioMutex.Lock()
	defer audioMutex.Unlock()

	if activeCtrl == nil {
		return
	}
	if streamer, ok := activeCtrl.Streamer.(*BinauralBeatStreamer); ok {
		freqL, freqR := getBinauralFrequencies(cfg.BinauralPreset, cfg.BinauralCarrier, cfg.BinauralBeat)
		streamer.freqL = freqL
		streamer.freqR = freqR
	}
}

// StartNativeSynth starts playing the selected synthetic soundscape.
func StartNativeSynth(trackName string, cfg config.Config) error {
	if err := InitSpeaker(); err != nil {
		return err
	}

	audioMutex.Lock()
	defer audioMutex.Unlock()

	// Stop active soundscape
	if activeCtrl != nil {
		activeCtrl.Paused = true
		activeCtrl = nil
	}

	var streamer beep.Streamer
	switch trackName {
	case "[Synth] White Noise":
		streamer = &WhiteNoiseStreamer{baseVolume: 0.15}
	case "[Synth] Pink Noise":
		streamer = &PinkNoiseStreamer{baseVolume: 0.20}
	case "[Synth] Brown Noise":
		streamer = &BrownNoiseStreamer{baseVolume: 0.30}
	case "[Synth] Focus Binaural Beats":
		freqL, freqR := getBinauralFrequencies(cfg.BinauralPreset, cfg.BinauralCarrier, cfg.BinauralBeat)
		streamer = &BinauralBeatStreamer{
			freqL:      freqL,
			freqR:      freqR,
			sampleRate: 44100.0,
			baseVolume: 0.25,
		}
	default:
		return nil
	}

	synthVolume = 0.0 // Start at silence to allow smooth fade-in
	ctrl := &beep.Ctrl{Streamer: streamer, Paused: false}
	activeCtrl = ctrl
	activeTrack = trackName

	speaker.Play(ctrl)
	return nil
}

// StopNativeSynth stops playing the native soundscape.
func StopNativeSynth() {
	audioMutex.Lock()
	defer audioMutex.Unlock()

	if activeCtrl != nil {
		activeCtrl.Paused = true
		activeCtrl = nil
		activeTrack = ""
	}
}

// PauseNativeSynth pauses or resumes the native soundscape.
func PauseNativeSynth(paused bool) {
	audioMutex.Lock()
	defer audioMutex.Unlock()

	if activeCtrl != nil {
		activeCtrl.Paused = paused
	}
}

// SetNativeVolume sets the native synthesizer volume dynamically.
func SetNativeVolume(vol float64) {
	audioMutex.Lock()
	defer audioMutex.Unlock()

	if vol < 0.0 {
		vol = 0.0
	} else if vol > 1.0 {
		vol = 1.0
	}
	synthVolume = vol
}

// FadeNativeVolume transitions volume smoothly to target over a duration.
func FadeNativeVolume(target float64, duration time.Duration) {
	audioMutex.Lock()
	if activeCtrl == nil {
		audioMutex.Unlock()
		return
	}
	audioMutex.Unlock()

	if volumeFadeTicker != nil {
		// Stop any active fading
		volumeFadeTicker.Stop()
		select {
		case fadeDone <- true:
		default:
		}
	}

	volumeFadeTicker = time.NewTicker(20 * time.Millisecond)
	fadeDone = make(chan bool, 1)

	go func() {
		steps := int(duration / (20 * time.Millisecond))
		if steps <= 0 {
			SetNativeVolume(target)
			return
		}

		audioMutex.Lock()
		startVol := synthVolume
		audioMutex.Unlock()

		diff := target - startVol

		for i := 0; i < steps; i++ {
			select {
			case <-volumeFadeTicker.C:
				audioMutex.Lock()
				// Use cubic smoothstep interpolation: t * t * (3 - 2 * t)
				t := float64(i+1) / float64(steps)
				v := t * t * (3.0 - 2.0 * t)
				synthVolume = startVol + diff * v
				if synthVolume < 0.0 {
					synthVolume = 0.0
				} else if synthVolume > 1.0 {
					synthVolume = 1.0
				}
				audioMutex.Unlock()
			case <-fadeDone:
				return
			}
		}

		// Ensure precise target value
		SetNativeVolume(target)
		volumeFadeTicker.Stop()
	}()
}
