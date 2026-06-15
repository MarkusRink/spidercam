package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/markus/spidercam/internal/audio"
	"github.com/markus/spidercam/internal/capture"
	"github.com/markus/spidercam/internal/fixtures"
	"github.com/markus/spidercam/internal/output"
	"github.com/markus/spidercam/internal/preview"
	"github.com/markus/spidercam/internal/protocol"
	"github.com/markus/spidercam/internal/room"
	"github.com/markus/spidercam/internal/scenario"
	"github.com/markus/spidercam/internal/signaling"
	"github.com/markus/spidercam/internal/webrtc"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, cfg Config) error {
	printBanner(cfg)

	rm := room.New(cfg.ParticipantURL)
	if cfg.Mock {
		if state, err := fixtures.LoadRoutingState(); err == nil {
			rm.SetState(state)
		}
		if _, _, err := signaling.EnsureDefaultCaptureSelection(rm, true); err != nil {
			log.Printf("capture defaults: %v", err)
		}
	} else {
		room.ApplyBootstrapIdle(rm)
		if _, _, err := signaling.EnsureDefaultCaptureSelection(rm, false); err != nil {
			log.Printf("capture defaults: %v", err)
		}
	}

	engine := scenario.New(rm)
	engine.SetAudioDriven(true)
	webrtcHub := webrtc.NewHub(rm)
	hostHub := signaling.NewHostHub(rm, cfg.Mock)
	participantHub := signaling.NewParticipantHub(rm, webrtcHub)

	keyframe, err := fixtures.LoadPreviewKeyframe()
	if err != nil {
		return fmt.Errorf("preview fixture: %w", err)
	}
	previewStream, err := preview.New(preview.Config{
		Width:        preview.DefaultWidth,
		Height:       preview.DefaultHeight,
		FPS:          preview.DefaultFPS,
		BitrateKbps:  preview.DefaultBitrateKbps,
		Mock:         cfg.Mock,
		MockKeyframe: keyframe,
	})
	if err != nil {
		return fmt.Errorf("preview stream: %w", err)
	}
	defer previewStream.Close()
	previewHub := signaling.NewPreviewHub(previewStream)

	engine.OnState(hostHub.BroadcastState)
	engine.OnSelectionChange(participantHub.ScheduleBroadcast)

	var audioEngine *audio.Engine
	var outWriter output.Writer
	if cfg.Mock {
		audioEngine, _, outWriter = audio.SetupMockAudio(ctx, rm, engine)
		hostHub.SetStreamProcessor(audioEngine)
	} else {
		outCfg := output.DefaultConfig()
		type openResult struct {
			out output.Writer
			err error
		}
		openCh := make(chan openResult, 1)
		go func() {
			out, err := output.Open(ctx, outCfg)
			openCh <- openResult{out: out, err: err}
		}()
		var out output.Writer
		var err error
		select {
		case res := <-openCh:
			out, err = res.out, res.err
		case <-time.After(3 * time.Second):
			err = errors.New("timed out opening virtual output (v4l2loopback/pulseaudio)")
		}
		if err != nil {
			log.Printf("virtual output unavailable: %v", err)
			fallback := output.NewMockWriter()
			fallback.SetHealthy(false)
			outWriter = fallback
		} else {
			outWriter = out
			defer func() { _ = out.Close() }()
		}
		rm.UpdateState(func(s *protocol.RoomState) {
			s.OutputHealthy = outWriter.Healthy()
		})
	}

	var captureReopener signaling.CaptureReopener
	var previewCamera preview.CameraReader
	if !cfg.Mock {
		capState := rm.State().Capture
		bundle, err := capture.Open(ctx, capture.Selection{
			MicID:    capState.MicID,
			CameraID: capState.CameraID,
			SinkID:   capState.SinkID,
		}, capture.DefaultSampleRate)
		if err != nil {
			log.Printf("capture open: %v", err)
		} else {
			previewCamera = bundle
			captureReopener = &signaling.CaptureBundleReopener{Bundle: bundle}
			hostHub.SetCaptureReopener(captureReopener)
			audioEngine = audio.SetupProductionAudio(ctx, rm, engine, bundle, outWriter)
			hostHub.SetStreamProcessor(audioEngine)
			defer func() { _ = bundle.Close() }()
		}
	}

	if cfg.Mock {
		log.Print("mock mode enabled (capture/output stubbed)")
	} else if !capture.NativeEnumeration {
		log.Print("warning: device list uses mock I/O stubs; install libpipewire-0.3-dev and run make build for real hardware")
	}

	engine.Start(ctx)
	go previewHub.PublishLoop(ctx)
	onPreviewCut := func(cut bool, sel *protocol.SelectionState) {
		if !cut || sel == nil {
			return
		}
		previewHub.BroadcastCut(protocol.PreviewCutMsg{
			Type:          "preview-cut",
			ActiveVideoID: sel.ActiveVideoID,
			Seq:           int(previewStream.Seq()),
		})
	}
	if cfg.Mock {
		preview.RunMockCompositor(ctx, previewStream, rm, onPreviewCut)
	} else {
		preview.RunCompositor(ctx, previewStream, rm, previewCamera, outWriter, onPreviewCut)
	}

	hostSrv := &http.Server{
		Addr: cfg.HostHTTPAddr,
		Handler: signaling.NewHostMux(hostFS, hostUIRoot, rm, signaling.HostServices{
			Hub:               hostHub,
			PreviewHub:        previewHub,
			StreamProcessor:   audioEngine,
			UseFixtureDevices: cfg.Mock,
			CaptureReopener:   captureReopener,
		}),
	}
	participantSrv := &http.Server{
		Addr: cfg.ParticipantHTTPAddr,
		Handler: signaling.NewParticipantMux(participantFS, participantUIRoot, signaling.ParticipantServices{
			Hub: participantHub,
		}),
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Printf("host UI listening on %s", cfg.HostHTTPAddr)
		err := hostSrv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("host server: %w", err)
	})

	g.Go(func() error {
		log.Printf("participant UI listening on %s", cfg.ParticipantHTTPAddr)
		err := participantSrv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("participant server: %w", err)
	})

	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var errs []error
		if err := hostSrv.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, err)
		}
		if err := participantSrv.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	})

	printReady(cfg)

	if cfg.OpenBrowser {
		go openBrowser(hostURL(cfg.HostHTTPAddr))
	}

	return g.Wait()
}

func printBanner(cfg Config) {
	fmt.Printf("spidercamd %s\n", Version)
}

func printReady(cfg Config) {
	fmt.Printf("  host UI:        %s\n", hostURL(cfg.HostHTTPAddr))
	fmt.Printf("  participant UI: %s\n", cfg.ParticipantURL)
	if cfg.Mock {
		fmt.Printf("  mode:           mock\n")
	}
}
