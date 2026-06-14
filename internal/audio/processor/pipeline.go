package processor

import (
	"time"

	"github.com/markus/spidercam/internal/audio/aec"
	"github.com/markus/spidercam/internal/audio/enhance"
	audiomath "github.com/markus/spidercam/internal/audio/math"
	"github.com/markus/spidercam/internal/audio/jitter"
	"github.com/markus/spidercam/internal/protocol"
)

type State struct {
	NoiseFloorDbfs   float64
	SpeechLevelDbfs  float64
	CalibrationGain  float64
	CalibrationPhase string
	Vad              bool
	VadHangoverUntil time.Time
	ScoreSmooth      float64
	GateOpen         bool
	DuckingGainDb    float64

	AecEnabled     bool
	DenoiseEnabled bool
	AecUs          float64
	DenoiseUs      float64
}

type Pipeline struct {
	state     State
	aec       aec.Processor
	denoiser  enhance.Denoiser
	jitter    *jitter.Buffer
	lastFrame []float32
	role      protocol.StreamRole
	id        string
	name      string
	hpfPrevIn float64
	hpfPrevOut float64
	workBuf   []float32
}

func NewPipeline(id, name string, role protocol.StreamRole) *Pipeline {
	return &Pipeline{
		id:     id,
		name:   name,
		role:   role,
		jitter: jitter.NewBuffer(5),
		state: State{
			NoiseFloorDbfs:   -60,
			CalibrationGain:  1,
			CalibrationPhase: "idle",
		},
		lastFrame: make([]float32, audiomath.FrameSamples),
		workBuf:   make([]float32, audiomath.FrameSamples),
	}
}

func (p *Pipeline) ID() string { return p.id }
func (p *Pipeline) Jitter() *jitter.Buffer { return p.jitter }
func (p *Pipeline) State() State { return p.state }
func (p *Pipeline) LastFrame() []float32 { return p.lastFrame }
func (p *Pipeline) Role() protocol.StreamRole { return p.role }

func (p *Pipeline) SetProcessing(flags protocol.StreamProcessingFlags) error {
	if flags.AecEnabled != p.state.AecEnabled {
		p.state.AecEnabled = flags.AecEnabled
		if flags.AecEnabled {
			proc, err := aec.New(aec.Config{Enabled: true})
			if err != nil {
				return err
			}
			p.aec = proc
		} else {
			p.aec = nil
			p.state.AecUs = 0
		}
	}
	if flags.DenoiseEnabled != p.state.DenoiseEnabled {
		p.state.DenoiseEnabled = flags.DenoiseEnabled
		if flags.DenoiseEnabled {
			proc, err := enhance.New()
			if err != nil {
				return err
			}
			p.denoiser = proc
		} else {
			p.denoiser = nil
			p.state.DenoiseUs = 0
		}
	}
	return nil
}

func (p *Pipeline) SetDuckingGainDb(db float64) {
	p.state.DuckingGainDb = db
}

