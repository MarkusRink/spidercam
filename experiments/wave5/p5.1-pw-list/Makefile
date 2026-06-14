CC ?= gcc
CFLAGS += -Wall -Wextra -O2
PW_PREFIX ?=

ifneq ($(PW_PREFIX),)
CFLAGS += -I$(PW_PREFIX)/include/pipewire-0.3 -I$(PW_PREFIX)/include/spa-0.2 -D_REENTRANT
LDFLAGS = -pthread -lm -l:libpipewire-0.3.so.0
else
CFLAGS += $(shell pkg-config --cflags libpipewire-0.3)
LDFLAGS += $(shell pkg-config --libs libpipewire-0.3)
endif

pw_list: pw_list.c
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

.PHONY: clean
clean:
	rm -f pw_list
