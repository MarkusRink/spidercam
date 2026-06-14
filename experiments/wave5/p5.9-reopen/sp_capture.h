#ifndef SP_CAPTURE_H
#define SP_CAPTURE_H

typedef struct sp_capture sp_capture;

sp_capture *sp_capture_open(const char *mic_id, const char *sink_id,
			    int sample_rate);
void sp_capture_close(sp_capture *c);
int sp_capture_read_mic(sp_capture *c, float *buf, int frames);
int sp_capture_read_monitor(sp_capture *c, float *buf, int frames);

#endif
