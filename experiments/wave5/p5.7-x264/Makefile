CC ?= gcc
CFLAGS ?= -Wall -Wextra -O2
PKG_CFLAGS := $(shell pkg-config --cflags x264 2>/dev/null)
PKG_LIBS := $(shell pkg-config --libs x264 2>/dev/null)

ifeq ($(strip $(PKG_LIBS)),)
$(error libx264-dev not installed — sudo apt install libx264-dev pkg-config)
endif

CFLAGS += $(PKG_CFLAGS)
LDFLAGS = $(PKG_LIBS)

.PHONY: all clean run

all: enc_test

enc_test: enc_test.c
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)

run: enc_test
	./enc_test

clean:
	rm -f enc_test
