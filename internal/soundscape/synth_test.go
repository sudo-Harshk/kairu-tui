package soundscape

import (
	"testing"
)

func TestIsSyntheticTrack(t *testing.T) {
	t.Parallel()

	cases := []struct {
		track string
		want  bool
	}{
		{"[Synth] White Noise", true},
		{"[Synth] Pink Noise", true},
		{"[Synth] Brown Noise", true},
		{"[Synth] Focus Binaural Beats", true},
		{"rain.mp3", false},
		{"focus.wav", false},
		{"", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.track, func(t *testing.T) {
			t.Parallel()
			got := IsSyntheticTrack(tc.track)
			if got != tc.want {
				t.Fatalf("IsSyntheticTrack(%q) = %t, want %t", tc.track, got, tc.want)
			}
		})
	}
}

func TestStreamers(t *testing.T) {
	t.Parallel()

	t.Run("WhiteNoise", func(t *testing.T) {
		s := &WhiteNoiseStreamer{baseVolume: 1.0}
		samples := make([][2]float64, 100)
		n, ok := s.Stream(samples)
		if !ok || n != len(samples) {
			t.Fatalf("expected to stream %d samples, got %d", len(samples), n)
		}
		for i, sample := range samples {
			if sample[0] < -1.0 || sample[0] > 1.0 || sample[1] < -1.0 || sample[1] > 1.0 {
				t.Fatalf("sample %d is out of bounds: %v", i, sample)
			}
		}
	})

	t.Run("BrownNoise", func(t *testing.T) {
		s := &BrownNoiseStreamer{baseVolume: 1.0}
		samples := make([][2]float64, 100)
		n, ok := s.Stream(samples)
		if !ok || n != len(samples) {
			t.Fatalf("expected to stream %d samples, got %d", len(samples), n)
		}
		for i, sample := range samples {
			if sample[0] < -1.0 || sample[0] > 1.0 || sample[1] < -1.0 || sample[1] > 1.0 {
				t.Fatalf("sample %d is out of bounds: %v", i, sample)
			}
		}
	})

	t.Run("PinkNoise", func(t *testing.T) {
		s := &PinkNoiseStreamer{baseVolume: 1.0}
		samples := make([][2]float64, 100)
		n, ok := s.Stream(samples)
		if !ok || n != len(samples) {
			t.Fatalf("expected to stream %d samples, got %d", len(samples), n)
		}
		for i, sample := range samples {
			if sample[0] < -1.0 || sample[0] > 1.0 || sample[1] < -1.0 || sample[1] > 1.0 {
				t.Fatalf("sample %d is out of bounds: %v", i, sample)
			}
		}
	})

	t.Run("BinauralBeat", func(t *testing.T) {
		s := &BinauralBeatStreamer{
			freqL:      115.0,
			freqR:      125.0,
			sampleRate: 44100.0,
			baseVolume: 1.0,
		}
		samples := make([][2]float64, 100)
		n, ok := s.Stream(samples)
		if !ok || n != len(samples) {
			t.Fatalf("expected to stream %d samples, got %d", len(samples), n)
		}
		for i, sample := range samples {
			if sample[0] < -1.0 || sample[0] > 1.0 || sample[1] < -1.0 || sample[1] > 1.0 {
				t.Fatalf("sample %d is out of bounds: %v", i, sample)
			}
		}
	})
}

func TestBinauralPresets(t *testing.T) {
	t.Parallel()

	cases := []struct {
		preset    string
		carrier   float64
		beat      float64
		wantLeft  float64
		wantRight float64
	}{
		{preset: "alpha", wantLeft: 115.0, wantRight: 125.0},
		{preset: "beta", wantLeft: 140.0, wantRight: 160.0},
		{preset: "theta", wantLeft: 97.0, wantRight: 103.0},
		{preset: "delta", wantLeft: 68.5, wantRight: 71.5},
		{preset: "custom", carrier: 100.0, beat: 10.0, wantLeft: 95.0, wantRight: 105.0},
		{preset: "invalid", wantLeft: 115.0, wantRight: 125.0}, // defaults to alpha
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.preset, func(t *testing.T) {
			t.Parallel()
			gotLeft, gotRight := getBinauralFrequencies(tc.preset, tc.carrier, tc.beat)
			if gotLeft != tc.wantLeft || gotRight != tc.wantRight {
				t.Fatalf("for %s, got (%f, %f), want (%f, %f)", tc.preset, gotLeft, gotRight, tc.wantLeft, tc.wantRight)
			}
		})
	}
}
