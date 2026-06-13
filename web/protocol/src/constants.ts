export const HostStreamID = "host" as const;
export const PlaybackRefStreamID = "playback-ref" as const;

export const MixerLocked = "LOCKED" as const;
export const MixerHold = "HOLD" as const;
export const MixerSwitch = "SWITCH" as const;
export const MixerSilence = "SILENCE" as const;

export type MixerState =
  | typeof MixerLocked
  | typeof MixerHold
  | typeof MixerSwitch
  | typeof MixerSilence;
