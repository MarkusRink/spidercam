#include "enc_x264.h"

#include <stdlib.h>
#include <string.h>

#include <x264.h>

struct sp_x264_enc {
	x264_t *enc;
	x264_picture_t pic_in;
	int width;
	int height;
	int fps;
	int force_keyframe;
	uint8_t *avcc_buf;
	int avcc_cap;
	int avcc_len;
};

static void write_be32(uint8_t *dst, uint32_t v)
{
	dst[0] = (uint8_t)((v >> 24) & 0xff);
	dst[1] = (uint8_t)((v >> 16) & 0xff);
	dst[2] = (uint8_t)((v >> 8) & 0xff);
	dst[3] = (uint8_t)(v & 0xff);
}

static void strip_start_code(const uint8_t **payload, int *len)
{
	const uint8_t *p = *payload;
	int n = *len;
	if (n >= 4 && p[0] == 0 && p[1] == 0 && p[2] == 0 && p[3] == 1) {
		p += 4;
		n -= 4;
	} else if (n >= 3 && p[0] == 0 && p[1] == 0 && p[2] == 1) {
		p += 3;
		n -= 3;
	}
	*payload = p;
	*len = n;
}

static void rgba_to_i420(
	const uint8_t *rgba,
	int stride,
	int width,
	int height,
	x264_image_t *img
)
{
	for (int y = 0; y < height; y++) {
		for (int x = 0; x < width; x++) {
			uint8_t r = rgba[y * stride + x * 4 + 0];
			uint8_t g = rgba[y * stride + x * 4 + 1];
			uint8_t b = rgba[y * stride + x * 4 + 2];
			img->plane[0][y * img->i_stride[0] + x] =
				(uint8_t)(((66 * r + 129 * g + 25 * b + 128) >> 8) + 16);
		}
	}
	for (int y = 0; y < height; y += 2) {
		for (int x = 0; x < width; x += 2) {
			int u_sum = 0;
			int v_sum = 0;
			for (int dy = 0; dy < 2; dy++) {
				for (int dx = 0; dx < 2; dx++) {
					int px = x + dx;
					int py = y + dy;
					if (px >= width || py >= height) {
						continue;
					}
					uint8_t r = rgba[py * stride + px * 4 + 0];
					uint8_t g = rgba[py * stride + px * 4 + 1];
					uint8_t b = rgba[py * stride + px * 4 + 2];
					u_sum += ((-38 * r - 74 * g + 112 * b + 128) >> 8) + 128;
					v_sum += ((112 * r - 94 * g - 18 * b + 128) >> 8) + 128;
				}
			}
			img->plane[1][(y / 2) * img->i_stride[1] + (x / 2)] =
				(uint8_t)(u_sum / 4);
			img->plane[2][(y / 2) * img->i_stride[2] + (x / 2)] =
				(uint8_t)(v_sum / 4);
		}
	}
}

static int append_avcc(sp_x264_enc *e, x264_nal_t *nals, int i_nals)
{
	int need = 0;
	for (int i = 0; i < i_nals; i++) {
		const uint8_t *p = nals[i].p_payload;
		int len = nals[i].i_payload;
		strip_start_code(&p, &len);
		if (len > 0) {
			need += 4 + len;
		}
	}
	if (need <= 0) {
		e->avcc_len = 0;
		return 0;
	}
	if (need > e->avcc_cap) {
		uint8_t *nb = realloc(e->avcc_buf, (size_t)need);
		if (!nb) {
			return -1;
		}
		e->avcc_buf = nb;
		e->avcc_cap = need;
	}

	int off = 0;
	for (int i = 0; i < i_nals; i++) {
		const uint8_t *p = nals[i].p_payload;
		int len = nals[i].i_payload;
		strip_start_code(&p, &len);
		if (len <= 0) {
			continue;
		}
		write_be32(e->avcc_buf + off, (uint32_t)len);
		off += 4;
		memcpy(e->avcc_buf + off, p, (size_t)len);
		off += len;
	}
	e->avcc_len = off;
	return 0;
}

