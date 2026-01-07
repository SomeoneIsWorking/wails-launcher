import type { LogEntry as ServerLogEntry, ServiceInfo as ServerServiceInfo } from './service'

export interface ClientLogEntry extends ServerLogEntry {
  read?: boolean
}

export interface ClientServiceInfo extends Omit<ServerServiceInfo, 'logs'> {
  logs: ClientLogEntry[];
  unreadErrors?: number;
}

export interface ScrollPosition {
  topIndex: number;
  offset: number;
}

export interface ClientGroupInfo {
  name: string;
  env: Record<string, string>;
  services: Record<string, ClientServiceInfo>;
}
