#ifndef SP_RING_H
#define SP_RING_H

#include <stdatomic.h>
#include <stdint.h>

enum {
	SP_FRAME_SAMPLES = 480,
	SP_RING_FRAMES = 8,
	SP_RING_CAPACITY = SP_FRAME_SAMPLES * SP_RING_FRAMES,
};

typedef struct sp_ring {
	float buf[SP_RING_CAPACITY];
	atomic_uint write_pos;
	atomic_uint read_pos;
} sp_ring;

void sp_ring_init(sp_ring *r);
int sp_ring_write(sp_ring *r, const float *src, int samples);
int sp_ring_read(sp_ring *r, float *dst, int samples);
int sp_ring_avail(const sp_ring *r);

#endif
