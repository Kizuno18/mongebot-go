// Shared TypeScript types matching Go backend structures.

export interface EngineMetrics {
  activeViewers: number;
  totalWorkers: number;
  segmentsFetched: number;
  bytesReceived: number;
  heartbeatsSent: number;
  adsWatched: number;
  uptime: number; // nanoseconds
  engineState: string;
  channel: string;
}

export interface ProxyInfo {
  host: string;
  port: string;
  username?: string;
  type: number;
  health: number;
  latency: number;
  lastUsed: string;
  useCount: number;
  country?: string;
}

export interface ProxyListResponse {
  proxies: ProxyInfo[];
  total: number;
  available: number;
  inUse: number;
}

export interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
  component?: string;
  worker?: string;
  fields?: Record<string, unknown>;
}

export interface AppConfig {
  version: number;
  engine: EngineConfig;
  api: APIConfig;
  logging: LogConfig;
  profiles: ProfileConfig[];
}

export interface EngineConfig {
  maxWorkers: number;
  restartInterval: string;
  heartbeatInterval: string;
  segmentFetchDelay: { min: string; max: string };
  gqlPulseInterval: { min: string; max: string };
  proxyTimeout: string;
  maxRetries: number;
  features: FeatureFlags;
}

export interface FeatureFlags {
  ads: boolean;
  chat: boolean;
  pubsub: boolean;
  segments: boolean;
  gqlPulse: boolean;
  spade: boolean;
}

export interface APIConfig {
  port: number;
  host: string;
}

export interface LogConfig {
  level: string;
  file: string;
  maxSizeMb: number;
  ringBuffer: number;
}

export interface ProfileConfig {
  id: string;
  name: string;
  platform: string;
  channel: string;
  active: boolean;
  maxWorkers?: number;
  features?: FeatureFlags;
}

export type EngineState = "stopped" | "starting" | "running" | "paused" | "stopping";
export type HealthStatus = "unknown" | "good" | "slow" | "dead";

export const healthColors: Record<HealthStatus, string> = {
  unknown: "text-gray-400",
  good: "text-emerald-400",
  slow: "text-amber-400",
  dead: "text-red-400",
};
