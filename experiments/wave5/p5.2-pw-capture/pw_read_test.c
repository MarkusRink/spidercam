#include "sp_capture.h"
#include "sp_ring.h"

#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

static void usage(const char *prog)
{
	fprintf(stderr,
		"usage: %s [--mic-id ID] [--sink-id ID]\n"
		"  empty ID = PipeWire default route\n",
		prog);
}

static int parse_args(int argc, char **argv, const char **mic_id,
		      const char **sink_id)
{
	*mic_id = NULL;
	*sink_id = NULL;

	for (int i = 1; i < argc; i++) {
		if (strcmp(argv[i], "--mic-id") == 0) {
			if (i + 1 >= argc) {
				usage(argv[0]);
				return -1;
			}
			*mic_id = argv[++i];
		} else if (strcmp(argv[i], "--sink-id") == 0) {
			if (i + 1 >= argc) {
				usage(argv[0]);
				return -1;
			}
			*sink_id = argv[++i];
		} else if (strcmp(argv[i], "-h") == 0 ||
			   strcmp(argv[i], "--help") == 0) {
			usage(argv[0]);
			return -1;
		} else {
			fprintf(stderr, "unknown argument: %s\n", argv[i]);
			usage(argv[0]);
			return -1;
		}
	}

	return 0;
}

static float frame_rms(const float *buf, int n)
{
	if (n <= 0)
		return 0.0f;

	double sum = 0.0;
	for (int i = 0; i < n; i++)
		sum += (double)buf[i] * (double)buf[i];
	return (float)sqrt(sum / (double)n);
}

int main(int argc, char **argv)
{
	const char *mic_id = NULL;
	const char *sink_id = NULL;

	if (parse_args(argc, argv, &mic_id, &sink_id) < 0)
		return 1;

	printf("opening capture mic_id=%s sink_id=%s\n",
	       mic_id && mic_id[0] ? mic_id : "(default)",
	       sink_id && sink_id[0] ? sink_id : "(default)");

	sp_capture *cap = sp_capture_open(mic_id, sink_id, 48000);
	if (!cap) {
		fprintf(stderr, "sp_capture_open failed\n");
		return 1;
	}

	float mic_buf[SP_FRAME_SAMPLES];
	float mon_buf[SP_FRAME_SAMPLES];

	for (int frame = 0; frame < 100; frame++) {
		usleep(10000);

		int mic_n = sp_capture_read_mic(cap, mic_buf, SP_FRAME_SAMPLES);
		int mon_n =
			sp_capture_read_monitor(cap, mon_buf, SP_FRAME_SAMPLES);

		printf("frame %3d  mic_rms=%.6f (%d)  mon_rms=%.6f (%d)\n", frame,
		       frame_rms(mic_buf, mic_n), mic_n,
		       frame_rms(mon_buf, mon_n), mon_n);
	}

	sp_capture_close(cap);
	printf("done: 100 frames\n");
	return 0;
}
