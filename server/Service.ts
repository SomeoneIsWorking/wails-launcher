import { ChildProcess } from "child_process";
import type { IProcessManager, LogEntry, ServiceStatus } from "./types";
import { broadcast } from "./api/socket";
import { readFileSync, writeFileSync } from "node:fs";
import { randomUUID } from "crypto";
import { mapValues } from "lodash-es";
import { DotnetService } from "./dotnet";

export interface ServiceConfig {
  name: string;
  path: string;
  env: Record<string, string>;
}

export class Service {
  id: string;
  name: string;
  status: ServiceStatus = "stopped";
  logs: LogEntry[] = [];
  url?: string;
  env: Record<string, string>;
  path: string;
  private processManager?: IProcessManager;

  constructor(id: string, config: ServiceConfig) {
    this.id = id;
    this.name = config.name;
    this.path = config.path;
    this.env = config.env;
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

  setProcessManager(manager: IProcessManager) {
    if (this.processManager) {
      // Cleanup old listeners
      this.processManager.off("log", this.handleLog);
      this.processManager.off("url", this.handleUrl);
      this.processManager.off("statusChange", this.handleStatus);
    }

    this.processManager = manager;

    // Set up new listeners
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
    return this.getProcessManager().start();
  }

  public stop() {
    return this.getProcessManager().stop();
  }

  getProcessManager() {
    if (!this.processManager) {
      this.processManager = new DotnetService(this);
      this.processManager.on("log", this.handleLog);
      this.processManager.on("url", this.handleUrl);
      this.processManager.on("statusChange", this.handleStatus);
    }

    return this.processManager;
  }

  toConfig(): ServiceConfig {
    return {
      name: this.name,
      path: this.path,
      env: this.env,
    };
  }
}

export const loadServices = () => {
  try {
    const configs = JSON.parse(
      readFileSync("services.json", "utf-8")
    ) as Record<string, ServiceConfig>;
    return mapValues(configs, (config, id) => new Service(id, config));
  } catch {
    return {};
  }
};

export const saveServices = (services: Record<string, Service>) => {
  const configs = mapValues(services, (service) => service.toConfig());
  writeFileSync("services.json", JSON.stringify(configs, null, 2));
};

export const services = loadServices();

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

  Object.assign(service, config);
  saveServices(services);
  return service;
};

export const editService = (id: string, config: Partial<ServiceConfig>) => {
  const service = services[id];
  if (!service) {
    throw new Error("Service not found");
  }

  Object.assign(service, config);
  saveServices(services);
  return service;
};
