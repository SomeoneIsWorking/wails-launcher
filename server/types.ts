export type LogLevel = 'ERR' | 'INF' | 'WARN' | 'DBG';
export type ServiceStatus = 'stopped' | 'starting' | 'initializing' | 'running' | 'stopping' | 'error';

export interface LogEntry {
    timestamp: string;
    level: LogLevel;
    message: string;
    raw: string;
}

export interface ServiceInfo {
    name: string;
    path: string;
    status: ServiceStatus;
    url?: string;
    logs: LogEntry[];
}

export interface ServiceStatusUpdate {
    name: string;
    status: ServiceStatus;
}

export interface LogOptions {
  'statusUpdate': {
    service: string;
    status: ServiceStatus;
    url?: string;
  };
  'newLog': {
    service: string;
    log: LogEntry;
  };
  'initialState': {
    services: ServiceInfo[];
  };
}
export type LogType = keyof LogOptions;

export type WebSocketMessage = {
  [K in LogType]: { type: K; data: LogOptions[K] }
}[LogType]
