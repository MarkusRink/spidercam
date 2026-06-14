#ifndef SP_DEVICES_H
#define SP_DEVICES_H

typedef struct sp_device {
	char id[32];
	char label[128];
} sp_device;

int sp_list_sources(sp_device *out, int max);
int sp_list_sinks(sp_device *out, int max);

#endif
