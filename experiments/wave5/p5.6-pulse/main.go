package main

import (
	"fmt"
	"math"
	"os"

	"github.com/jfreymuth/pulse"
)

const (
	sampleRate = 48000
	frequency  = 440.0
	duration   = 3.0
	sinkName   = "spidercam_sink"
)

type toneGen struct {
	sample int
	total  int
}

func (g *toneGen) Read(out []float32) (int, error) {
	for i := range out {
		if g.sample >= g.total {
			if i == 0 {
				return 0, pulse.EndOfData
			}
			return i, nil
		}
		phase := float64(g.sample) * frequency / sampleRate
		out[i] = float32(math.Sin(2 * math.Pi * phase))
		g.sample++
	}
	return len(out), nil
}

func main() {
	client, err := pulse.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pulse client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	sink, err := client.SinkByID(sinkName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sink %q: %v\n", sinkName, err)
		os.Exit(1)
	}

	gen := &toneGen{total: int(duration * sampleRate)}
	stream, err := client.NewPlayback(
		pulse.Float32Reader(gen.Read),
		pulse.PlaybackMono,
		pulse.PlaybackSampleRate(sampleRate),
		pulse.PlaybackSink(sink),
		pulse.PlaybackLatency(0.1),
		pulse.PlaybackMediaName("spidercam-p5.6-tone"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "playback: %v\n", err)
		os.Exit(1)
	}
	defer stream.Close()

	stream.Start()
	stream.Drain()

	if err := stream.Error(); err != nil {
		fmt.Fprintf(os.Stderr, "stream: %v\n", err)
		os.Exit(1)
	}
	if stream.Underflow() {
		fmt.Fprintln(os.Stderr, "warning: playback underflow")
	}

	fmt.Printf("played %.0fs %gHz mono float32 @ %dHz to sink %q\n",
		duration, frequency, sampleRate, sinkName)
}
