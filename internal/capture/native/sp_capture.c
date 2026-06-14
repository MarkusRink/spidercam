#include "sp_capture.h"
#include "sp_ring.h"

#include <errno.h>
#include <pthread.h>
#include <spa/param/audio/format-utils.h>
#include <spa/pod/builder.h>
#include <stdatomic.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <pipewire/pipewire.h>

struct sp_capture {
	pthread_t thread;
	struct pw_main_loop *loop;
	struct pw_context *context;
	struct pw_core *core;
	struct pw_stream *mic_stream;
	struct pw_stream *mon_stream;
	sp_ring mic_ring;
	sp_ring mon_ring;
	int sample_rate;
	char mic_id[128];
	char sink_id[128];
	pthread_mutex_t ready_mtx;
	pthread_cond_t ready_cv;
	bool ready;
	int error;
	int pending_sync;
	struct spa_hook core_listener;
	atomic_bool alive;
};

static void push_stream_pcm(sp_ring *ring, struct pw_stream *stream)
{
	struct pw_buffer *b = pw_stream_dequeue_buffer(stream);

	if (!b || !b->buffer || b->buffer->n_datas == 0)
		return;

	struct spa_data *d = &b->buffer->datas[0];
	if (!d->data || !d->chunk || d->chunk->size == 0) {
		pw_stream_queue_buffer(stream, b);
		return;
	}

	int samples = (int)(d->chunk->size / sizeof(float));
	sp_ring_write(ring, (const float *)d->data, samples);
	pw_stream_queue_buffer(stream, b);
}

static void on_mic_process(void *userdata)
{
	struct sp_capture *c = userdata;

	if (!atomic_load_explicit(&c->alive, memory_order_acquire))
		return;
	push_stream_pcm(&c->mic_ring, c->mic_stream);
}

static void on_mon_process(void *userdata)
{
	struct sp_capture *c = userdata;

	if (!atomic_load_explicit(&c->alive, memory_order_acquire))
		return;
	push_stream_pcm(&c->mon_ring, c->mon_stream);
}

static const struct pw_stream_events mic_stream_events = {
	PW_VERSION_STREAM_EVENTS,
	.process = on_mic_process,
};

static const struct pw_stream_events mon_stream_events = {
	PW_VERSION_STREAM_EVENTS,
	.process = on_mon_process,
};

static const struct spa_pod *build_format(struct spa_pod_builder *b, int rate)
{
	return spa_format_audio_raw_build(
		b, SPA_PARAM_EnumFormat,
		&SPA_AUDIO_INFO_RAW_INIT(.format = SPA_AUDIO_FORMAT_F32,
					 .channels = 1, .rate = rate));
}

static int connect_capture_stream(struct sp_capture *c, struct pw_stream **stream_out,
				  const char *stream_name, const char *target_id,
				  bool sink_monitor,
				  const struct pw_stream_events *events)
{
	struct pw_loop *loop = pw_main_loop_get_loop(c->loop);
	struct pw_properties *props = pw_properties_new(
		PW_KEY_MEDIA_TYPE, "Audio", PW_KEY_MEDIA_CATEGORY, "Capture",
		PW_KEY_MEDIA_ROLE, "DSP", NULL);

	if (sink_monitor)
		pw_properties_set(props, PW_KEY_STREAM_CAPTURE_SINK, "true");
	if (target_id && target_id[0])
		pw_properties_set(props, PW_KEY_TARGET_OBJECT, target_id);

	struct pw_stream *stream =
		pw_stream_new_simple(loop, stream_name, props, events, c);
	if (!stream)
		return -ENOMEM;

	uint8_t buffer[1024];
	struct spa_pod_builder b = SPA_POD_BUILDER_INIT(buffer, sizeof(buffer));
	const struct spa_pod *params[1];

	params[0] = build_format(&b, c->sample_rate);

	int flags = PW_STREAM_FLAG_AUTOCONNECT | PW_STREAM_FLAG_MAP_BUFFERS;
	int err = pw_stream_connect(stream, PW_DIRECTION_INPUT, PW_ID_ANY, flags,
				    params, 1);
	if (err < 0) {
		pw_stream_destroy(stream);
		return err;
	}

	*stream_out = stream;
	return 0;
}

static void on_core_done(void *data, uint32_t id, int seq)
{
	struct sp_capture *c = data;

	(void)id;
	if (seq == c->pending_sync)
		c->pending_sync = -1;
}

static void on_core_error(void *data, uint32_t id, int seq, int res,
			  const char *message)
{
	struct sp_capture *c = data;

	(void)id;
	(void)seq;
	c->error = res;
	fprintf(stderr, "pipewire core error: %s (%d)\n", message, res);
}

static const struct pw_core_events core_events = {
	PW_VERSION_CORE_EVENTS,
	.done = on_core_done,
	.error = on_core_error,
};

static int wait_for_sync(struct sp_capture *c)
{
	int seq = pw_core_sync(c->core, PW_ID_CORE, 0);

	c->pending_sync = seq;

	while (c->pending_sync == seq) {
		int res = pw_loop_iterate(pw_main_loop_get_loop(c->loop), 0);

		if (res < 0)
			return res;
		if (c->error < 0)
			return c->error;
	}
	return 0;
}

