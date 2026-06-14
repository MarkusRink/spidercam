#include "sp_devices.h"

#include <pipewire/pipewire.h>
#include <stdio.h>
#include <string.h>

struct list_ctx {
	struct pw_main_loop *loop;
	struct pw_context *context;
	struct pw_core *core;
	struct pw_registry *registry;
	struct spa_hook core_listener;
	struct spa_hook registry_listener;
	sp_device *out;
	int max;
	int count;
	const char *kind;
	int sync_seq;
};

static void copy_label(sp_device *dev, const struct spa_dict *props)
{
	const char *label;

	label = spa_dict_lookup(props, "node.description");
	if (!label)
		label = spa_dict_lookup(props, "node.nick");
	if (!label)
		label = spa_dict_lookup(props, "node.name");
	if (!label)
		label = "";

	snprintf(dev->label, sizeof(dev->label), "%s", label);
}

static void on_global(void *userdata, uint32_t id, uint32_t permissions,
		      const char *type, uint32_t version, const struct spa_dict *props)
{
	struct list_ctx *ctx = userdata;
	const char *media_class;

	(void)permissions;
	(void)version;

	if (strcmp(type, PW_TYPE_INTERFACE_Node) != 0)
		return;

	media_class = spa_dict_lookup(props, "media.class");
	if (!media_class)
		return;

	if (strcmp(ctx->kind, "sink") == 0) {
		if (strcmp(media_class, "Audio/Sink") != 0)
			return;
	} else if (strcmp(ctx->kind, "source") == 0) {
		if (strcmp(media_class, "Audio/Source") != 0)
			return;
	} else {
		return;
	}

	if (ctx->count >= ctx->max)
		return;

	snprintf(ctx->out[ctx->count].id, sizeof(ctx->out[ctx->count].id), "%u", id);
	copy_label(&ctx->out[ctx->count], props);
	ctx->count++;
}

static const struct pw_registry_events registry_events = {
	PW_VERSION_REGISTRY_EVENTS,
	.global = on_global,
};

static void on_core_done(void *data, uint32_t id, int seq)
{
	struct list_ctx *ctx = data;

	if (id == PW_ID_CORE && seq == ctx->sync_seq)
		pw_main_loop_quit(ctx->loop);
}

static const struct pw_core_events core_events = {
	PW_VERSION_CORE_EVENTS,
	.done = on_core_done,
};

static int list_nodes(sp_device *out, int max, const char *kind)
{
	struct list_ctx ctx = { 0 };
	bool pw_inited = false;

	if (!out || max <= 0 || !kind)
		return -1;

	ctx.out = out;
	ctx.max = max;
	ctx.kind = kind;
	ctx.count = 0;

	pw_init(NULL, NULL);
	pw_inited = true;

	ctx.loop = pw_main_loop_new(NULL);
	if (!ctx.loop)
		goto fail;

	ctx.context = pw_context_new(pw_main_loop_get_loop(ctx.loop), NULL, 0);
	if (!ctx.context)
		goto fail;

	ctx.core = pw_context_connect(ctx.context, NULL, 0);
	if (!ctx.core)
		goto fail;

	pw_core_add_listener(ctx.core, &ctx.core_listener, &core_events, &ctx);

	ctx.registry = pw_core_get_registry(ctx.core, PW_VERSION_REGISTRY, 0);
	if (!ctx.registry)
		goto fail;

	pw_registry_add_listener(ctx.registry, &ctx.registry_listener,
				 &registry_events, &ctx);

	ctx.sync_seq = pw_core_sync(ctx.core, PW_ID_CORE, 0);
	pw_main_loop_run(ctx.loop);

	spa_hook_remove(&ctx.registry_listener);
	spa_hook_remove(&ctx.core_listener);
	if (ctx.core)
		pw_core_disconnect(ctx.core);
	if (ctx.context)
		pw_context_destroy(ctx.context);
	if (ctx.loop)
		pw_main_loop_destroy(ctx.loop);
	if (pw_inited)
		pw_deinit();
	return ctx.count;

fail:
	spa_hook_remove(&ctx.registry_listener);
	spa_hook_remove(&ctx.core_listener);
	if (ctx.core)
		pw_core_disconnect(ctx.core);
	if (ctx.context)
		pw_context_destroy(ctx.context);
	if (ctx.loop)
		pw_main_loop_destroy(ctx.loop);
	if (pw_inited)
		pw_deinit();
	return -1;
}

int sp_list_sources(sp_device *out, int max)
{
	return list_nodes(out, max, "source");
}

int sp_list_sinks(sp_device *out, int max)
{
	return list_nodes(out, max, "sink");
}
