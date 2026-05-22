package main

import (
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
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

// initSpeaker initializes the Beep speaker once.
func initSpeaker() error {
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
	audioMutex.Unlock()

	stepL := 2.0 * math.Pi * b.freqL / b.sampleRate
	stepR := 2.0 * math.Pi * b.freqR / b.sampleRate
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

// startNativeSynth starts playing the selected synthetic soundscape.
func startNativeSynth(trackName string) error {
	if err := initSpeaker(); err != nil {
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
		streamer = &BinauralBeatStreamer{
			freqL:      115.0, // Carrier freq: 120Hz, beat: 10Hz (Alpha wave)
			freqR:      125.0,
			sampleRate: 44100.0,
			baseVolume: 0.25,
		}
	default:
		return nil
	}

	synthVolume = 0.5 // Reset volume to default base level
	ctrl := &beep.Ctrl{Streamer: streamer, Paused: false}
	activeCtrl = ctrl
	activeTrack = trackName

	speaker.Play(ctrl)
	return nil
}

// stopNativeSynth stops playing the native soundscape.
func stopNativeSynth() {
	audioMutex.Lock()
	defer audioMutex.Unlock()

	if activeCtrl != nil {
		activeCtrl.Paused = true
		activeCtrl = nil
		activeTrack = ""
	}
}

// pauseNativeSynth pauses or resumes the native soundscape.
func pauseNativeSynth(paused bool) {
	audioMutex.Lock()
	defer audioMutex.Unlock()

	if activeCtrl != nil {
		activeCtrl.Paused = paused
	}
}

// setNativeVolume sets the native synthesizer volume dynamically.
func setNativeVolume(vol float64) {
	audioMutex.Lock()
	defer audioMutex.Unlock()

	if vol < 0.0 {
		vol = 0.0
	} else if vol > 1.0 {
		vol = 1.0
	}
	synthVolume = vol
}

// fadeNativeVolume transitions volume smoothly to target over a duration.
func fadeNativeVolume(target float64, duration time.Duration) {
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
			setNativeVolume(target)
			return
		}

		audioMutex.Lock()
		startVol := synthVolume
		audioMutex.Unlock()

		diff := target - startVol
		stepVal := diff / float64(steps)

		for i := 0; i < steps; i++ {
			select {
			case <-volumeFadeTicker.C:
				audioMutex.Lock()
				synthVolume += stepVal
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
		setNativeVolume(target)
		volumeFadeTicker.Stop()
	}()
}
