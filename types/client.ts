import type { LogEntry as ServerLogEntry, ServiceInfo as ServerServiceInfo } from '~/server/types'

export interface ClientLogEntry extends ServerLogEntry {
  read?: boolean
}

export interface ClientServiceInfo extends Omit<ServerServiceInfo, 'logs'> {
  id: string;
  logs: ClientLogEntry[];
  unreadErrors?: number;
}
