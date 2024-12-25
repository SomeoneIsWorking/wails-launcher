import { ChildProcess } from "child_process";
import type { LogEntry, ServiceStatus } from "./types";
import { broadcast } from "./api/socket";
import { readFileSync, writeFileSync } from "node:fs";
import { randomUUID } from "crypto";
import { mapValues } from "lodash-es";

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
  private process?: ChildProcess;

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

  update(
    updates: Partial<{
      status: ServiceStatus;
      logs: LogEntry[];
      url: string;
    }>
  ) {
    Object.assign(this, updates);
    this.broadcastStatus();
  }

  setStatus(status: ServiceStatus) {
    this.update({ status });
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

  setProcess(proc: ChildProcess | undefined) {
    this.process = proc;
  }

  getProcess() {
    return this.process;
  }

  clearLogs() {
    this.update({
      logs: [],
    });
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
