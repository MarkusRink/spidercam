import { HostStreamID } from "./constants.js";

export interface ScoreWeights {
  level: number;
  snr: number;
  vad: number;
  priority: number;
  echoPenalty: number;
}

export interface HostConfig {
  defaultVideoId: string;
  defaultAudioId: string;
  silenceScoreThreshold: number;
  videoHoldMs: number;
  audioHoldMs: number;
  minHoldAfterSwitchMs: number;
  crossfadeMs: number;
  switchMargin: number;
  emergencyScoreRatio: number;
  scoreSmoothingAlpha: number;
  scoreWeights: ScoreWeights;
  hostPriority: number;
  targetSpeechDbfs: number;
  calibrationGainClampDb: [number, number];
  vadSnrOnDb: number;
  vadSnrOffDb: number;
  vadHangoverMs: number;
  gateAttenuationDb: number;
  referenceVadOnDbfs: number;
  referenceVadOffDbfs: number;
  referenceDuckDb: number;
  referenceDelayMs: number;
  loopDelayScaleMaxMs: number;
  loopDelayWindowMs: number;
  loopDelayLagSearchMs: number;
  loopDelayAnalysisMs: number;
  loopDelayPublishMs: number;
  loopDelayMinSamples: number;
  loopDelayMinPeak: number;
  loopDelayStaleMs: number;
}

export const DefaultScoreWeights: ScoreWeights = {
  level: 0.35,
  snr: 0.35,
  vad: 0.25,
  priority: 0.2,
  echoPenalty: 0.35,
};

export const DefaultHostConfig: HostConfig = {
  defaultVideoId: HostStreamID,
  defaultAudioId: HostStreamID,
  silenceScoreThreshold: 0.15,
  videoHoldMs: 300,
  audioHoldMs: 400,
  minHoldAfterSwitchMs: 600,
  crossfadeMs: 100,
  switchMargin: 1.0,
  emergencyScoreRatio: 3.0,
  scoreSmoothingAlpha: 0.1,
  scoreWeights: DefaultScoreWeights,
  hostPriority: 1.0,
  targetSpeechDbfs: -20,
  calibrationGainClampDb: [-12, 18],
  vadSnrOnDb: 7,
  vadSnrOffDb: 3,
  vadHangoverMs: 150,
  gateAttenuationDb: 12,
  referenceVadOnDbfs: -35,
  referenceVadOffDbfs: -45,
  referenceDuckDb: -12,
  referenceDelayMs: 0,
  loopDelayScaleMaxMs: 100,
  loopDelayWindowMs: 500,
  loopDelayLagSearchMs: 300,
  loopDelayAnalysisMs: 250,
  loopDelayPublishMs: 3000,
  loopDelayMinSamples: 3,
  loopDelayMinPeak: 0.25,
  loopDelayStaleMs: 300_000,
};
