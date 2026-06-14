#include "sp_ring.h"

static int ring_avail_locked(const sp_ring *r)
{
	uint32_t w = atomic_load_explicit(&r->write_pos, memory_order_acquire);
	uint32_t rd = atomic_load_explicit(&r->read_pos, memory_order_acquire);
	int avail = (int)(w - rd);

	if (avail > SP_RING_CAPACITY)
		return SP_RING_CAPACITY;
	return avail;
}

void sp_ring_init(sp_ring *r)
{
	atomic_store_explicit(&r->write_pos, 0, memory_order_relaxed);
	atomic_store_explicit(&r->read_pos, 0, memory_order_relaxed);
}

int sp_ring_avail(const sp_ring *r)
{
	return ring_avail_locked(r);
}

int sp_ring_write(sp_ring *r, const float *src, int samples)
{
	int free_space = SP_RING_CAPACITY - ring_avail_locked(r);

	if (samples > free_space)
		samples = free_space;
	if (samples <= 0)
		return 0;

	uint32_t w = atomic_load_explicit(&r->write_pos, memory_order_relaxed);
	int idx = (int)(w % SP_RING_CAPACITY);
	int first = SP_RING_CAPACITY - idx;

	if (first > samples)
		first = samples;

	for (int i = 0; i < first; i++)
		r->buf[idx + i] = src[i];
	for (int i = 0; i < samples - first; i++)
		r->buf[i] = src[first + i];

	atomic_store_explicit(&r->write_pos, w + (uint32_t)samples,
			      memory_order_release);
	return samples;
}

int sp_ring_read(sp_ring *r, float *dst, int samples)
{
	int avail = ring_avail_locked(r);

	if (samples > avail)
		samples = avail;
	if (samples <= 0)
		return 0;

	uint32_t rd = atomic_load_explicit(&r->read_pos, memory_order_relaxed);
	int idx = (int)(rd % SP_RING_CAPACITY);
	int first = SP_RING_CAPACITY - idx;

	if (first > samples)
		first = samples;

	for (int i = 0; i < first; i++)
		dst[i] = r->buf[idx + i];
	for (int i = 0; i < samples - first; i++)
		dst[first + i] = r->buf[i];

	atomic_store_explicit(&r->read_pos, rd + (uint32_t)samples,
			      memory_order_release);
	return samples;
}
