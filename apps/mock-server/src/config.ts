export const PARTICIPANT_HOST = "0.0.0.0";
export const HOST_HOST = "127.0.0.1";

export const PARTICIPANT_PORT = Number(
  process.env.SPIDERCAM_PARTICIPANT_PORT ?? 1234,
);
export const HOST_PORT = Number(process.env.SPIDERCAM_HOST_PORT ?? 1235);

export const TICK_MS = 20;
export const HOST_STATE_HZ = 50;
export const PREVIEW_FPS = 15;
export const PARTICIPANT_VIEW_MAX_HZ = 10;
export const LOOP_DELAY_PUBLISH_MS = 3000;

export const PREVIEW_WIDTH = 1280;
export const PREVIEW_HEIGHT = 720;
export const PREVIEW_CODEC = "avc1.42E01E";
