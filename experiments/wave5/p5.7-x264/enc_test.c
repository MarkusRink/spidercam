#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <x264.h>

#define WIDTH 1280
#define HEIGHT 720

static void write_be32(FILE *f, uint32_t v)
{
	uint8_t b[4] = {
		(uint8_t)((v >> 24) & 0xff),
		(uint8_t)((v >> 16) & 0xff),
		(uint8_t)((v >> 8) & 0xff),
		(uint8_t)(v & 0xff),
	};
	fwrite(b, 1, 4, f);
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

static void rgba_to_i420(const uint8_t *rgba, int stride, x264_image_t *img)
{
	for (int y = 0; y < HEIGHT; y++) {
		for (int x = 0; x < WIDTH; x++) {
			uint8_t r = rgba[y * stride + x * 4 + 0];
			uint8_t g = rgba[y * stride + x * 4 + 1];
			uint8_t b = rgba[y * stride + x * 4 + 2];
			img->plane[0][y * img->i_stride[0] + x] =
				(uint8_t)(((66 * r + 129 * g + 25 * b + 128) >> 8) + 16);
		}
	}
	for (int y = 0; y < HEIGHT; y += 2) {
		for (int x = 0; x < WIDTH; x += 2) {
			int u_sum = 0;
			int v_sum = 0;
			for (int dy = 0; dy < 2; dy++) {
				for (int dx = 0; dx < 2; dx++) {
					int px = x + dx;
					int py = y + dy;
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

static int write_annexb_frame(FILE *h264, x264_nal_t *nals, int i_nals)
{
	for (int i = 0; i < i_nals; i++) {
		if (fwrite(nals[i].p_payload, 1, (size_t)nals[i].i_payload, h264) !=
		    (size_t)nals[i].i_payload) {
			return -1;
		}
	}
	return 0;
}

static int write_avcc_access_unit(FILE *avcc, x264_nal_t *nals, int i_nals)
{
	for (int i = 0; i < i_nals; i++) {
		const uint8_t *p = nals[i].p_payload;
		int len = nals[i].i_payload;
		strip_start_code(&p, &len);
		if (len <= 0) {
			continue;
		}
		write_be32(avcc, (uint32_t)len);
		if (fwrite(p, 1, (size_t)len, avcc) != (size_t)len) {
			return -1;
		}
	}
	return 0;
}

int main(void)
{
	x264_param_t param;
	x264_picture_t pic_in;
	x264_picture_t pic_out;
	x264_t *enc = NULL;
	x264_nal_t *nals = NULL;
	int i_nals = 0;
	uint8_t *rgba = NULL;
	FILE *h264 = NULL;
	FILE *avcc = NULL;
	int ret = 1;
	int wrote_avcc = 0;

	if (x264_param_default_preset(&param, "ultrafast", "zerolatency") < 0) {
		fprintf(stderr, "x264_param_default_preset failed\n");
		goto done;
	}
	param.i_width = WIDTH;
	param.i_height = HEIGHT;
	param.i_csp = X264_CSP_I420;
	param.i_fps_num = 30;
	param.i_fps_den = 1;
	param.i_keyint_max = 30;
	param.b_repeat_headers = 1;
	param.b_annexb = 1;
	if (x264_param_apply_profile(&param, "baseline") < 0) {
		fprintf(stderr, "x264_param_apply_profile failed\n");
		goto done;
	}

	enc = x264_encoder_open(&param);
	if (!enc) {
		fprintf(stderr, "x264_encoder_open failed\n");
		goto done;
	}

	if (x264_picture_alloc(&pic_in, X264_CSP_I420, WIDTH, HEIGHT) < 0) {
		fprintf(stderr, "x264_picture_alloc failed\n");
		goto done;
	}

	rgba = malloc((size_t)WIDTH * HEIGHT * 4);
	if (!rgba) {
		fprintf(stderr, "malloc rgba failed\n");
		goto done;
	}
	for (int i = 0; i < WIDTH * HEIGHT; i++) {
		rgba[i * 4 + 0] = 255;
		rgba[i * 4 + 1] = 0;
		rgba[i * 4 + 2] = 0;
		rgba[i * 4 + 3] = 255;
	}
	rgba_to_i420(rgba, WIDTH * 4, &pic_in.img);
	pic_in.i_pts = 0;
	pic_in.i_dts = 0;
	pic_in.i_type = X264_TYPE_IDR;

	h264 = fopen("output/test.h264", "wb");
	if (!h264) {
		perror("fopen output/test.h264");
		goto done;
	}
	avcc = fopen("output/sample.avcc", "wb");
	if (!avcc) {
		perror("fopen output/sample.avcc");
		goto done;
	}

	if (x264_encoder_encode(enc, &nals, &i_nals, &pic_in, &pic_out) < 0) {
		fprintf(stderr, "x264_encoder_encode failed\n");
		goto done;
	}
	if (i_nals == 0) {
		fprintf(stderr, "no NALs on first encode\n");
		goto done;
	}
	if (write_annexb_frame(h264, nals, i_nals) < 0) {
		fprintf(stderr, "write annexb failed\n");
		goto done;
	}
	if (write_avcc_access_unit(avcc, nals, i_nals) < 0) {
		fprintf(stderr, "write avcc failed\n");
		goto done;
	}
	wrote_avcc = 1;

	while (x264_encoder_encode(enc, &nals, &i_nals, NULL, &pic_out) > 0) {
		if (write_annexb_frame(h264, nals, i_nals) < 0) {
			fprintf(stderr, "write annexb flush failed\n");
			goto done;
		}
	}

	if (!wrote_avcc) {
		fprintf(stderr, "no AVCC sample written\n");
		goto done;
	}

	printf("wrote output/test.h264 and output/sample.avcc (%dx%d IDR)\n",
	       WIDTH, HEIGHT);
	ret = 0;

done:
	if (h264) {
		fclose(h264);
	}
	if (avcc) {
		fclose(avcc);
	}
	if (rgba) {
		free(rgba);
	}
	x264_picture_clean(&pic_in);
	if (enc) {
		x264_encoder_close(enc);
	}
	return ret;
}
