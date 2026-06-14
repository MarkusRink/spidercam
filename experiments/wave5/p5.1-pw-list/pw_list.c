#include <pipewire/pipewire.h>
#include <spa/utils/result.h>

#include <stdio.h>
#include <string.h>

struct app {
	struct pw_main_loop *loop;
	struct pw_context *context;
	struct pw_core *core;
	struct pw_registry *registry;
	struct spa_hook core_listener;
	struct spa_hook registry_listener;
	int sync_seq;
};

static void json_escape(FILE *out, const char *s)
{
	fputc('"', out);
	for (; *s; s++) {
		unsigned char c = (unsigned char)*s;
		switch (c) {
		case '"':
			fputs("\\\"", out);
			break;
		case '\\':
			fputs("\\\\", out);
			break;
		case '\n':
			fputs("\\n", out);
			break;
		case '\r':
			fputs("\\r", out);
			break;
		case '\t':
			fputs("\\t", out);
			break;
		default:
			if (c < 0x20)
				fprintf(out, "\\u%04x", c);
			else
				fputc((int)c, out);
		}
	}
	fputc('"', out);
}

static void emit_device(const char *kind, uint32_t id, const char *label)
{
	char id_buf[32];

	snprintf(id_buf, sizeof id_buf, "%u", id);
	fputs("{\"kind\":\"", stdout);
	fputs(kind, stdout);
	fputs("\",\"id\":", stdout);
	json_escape(stdout, id_buf);
	fputs(",\"label\":", stdout);
	json_escape(stdout, label ? label : "");
	fputs("}\n", stdout);
}

static void on_global(void *userdata, uint32_t id, uint32_t permissions,
		      const char *type, uint32_t version, const struct spa_dict *props)
{
	const char *media_class;
	const char *label;
	const char *kind;

	(void)userdata;
	(void)permissions;
	(void)version;

	if (strcmp(type, PW_TYPE_INTERFACE_Node) != 0)
		return;

	media_class = spa_dict_lookup(props, "media.class");
	if (!media_class)
		return;

	if (strcmp(media_class, "Audio/Sink") == 0)
		kind = "sink";
	else if (strcmp(media_class, "Audio/Source") == 0)
		kind = "source";
	else
		return;

	label = spa_dict_lookup(props, "node.description");
	if (!label)
		label = spa_dict_lookup(props, "node.nick");
	if (!label)
		label = spa_dict_lookup(props, "node.name");
	if (!label)
		label = "";

	emit_device(kind, id, label);
}

static const struct pw_registry_events registry_events = {
	PW_VERSION_REGISTRY_EVENTS,
	.global = on_global,
};

static void on_core_done(void *data, uint32_t id, int seq)
{
	struct app *app = data;

	if (id == PW_ID_CORE && seq == app->sync_seq)
		pw_main_loop_quit(app->loop);
}

static const struct pw_core_events core_events = {
	PW_VERSION_CORE_EVENTS,
	.done = on_core_done,
};

int main(int argc, char *argv[])
{
	struct app app = { 0 };

	pw_init(&argc, &argv);

	app.loop = pw_main_loop_new(NULL);
	app.context = pw_context_new(pw_main_loop_get_loop(app.loop), NULL, 0);
	if (!app.context) {
		fprintf(stderr, "pw_context_new failed\n");
		return 1;
	}

	app.core = pw_context_connect(app.context, NULL, 0);
	if (!app.core) {
		fprintf(stderr, "pw_context_connect failed\n");
		return 1;
	}

	pw_core_add_listener(app.core, &app.core_listener, &core_events, &app);

	app.registry = pw_core_get_registry(app.core, PW_VERSION_REGISTRY, 0);
	pw_registry_add_listener(app.registry, &app.registry_listener,
				 &registry_events, &app);

	app.sync_seq = pw_core_sync(app.core, PW_ID_CORE, 0);
	pw_main_loop_run(app.loop);

	spa_hook_remove(&app.registry_listener);
	spa_hook_remove(&app.core_listener);
	pw_core_disconnect(app.core);
	pw_context_destroy(app.context);
	pw_main_loop_destroy(app.loop);
	pw_deinit();
	return 0;
}
