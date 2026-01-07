// Re-export Wails-generated types for convenience
export type LogEntry = import('../../wailsjs/go/models').process.LogEntry;
export type ServiceConfig = import('../../wailsjs/go/models').main.ServiceConfig;
export type ServiceInfo = import('../../wailsjs/go/models').main.ServiceInfo;
export type Service = import('../../wailsjs/go/models').main.Service;

// Type aliases for backward compatibility
export type LogLevel = LogEntry['level'];
export type ServiceStatus = ServiceInfo['status'];
export type ServiceEnv = Record<string, string>;