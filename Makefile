.PHONY: test vet build web sync-ui e2e check

WEB_HOST_DIST := web/host/dist/index.html
WEB_PARTICIPANT_DIST := web/participant/dist/index.html
EMBED_HOST_UI := internal/daemon/ui/host/index.html
EMBED_PARTICIPANT_UI := internal/daemon/ui/participant/index.html

NATIVE_CAPTURE_TAGS :=
ifeq ($(shell pkg-config --exists libpipewire-0.3 2>/dev/null && echo yes),yes)
NATIVE_CAPTURE_TAGS := -tags spidercam_native_capture
endif

test:
	go test ./... -count=1

e2e:
	go test ./test/e2e/... -tags=e2e -count=1

check: vet
	go test ./internal/... -race -count=1 $(NATIVE_CAPTURE_TAGS)
	$(MAKE) e2e

vet:
	go vet ./...

web:
	@if [ ! -f node_modules/.package-lock.json ]; then npm ci; fi
	npm run build --workspace=web/host --workspace=web/participant

$(WEB_HOST_DIST) $(WEB_PARTICIPANT_DIST):
	$(MAKE) web

sync-ui: $(WEB_HOST_DIST) $(WEB_PARTICIPANT_DIST)
	rsync -a --delete web/host/dist/ internal/daemon/ui/host/
	rsync -a --delete web/participant/dist/ internal/daemon/ui/participant/

$(EMBED_HOST_UI) $(EMBED_PARTICIPANT_UI): sync-ui

build: sync-ui
	go build -o bin/spidercamd ./cmd/spidercamd
