//go:build linux

package output

import (
	"errors"
	"fmt"
	"sync"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
)

type pulseSink struct {
	client      *pulse.Client
	stream      *pulse.PlaybackStream
	queue       *sampleQueue
	healthy     bool
	moduleIndex uint32
	createdSink bool
}

type sampleQueue struct {
	mu     sync.Mutex
	buf    []float32
	closed bool
	wait   sync.Cond
}

func newSampleQueue() *sampleQueue {
	q := &sampleQueue{}
	q.wait.L = &q.mu
	return q
}

func (q *sampleQueue) push(samples []float32) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.buf = append(q.buf, samples...)
	q.wait.Signal()
}

func (q *sampleQueue) read(out []float32) (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed && len(q.buf) == 0 {
		return 0, pulse.EndOfData
	}
	if len(q.buf) == 0 {
		for i := range out {
			out[i] = 0
		}
		return len(out), nil
	}
	n := copy(out, q.buf)
	q.buf = q.buf[n:]
	return n, nil
}

func (q *sampleQueue) close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.wait.Broadcast()
}

func openPulseSink(name string) (*pulseSink, error) {
	client, err := pulse.NewClient()
	if err != nil {
		return nil, fmt.Errorf("pulse client: %w", err)
	}

	sink, moduleIndex, created, err := ensurePulseSink(client, name)
	if err != nil {
		client.Close()
		return nil, err
	}

	queue := newSampleQueue()
	stream, err := client.NewPlayback(
		pulse.Float32Reader(queue.read),
		pulse.PlaybackMono,
		pulse.PlaybackSampleRate(48000),
		pulse.PlaybackSink(sink),
		pulse.PlaybackLatency(0.1),
		pulse.PlaybackMediaName("spidercam"),
	)
	if err != nil {
		if created {
			_ = client.RawRequest(&proto.UnloadModule{ModuleIndex: moduleIndex}, nil)
		}
		client.Close()
		return nil, fmt.Errorf("playback: %w", err)
	}

	stream.Start()

	return &pulseSink{
		client:      client,
		stream:      stream,
		queue:       queue,
		healthy:     true,
		moduleIndex: moduleIndex,
		createdSink: created,
	}, nil
}

func ensurePulseSink(client *pulse.Client, name string) (*pulse.Sink, uint32, bool, error) {
	sink, err := client.SinkByID(name)
	if err == nil {
		return sink, 0, false, nil
	}

	var reply proto.LoadModuleReply
	if err := client.RawRequest(&proto.LoadModule{
		Name: "module-null-sink",
		Args: NullSinkModuleArgs(name),
	}, &reply); err != nil {
		return nil, 0, false, fmt.Errorf("create sink %q: %w", name, err)
	}

	sink, err = client.SinkByID(name)
	if err != nil {
		_ = client.RawRequest(&proto.UnloadModule{ModuleIndex: reply.ModuleIndex}, nil)
		return nil, 0, false, fmt.Errorf("sink %q after create: %w", name, err)
	}
	return sink, reply.ModuleIndex, true, nil
}

func (p *pulseSink) WritePCM(samples []float32) error {
	if !p.healthy {
		return errors.New("pulse output unhealthy")
	}
	p.queue.push(samples)
	if err := p.stream.Error(); err != nil {
		p.healthy = false
		return err
	}
	return nil
}

func (p *pulseSink) Healthy() bool {
	return p.healthy && p.stream.Error() == nil
}

func (p *pulseSink) Close() error {
	p.queue.close()
	if p.stream != nil {
		p.stream.Close()
	}
	if p.createdSink && p.moduleIndex != 0 && p.client != nil {
		_ = p.client.RawRequest(&proto.UnloadModule{ModuleIndex: p.moduleIndex}, nil)
	}
	if p.client != nil {
		p.client.Close()
	}
	return nil
}
