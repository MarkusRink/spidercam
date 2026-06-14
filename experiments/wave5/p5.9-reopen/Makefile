CC ?= gcc
CFLAGS ?= -Wall -Wextra -O2 -pthread
PW_PREFIX ?=

ifneq ($(PW_PREFIX),)
CFLAGS += -I$(PW_PREFIX)/include/pipewire-0.3 -I$(PW_PREFIX)/include/spa-0.2 -D_REENTRANT
LDFLAGS = -pthread -lm -l:libpipewire-0.3.so.0
else
CFLAGS += $(shell pkg-config --cflags libpipewire-0.3)
LDFLAGS = -pthread -lm $(shell pkg-config --libs libpipewire-0.3)
endif

SRCS = reopen_test.c sp_capture.c sp_ring.c
OBJS = $(SRCS:.c=.o)

reopen_test: $(OBJS)
	$(CC) $(CFLAGS) -o $@ $(OBJS) $(LDFLAGS)

%.o: %.c sp_capture.h sp_ring.h
	$(CC) $(CFLAGS) -c -o $@ $<

clean:
	rm -f reopen_test $(OBJS)

.PHONY: clean