func (p *Pipeline) Process(
	raw []float32,
	ref []float32,
	echoPenalty float64,
	cfg protocol.HostConfig,
) protocol.StreamMetrics {
	if len(raw) != audiomath.FrameSamples {
		copy(p.workBuf, raw)
		if len(raw) < audiomath.FrameSamples {
			for i := len(raw); i < audiomath.FrameSamples; i++ {
				p.workBuf[i] = 0
			}
		}
		raw = p.workBuf[:audiomath.FrameSamples]
	} else {
		copy(p.workBuf, raw)
		raw = p.workBuf
	}

	highPass(raw, &p.hpfPrevIn, &p.hpfPrevOut)
	applyGain(raw, p.state.CalibrationGain)

	rmsRaw := audiomath.RmsDbfs(raw)
	p.updateNoiseFloor(rmsRaw, p.state.Vad)
	noiseLinear := audiomath.DbToLinear(p.state.NoiseFloorDbfs)
	rmsLinear := audiomath.DbToLinear(rmsRaw)
	snr := audiomath.SnrDb(rmsLinear, noiseLinear)
	p.state.Vad = updateVAD(snr, &p.state, cfg)
	if p.state.Vad {
		p.state.SpeechLevelDbfs = audiomath.EMA(p.state.SpeechLevelDbfs, rmsRaw, 0.05)
		p.updateCalibration(cfg)
	}

	components := audiomath.BuildScoreComponents(audiomath.ScoreInput{
		RmsDbfs: rmsRaw, SnrDb: snr, Vad: p.state.Vad,
		Role: p.role, EchoPenalty: echoPenalty,
		HostPriority: cfg.HostPriority,
	})
	score := audiomath.FrameScore(components, cfg.ScoreWeights)
	p.state.ScoreSmooth = audiomath.EMA(p.state.ScoreSmooth, score, cfg.ScoreSmoothingAlpha)

	enhanced := make([]float32, len(raw))
	copy(enhanced, raw)

	if p.state.AecEnabled && p.aec != nil {
		p.aec.ProcessReverse(ref)
		enhanced = p.aec.ProcessNear(enhanced)
		st := p.aec.Stats()
		p.state.AecUs = audiomath.EMA(p.state.AecUs, st.ProcessUs, 0.05)
	} else {
		p.state.AecUs = 0
	}
	if p.state.DenoiseEnabled && p.denoiser != nil {
		var us float64
		enhanced, us = p.denoiser.Process(enhanced)
		p.state.DenoiseUs = audiomath.EMA(p.state.DenoiseUs, us, 0.05)
	} else {
		p.state.DenoiseUs = 0
	}

	outRms := audiomath.RmsDbfs(enhanced)
	outPeak := audiomath.PeakDbfs(enhanced)

	gateDb := gate(&p.state, rmsRaw, cfg)
	copy(p.lastFrame, enhanced)
	applyGainDb(p.lastFrame, gateDb+p.state.DuckingGainDb)

	now := time.Now()
	hangover := 0
	if p.state.Vad && now.Before(p.state.VadHangoverUntil) {
		hangover = int(p.state.VadHangoverUntil.Sub(now).Milliseconds())
		if hangover < 0 {
			hangover = 0
		}
	}

	return protocol.StreamMetrics{
		ParticipantID:   p.id,
		Name:            p.name,
		Role:            p.role,
		RmsDbfs:         outRms,
		PeakDbfs:        outPeak,
		SpeechLevelDbfs: p.state.SpeechLevelDbfs,
		NoiseFloorDbfs:  p.state.NoiseFloorDbfs,
		SnrDb:           snr,
		Vad:             p.state.Vad,
		VadHangoverMs:   hangover,
		Score:           score,
		ScoreSmooth:     p.state.ScoreSmooth,
		ScoreComponents: components,
		GateGainDb:      gateDb,
		DuckingGainDb:   p.state.DuckingGainDb,
		CalibrationGain: p.state.CalibrationGain,
		CalibrationPhase: p.state.CalibrationPhase,
		JitterBufferFrames: p.jitter.Depth(),
		VideoActive:     true,
		AudioActive:     true,
		LastUpdated:     now.UnixMilli(),
		AecEnabled:      p.state.AecEnabled,
		DenoiseEnabled:  p.state.DenoiseEnabled,
		AecUs:           p.state.AecUs,
		DenoiseUs:       p.state.DenoiseUs,
	}
}

func (p *Pipeline) updateNoiseFloor(rmsDbfs float64, vad bool) {
	if !vad {
		p.state.NoiseFloorDbfs = audiomath.EMA(p.state.NoiseFloorDbfs, rmsDbfs, 0.02)
	}
}

func (p *Pipeline) updateCalibration(cfg protocol.HostConfig) {
	if p.state.SpeechLevelDbfs == 0 {
		return
	}
	target := cfg.TargetSpeechDbfs
	deltaDb := target - p.state.SpeechLevelDbfs
	clampMin, clampMax := cfg.CalibrationGainClampDb[0], cfg.CalibrationGainClampDb[1]
	if deltaDb < clampMin {
		deltaDb = clampMin
	}
	if deltaDb > clampMax {
		deltaDb = clampMax
	}
	targetGain := audiomath.DbToLinear(deltaDb)
	p.state.CalibrationGain = audiomath.EMA(p.state.CalibrationGain, targetGain, 0.01)
	if p.state.CalibrationPhase == "idle" {
		p.state.CalibrationPhase = "applying"
	}
}

func updateVAD(snr float64, st *State, cfg protocol.HostConfig) bool {
	now := time.Now()
	if st.Vad {
		if snr < cfg.VadSnrOffDb {
			if now.After(st.VadHangoverUntil) {
				st.Vad = false
			}
		} else {
			st.VadHangoverUntil = now.Add(time.Duration(cfg.VadHangoverMs) * time.Millisecond)
		}
	} else if snr > cfg.VadSnrOnDb {
		st.Vad = true
		st.VadHangoverUntil = now.Add(time.Duration(cfg.VadHangoverMs) * time.Millisecond)
	}
	return st.Vad
}

func gate(st *State, rmsDbfs float64, cfg protocol.HostConfig) float64 {
	if st.Vad || rmsDbfs > cfg.ReferenceVadOffDbfs {
		st.GateOpen = true
		return 0
	}
	st.GateOpen = false
	return -cfg.GateAttenuationDb
}

func highPass(samples []float32, prevIn, prevOut *float64) {
	const fc = 100.0
	alpha := 1.0 / (1.0 + 2*3.141592653589793*fc/float64(audiomath.SampleRate))
	for i, s := range samples {
		x := float64(s)
		y := alpha * (*prevOut + x - *prevIn)
		samples[i] = float32(y)
		*prevIn = x
		*prevOut = y
	}
}

func applyGain(samples []float32, gain float64) {
	for i := range samples {
		samples[i] = float32(float64(samples[i]) * gain)
	}
}

func applyGainDb(samples []float32, db float64) {
	if db == 0 {
		return
	}
	gain := audiomath.DbToLinear(db)
	applyGain(samples, gain)
}