sp_x264_enc *sp_x264_open(int width, int height, int fps, int bitrate_kbps)
{
	if (width <= 0 || height <= 0 || fps <= 0) {
		return NULL;
	}

	sp_x264_enc *e = calloc(1, sizeof(*e));
	if (!e) {
		return NULL;
	}
	e->width = width;
	e->height = height;
	e->fps = fps;

	x264_param_t param;
	if (x264_param_default_preset(&param, "ultrafast", "zerolatency") < 0) {
		goto fail;
	}
	param.i_width = width;
	param.i_height = height;
	param.i_csp = X264_CSP_I420;
	param.i_fps_num = fps;
	param.i_fps_den = 1;
	param.i_keyint_max = fps;
	param.i_bframe = 0;
	param.b_repeat_headers = 1;
	param.b_annexb = 1;
	param.rc.i_rc_method = X264_RC_ABR;
	param.rc.i_bitrate = bitrate_kbps;
	if (x264_param_apply_profile(&param, "baseline") < 0) {
		goto fail;
	}

	e->enc = x264_encoder_open(&param);
	if (!e->enc) {
		goto fail;
	}
	if (x264_picture_alloc(&e->pic_in, X264_CSP_I420, width, height) < 0) {
		goto fail;
	}
	e->force_keyframe = 1;
	return e;

fail:
	sp_x264_close(e);
	return NULL;
}

void sp_x264_close(sp_x264_enc *enc)
{
	if (!enc) {
		return;
	}
	x264_picture_clean(&enc->pic_in);
	if (enc->enc) {
		x264_encoder_close(enc->enc);
	}
	free(enc->avcc_buf);
	free(enc);
}

void sp_x264_force_keyframe(sp_x264_enc *enc)
{
	if (enc) {
		enc->force_keyframe = 1;
	}
}

static int nal_is_keyframe(x264_nal_t *nals, int i_nals)
{
	for (int i = 0; i < i_nals; i++) {
		const uint8_t *p = nals[i].p_payload;
		int len = nals[i].i_payload;
		if (len >= 4 && p[0] == 0 && p[1] == 0 && p[2] == 0 && p[3] == 1) {
			p += 4;
			len -= 4;
		} else if (len >= 3 && p[0] == 0 && p[1] == 0 && p[2] == 1) {
			p += 3;
			len -= 3;
		}
		if (len <= 0) {
			continue;
		}
		int type = p[0] & 0x1f;
		if (type == 5 || type == 7) {
			return 1;
		}
	}
	return 0;
}

int sp_x264_encode(
	sp_x264_enc *enc,
	const uint8_t *rgba,
	int width,
	int height,
	int64_t pts,
	const uint8_t **out_avcc,
	int *out_len,
	int *is_keyframe
)
{
	if (!enc || !enc->enc || !rgba || !out_avcc || !out_len || !is_keyframe) {
		return -1;
	}
	if (width != enc->width || height != enc->height) {
		return -1;
	}

	rgba_to_i420(rgba, width * 4, width, height, &enc->pic_in.img);
	enc->pic_in.i_pts = pts;
	enc->pic_in.i_dts = pts;
	int want_key = enc->force_keyframe;
	if (enc->force_keyframe) {
		enc->pic_in.i_type = X264_TYPE_IDR;
		enc->force_keyframe = 0;
	} else {
		enc->pic_in.i_type = X264_TYPE_AUTO;
	}

	x264_picture_t pic_out;
	x264_nal_t *nals = NULL;
	int i_nals = 0;
	if (x264_encoder_encode(enc->enc, &nals, &i_nals, &enc->pic_in, &pic_out) < 0) {
		return -1;
	}
	if (i_nals == 0) {
		*out_avcc = NULL;
		*out_len = 0;
		*is_keyframe = 0;
		return 0;
	}
	if (append_avcc(enc, nals, i_nals) < 0) {
		return -1;
	}
	*out_avcc = enc->avcc_buf;
	*out_len = enc->avcc_len;
	*is_keyframe = want_key || pic_out.i_type == X264_TYPE_IDR ||
		pic_out.i_type == X264_TYPE_I || nal_is_keyframe(nals, i_nals);
	return 0;
}
