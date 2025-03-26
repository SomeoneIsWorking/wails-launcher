import type { LogEntry as ServerLogEntry, ServiceInfo as ServerServiceInfo } from '~/server/types'

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