static void signal_ready(struct sp_capture *c)
{
	pthread_mutex_lock(&c->ready_mtx);
	c->ready = true;
	pthread_cond_broadcast(&c->ready_cv);
	pthread_mutex_unlock(&c->ready_mtx);
}

static void teardown_pw(struct sp_capture *c)
{
	spa_hook_remove(&c->core_listener);
	if (c->mon_stream) {
		pw_stream_destroy(c->mon_stream);
		c->mon_stream = NULL;
	}
	if (c->mic_stream) {
		pw_stream_destroy(c->mic_stream);
		c->mic_stream = NULL;
	}
	if (c->core) {
		pw_core_disconnect(c->core);
		c->core = NULL;
	}
	if (c->context) {
		pw_context_destroy(c->context);
		c->context = NULL;
	}
	if (c->loop) {
		pw_main_loop_destroy(c->loop);
		c->loop = NULL;
	}
}

static void *capture_thread(void *arg)
{
	struct sp_capture *c = arg;
	bool pw_inited = false;

	pw_init(NULL, NULL);
	pw_inited = true;

	c->loop = pw_main_loop_new(NULL);
	if (!c->loop) {
		c->error = -ENOMEM;
		goto out;
	}

	struct pw_loop *loop = pw_main_loop_get_loop(c->loop);
	c->context = pw_context_new(loop, NULL, 0);
	if (!c->context) {
		c->error = -ENOMEM;
		goto out;
	}

	c->core = pw_context_connect(c->context, NULL, 0);
	if (!c->core) {
		c->error = -EIO;
		goto out;
	}

	pw_core_add_listener(c->core, &c->core_listener, &core_events, c);
	if (wait_for_sync(c) < 0)
		goto out;

	if (connect_capture_stream(c, &c->mic_stream, "sp-mic",
				   c->mic_id[0] ? c->mic_id : NULL, false,
				   &mic_stream_events) < 0) {
		c->error = -EIO;
		goto out;
	}

	if (connect_capture_stream(c, &c->mon_stream, "sp-mon",
				   c->sink_id[0] ? c->sink_id : NULL, true,
				   &mon_stream_events) < 0) {
		c->error = -EIO;
		goto out;
	}

	if (wait_for_sync(c) < 0)
		goto out;

	signal_ready(c);
	pw_main_loop_run(c->loop);

out:
	teardown_pw(c);
	if (pw_inited)
		pw_deinit();
	if (!c->ready)
		signal_ready(c);
	return NULL;
}

sp_capture *sp_capture_open(const char *mic_id, const char *sink_id,
			    int sample_rate)
{
	struct sp_capture *c = calloc(1, sizeof(*c));

	if (!c)
		return NULL;

	if (sample_rate <= 0)
		sample_rate = 48000;
	c->sample_rate = sample_rate;

	if (mic_id && mic_id[0])
		snprintf(c->mic_id, sizeof(c->mic_id), "%s", mic_id);
	if (sink_id && sink_id[0])
		snprintf(c->sink_id, sizeof(c->sink_id), "%s", sink_id);

	sp_ring_init(&c->mic_ring);
	sp_ring_init(&c->mon_ring);
	pthread_mutex_init(&c->ready_mtx, NULL);
	pthread_cond_init(&c->ready_cv, NULL);
	atomic_store_explicit(&c->alive, true, memory_order_release);

	if (pthread_create(&c->thread, NULL, capture_thread, c) != 0) {
		pthread_mutex_destroy(&c->ready_mtx);
		pthread_cond_destroy(&c->ready_cv);
		free(c);
		return NULL;
	}

	pthread_mutex_lock(&c->ready_mtx);
	while (!c->ready)
		pthread_cond_wait(&c->ready_cv, &c->ready_mtx);
	pthread_mutex_unlock(&c->ready_mtx);

	if (c->error < 0) {
		sp_capture_close(c);
		return NULL;
	}

	return c;
}

void sp_capture_close(sp_capture *c)
{
	if (!c)
		return;

	atomic_store_explicit(&c->alive, false, memory_order_release);
	if (c->loop)
		pw_main_loop_quit(c->loop);
	pthread_join(c->thread, NULL);
	pthread_mutex_destroy(&c->ready_mtx);
	pthread_cond_destroy(&c->ready_cv);
	free(c);
}

int sp_capture_read_mic(sp_capture *c, float *buf, int frames)
{
	if (!c || !buf || frames <= 0)
		return 0;
	if (!atomic_load_explicit(&c->alive, memory_order_acquire))
		return 0;
	return sp_ring_read(&c->mic_ring, buf, frames);
}

int sp_capture_read_monitor(sp_capture *c, float *buf, int frames)
{
	if (!c || !buf || frames <= 0)
		return 0;
	if (!atomic_load_explicit(&c->alive, memory_order_acquire))
		return 0;
	return sp_ring_read(&c->mon_ring, buf, frames);
}
