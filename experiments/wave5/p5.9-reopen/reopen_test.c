#include "sp_capture.h"
#include "sp_ring.h"

#include <math.h>
#include <pipewire/pipewire.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

enum { MAX_SINKS = 32 };

struct sink_entry {
	uint32_t id;
	char label[128];
};

struct list_ctx {
	struct pw_main_loop *loop;
	struct pw_context *context;
	struct pw_core *core;
	struct pw_registry *registry;
	struct spa_hook core_listener;
	struct spa_hook registry_listener;
	struct sink_entry sinks[MAX_SINKS];
	int sink_count;
	int sync_seq;
};

static double now_ms(void)
{
	struct timespec ts;

	clock_gettime(CLOCK_MONOTONIC, &ts);
	return (double)ts.tv_sec * 1000.0 + (double)ts.tv_nsec / 1e6;
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

static void on_list_global(void *userdata, uint32_t id, uint32_t permissions,
			   const char *type, uint32_t version,
			   const struct spa_dict *props)
{
	struct list_ctx *ctx = userdata;
	const char *media_class;
	const char *label;

	(void)permissions;
	(void)version;

	if (strcmp(type, PW_TYPE_INTERFACE_Node) != 0)
		return;

	media_class = spa_dict_lookup(props, "media.class");
	if (!media_class || strcmp(media_class, "Audio/Sink") != 0)
		return;

	if (ctx->sink_count >= MAX_SINKS)
		return;

	label = spa_dict_lookup(props, "node.description");
	if (!label)
		label = spa_dict_lookup(props, "node.nick");
	if (!label)
		label = spa_dict_lookup(props, "node.name");
	if (!label)
		label = "";

	ctx->sinks[ctx->sink_count].id = id;
	snprintf(ctx->sinks[ctx->sink_count].label,
		 sizeof(ctx->sinks[ctx->sink_count].label), "%s", label);
	ctx->sink_count++;
}

static const struct pw_registry_events list_registry_events = {
	PW_VERSION_REGISTRY_EVENTS,
	.global = on_list_global,
};

static void on_list_core_done(void *data, uint32_t id, int seq)
{
	struct list_ctx *ctx = data;

	if (id == PW_ID_CORE && seq == ctx->sync_seq)
		pw_main_loop_quit(ctx->loop);
}

static const struct pw_core_events list_core_events = {
	PW_VERSION_CORE_EVENTS,
	.done = on_list_core_done,
};

static int list_sinks(struct sink_entry *sinks, int max_sinks)
{
	struct list_ctx ctx = { 0 };

	ctx.sink_count = 0;

	pw_init(NULL, NULL);

	ctx.loop = pw_main_loop_new(NULL);
	if (!ctx.loop)
		goto fail;

	ctx.context = pw_context_new(pw_main_loop_get_loop(ctx.loop), NULL, 0);
	if (!ctx.context)
		goto fail;

	ctx.core = pw_context_connect(ctx.context, NULL, 0);
	if (!ctx.core)
		goto fail;

	pw_core_add_listener(ctx.core, &ctx.core_listener, &list_core_events,
			     &ctx);

	ctx.registry = pw_core_get_registry(ctx.core, PW_VERSION_REGISTRY, 0);
	pw_registry_add_listener(ctx.registry, &ctx.registry_listener,
				 &list_registry_events, &ctx);

	ctx.sync_seq = pw_core_sync(ctx.core, PW_ID_CORE, 0);
	pw_main_loop_run(ctx.loop);

	for (int i = 0; i < ctx.sink_count && i < max_sinks; i++)
		sinks[i] = ctx.sinks[i];

	int count = ctx.sink_count;

	spa_hook_remove(&ctx.registry_listener);
	spa_hook_remove(&ctx.core_listener);
	if (ctx.core)
		pw_core_disconnect(ctx.core);
	if (ctx.context)
		pw_context_destroy(ctx.context);
	if (ctx.loop)
		pw_main_loop_destroy(ctx.loop);
	pw_deinit();
	return count;

fail:
	spa_hook_remove(&ctx.registry_listener);
	spa_hook_remove(&ctx.core_listener);
	if (ctx.core)
		pw_core_disconnect(ctx.core);
	if (ctx.context)
		pw_context_destroy(ctx.context);
	if (ctx.loop)
		pw_main_loop_destroy(ctx.loop);
	pw_deinit();
	return -1;
}

static int read_frames(sp_capture *cap, int count, const char *phase)
{
	float mon_buf[SP_FRAME_SAMPLES];
	double rms_sum = 0.0;
	int rms_count = 0;

	for (int frame = 0; frame < count; frame++) {
		usleep(10000);

		int mon_n = sp_capture_read_monitor(cap, mon_buf, SP_FRAME_SAMPLES);
		float rms = frame_rms(mon_buf, mon_n);

		rms_sum += (double)rms;
		rms_count++;

		if (frame < 3 || frame >= count - 2)
			printf("  %s frame %2d  mon_rms=%.6f (%d)\n", phase, frame,
			       rms, mon_n);
	}

	printf("  %s avg mon_rms=%.6f over %d frames\n", phase,
	       rms_count > 0 ? rms_sum / (double)rms_count : 0.0, count);
	return 0;
}

static char sink_a_id[32];
static char sink_b_id[32];

static const char *sink_id_str(uint32_t id, char *buf, size_t buflen)
{
	snprintf(buf, buflen, "%u", id);
	return buf;
}

int main(void)
{
	struct sink_entry sinks[MAX_SINKS];
	int sink_count;
	bool switch_sinks;
	const char *sink_a;
	const char *sink_b;
	double reopen_ms = 0.0;
	int result = 0;

	printf("P5.9 reopen test — enumerate sinks\n");

	sink_count = list_sinks(sinks, MAX_SINKS);
	if (sink_count < 0) {
		fprintf(stderr, "FAIL: sink enumeration failed\n");
		return 1;
	}

	if (sink_count == 0) {
		fprintf(stderr, "FAIL: no sinks found\n");
		return 1;
	}

	for (int i = 0; i < sink_count; i++)
		printf("  sink[%d] id=%u label=%s\n", i, sinks[i].id,
		       sinks[i].label);

	switch_sinks = sink_count >= 2;
	sink_a = sink_id_str(sinks[0].id, sink_a_id, sizeof(sink_a_id));
	sink_b = switch_sinks ? sink_id_str(sinks[1].id, sink_b_id, sizeof(sink_b_id))
			      : sink_a;

	if (!switch_sinks)
		printf("SKIP: only one sink — will close/reopen same sink (%s)\n",
		       sink_a);
	else
		printf("sink A=%s  sink B=%s\n", sink_a, sink_b);

	printf("phase 1: open mic=(default) sink=%s\n", sink_a);
	sp_capture *cap = sp_capture_open(NULL, sink_a, 48000);
	if (!cap) {
		fprintf(stderr, "FAIL: sp_capture_open (sink A) failed\n");
		return 1;
	}

	read_frames(cap, 50, "phase1");

	printf("closing capture...\n");
	sp_capture_close(cap);
	cap = NULL;

	double t0 = now_ms();
	printf("phase 2: reopen mic=(default) sink=%s\n", sink_b);
	cap = sp_capture_open(NULL, sink_b, 48000);
	reopen_ms = now_ms() - t0;

	if (!cap) {
		fprintf(stderr, "FAIL: sp_capture_open (sink B) failed\n");
		return 1;
	}

	read_frames(cap, 50, "phase2");
	sp_capture_close(cap);

	printf("reopen_ms=%.2f\n", reopen_ms);

	if (reopen_ms >= 500.0) {
		printf("FAIL: reopen took %.2f ms (limit 500 ms)\n", reopen_ms);
		result = 1;
	} else if (!switch_sinks) {
		printf("SKIP: single sink — close/reopen OK, reopen_ms=%.2f\n",
		       reopen_ms);
		result = 2;
	} else {
		printf("PASS: sink switch reopen_ms=%.2f, no crash\n", reopen_ms);
		result = 0;
	}

	return result;
}
