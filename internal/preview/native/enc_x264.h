#ifndef SP_ENC_X264_H
#define SP_ENC_X264_H

#include <stdint.h>

typedef struct sp_x264_enc sp_x264_enc;

sp_x264_enc *sp_x264_open(int width, int height, int fps, int bitrate_kbps);
void sp_x264_close(sp_x264_enc *enc);
void sp_x264_force_keyframe(sp_x264_enc *enc);
int sp_x264_encode(
	sp_x264_enc *enc,
	const uint8_t *rgba,
	int width,
	int height,
	int64_t pts,
	const uint8_t **out_avcc,
	int *out_len,
	int *is_keyframe
);

#endif
