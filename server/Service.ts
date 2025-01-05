import { MAX_LOGS } from "../constants";
import type { IProcessManager, LogEntry, ServiceStatus, ServiceEnv } from "./types";
import { broadcast } from "./api/socket";
import { readFileSync, writeFileSync } from "node:fs";
import { randomUUID } from "crypto";
import { mapValues } from "lodash-es";
import { DotnetService } from "./dotnet";

export interface ServiceConfig {
  name: string;
  path: string;
  env: ServiceEnv;
}

export class Service {
  id: string;
  private _config: ServiceConfig;
  status: ServiceStatus = "stopped";
  logs: LogEntry[] = [];
  url?: string;
  private processManager: IProcessManager;

  constructor(id: string, config: ServiceConfig) {
    this.id = id;
    this._config = config;
    this.processManager = new DotnetService(config.path, config.env);
    this.processManager.on("log", this.handleLog);
    this.processManager.on("url", this.handleUrl);
    this.processManager.on("statusChange", this.handleStatus);
  }

  get config(): ServiceConfig {
    return this._config;
  }

  set config(value: ServiceConfig) {
    this._config = value;
    this.processManager.updateConfig(value.path, value.env);
  }

  private broadcastStatus() {
    broadcast("statusUpdate", this.id, {
      status: this.status,
      url: this.url,
    });
  }

  setStatus(status: ServiceStatus) {
    this.status = status;
    this.broadcastStatus();
  }

  clearLogs() {
    this.logs = [];
    this.broadcastStatus();
  }

  addLog(log: LogEntry) {
    if (this.status === "initializing") {
      this.setStatus("running");
    }

    this.logs.push(log);
    if (this.logs.length > MAX_LOGS) {
      this.logs.shift();
    }

    broadcast("newLog", this.id, {
      log,
    });

    if (log.level === "ERR") {
      this.broadcastStatus();
    }
  }

  setUrl(url: string) {
    this.url = url;
    this.broadcastStatus();
  }

  private handleLog = (log: LogEntry) => {
    this.addLog(log);
  };

  private handleUrl = (url: string) => {
    this.setUrl(url);
  };

  private handleStatus = (status: ServiceStatus) => {
    this.setStatus(status);
  };

  public start() {
    return this.processManager.start();
  }

  public stop() {
    return this.processManager.stop();
  }
}

export const loadConfigs = () => {
  try {
    return JSON.parse(readFileSync("services.json", "utf-8")) as Record<
      string,
      ServiceConfig
    >;
  } catch {
    return {};
  }
};

export const reloadServices = async () => {
  const configs = loadConfigs();

  // Stop and remove services that no longer exist in config
  for (const id of Object.keys(services)) {
    if (!configs[id]) {
      const service = services[id];
      if (service.status === "running") {
        await service.stop();
      }
      delete services[id];
    }
  }

  // Update existing services and create new ones
  for (const [id, config] of Object.entries(configs)) {
    if (services[id]) {
      const service = services[id];
      service.config = config;
    } else {
      // Create new service
      services[id] = new Service(id, config);
    }
  }

  return services;
};

export const saveServices = (services: Record<string, Service>) => {
  const configs = mapValues(services, (service) => service.config);
  writeFileSync("services.json", JSON.stringify(configs, null, 2));
};

const configs = loadConfigs();
export const services = mapValues(
  configs,
  (config, id) => new Service(id, config)
);

export const addService = (config: ServiceConfig) => {
  const id = randomUUID();
  const service = new Service(id, {
    name: config.name,
    path: config.path,
    env: config.env || {},
  });
  services[id] = service;
  saveServices(services);
  return service;
};

export const updateService = (id: string, config: ServiceConfig) => {
  const service = services[id];
  if (!service) {
    throw new Error("Service not found");
  }

  service.config = config;
  saveServices(services);
  return service;
};

export const editService = (id: string, config: Partial<ServiceConfig>) => {
  const service = services[id];
  if (!service) {
    throw new Error("Service not found");
  }

  Object.assign(service.config, config);
  saveServices(services);
  return service;
};
