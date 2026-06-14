package preview

func scaleRGBA(dst []byte, dstW, dstH int, src []byte, srcW, srcH int) {
	if dstW <= 0 || dstH <= 0 || srcW <= 0 || srcH <= 0 {
		return
	}
	for dy := 0; dy < dstH; dy++ {
		sy := dy * srcH / dstH
		rowOff := sy * srcW * 4
		dstRow := dy * dstW * 4
		for dx := 0; dx < dstW; dx++ {
			sx := dx * srcW / dstW
			si := rowOff + sx*4
			di := dstRow + dx*4
			dst[di] = src[si]
			dst[di+1] = src[si+1]
			dst[di+2] = src[si+2]
			dst[di+3] = src[si+3]
		}
	}
}
