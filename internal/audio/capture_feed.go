package audio

type CaptureFeed interface {
	ReadMic(buf []float32) int
	ReadMonitor(buf []float32) int
}
