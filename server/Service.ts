import { ChildProcess } from "child_process";
import type { LogEntry, ServiceStatus } from "./types";
import { broadcast } from "./api/socket";
import { sep } from "path";
import { readFileSync, writeFileSync } from "node:fs";

export class Service {
  name: string;
  status: ServiceStatus = "stopped";
  logs: LogEntry[] = [];
  url?: string;
  private process?: ChildProcess;

  constructor(public path: string) {
    this.name = path.split(sep).pop()!;
  }

  private broadcastStatus() {
    broadcast("statusUpdate", {
      service: this.name,
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

    broadcast("newLog", {
      service: this.name,
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
}

export const loadServicePaths = () => {
  try {
    return JSON.parse(readFileSync("services.json", "utf-8")) as string[];
  } catch {
    return [] as string[];
  }
};

export const saveServicePaths = (paths: string[]) => {
  writeFileSync("services.json", JSON.stringify(paths, null, 2));
};

export const services = new Map<string, Service>(
  loadServicePaths().map((path) => {
    return [path, new Service(path)];
  })
);

export const addService = (filepath: string) => {
  const service = new Service(filepath);
  services.set(filepath, service);
  const paths = loadServicePaths();
  paths.push(filepath);
  saveServicePaths(paths);
  return service;
};
