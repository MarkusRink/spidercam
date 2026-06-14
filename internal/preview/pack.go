package preview

import "encoding/binary"

const chunkHeaderSize = 13

const flagKeyframe byte = 0x01

func PackChunk(avcc []byte, pts uint64, keyframe bool) []byte {
	flags := byte(0)
	if keyframe {
		flags = flagKeyframe
	}
	out := make([]byte, chunkHeaderSize+len(avcc))
	out[0] = flags
	binary.BigEndian.PutUint64(out[1:9], pts)
	binary.BigEndian.PutUint32(out[9:13], uint32(len(avcc)))
	copy(out[chunkHeaderSize:], avcc)
	return out
}

func AnnexBToAVCC(data []byte) []byte {
	var nals [][]byte
	i := 0
	for i < len(data) {
		if i+3 < len(data) && data[i] == 0 && data[i+1] == 0 && data[i+2] == 1 {
			i += 3
		} else if i+4 < len(data) && data[i] == 0 && data[i+1] == 0 && data[i+2] == 0 && data[i+3] == 1 {
			i += 4
		} else {
			i++
			continue
		}
		start := i
		for i < len(data) {
			if (i+3 < len(data) && data[i] == 0 && data[i+1] == 0 && data[i+2] == 1) ||
				(i+4 < len(data) && data[i] == 0 && data[i+1] == 0 && data[i+2] == 0 && data[i+3] == 1) {
				break
			}
			i++
		}
		if i > start {
			nals = append(nals, data[start:i])
		}
	}

	total := 0
	for _, nal := range nals {
		total += 4 + len(nal)
	}
	out := make([]byte, total)
	off := 0
	for _, nal := range nals {
		binary.BigEndian.PutUint32(out[off:], uint32(len(nal)))
		off += 4
		copy(out[off:], nal)
		off += len(nal)
	}
	return out
}
