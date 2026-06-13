import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import type { CaptureDevices, RoomState } from "@spidercam/protocol";

const require = createRequire(import.meta.url);

interface HostStateFixture {
  state: RoomState;
}

export function loadRoutingState(): RoomState {
  const fixture =
    require("@spidercam/test-fixtures/host-state/routing.json") as HostStateFixture;
  return structuredClone(fixture.state);
}

export function loadCaptureDevices(): CaptureDevices {
  return require("@spidercam/test-fixtures/capture-devices.json") as CaptureDevices;
}

export function loadPreviewKeyframe(): Buffer {
  const path =
    require.resolve("@spidercam/test-fixtures/preview/keyframe.h264");
  return readFileSync(path);
}
