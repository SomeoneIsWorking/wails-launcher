import { spawn, exec, type ChildProcess } from "child_process";
import { promisify } from "util";
import { EventEmitter } from "events";
import type { IProcessManager, LogEntry, LogLevel, ServiceEnv } from "./types";

const execAsync = promisify(exec);

export class DotnetService extends EventEmitter implements IProcessManager {
  private currentProcess?: ChildProcess;
  private path: string;
  private env: ServiceEnv;

  constructor(path: string, env: ServiceEnv) {
    super();
    this.path = path;
    this.env = env;
  }

  updateConfig(path: string, env: ServiceEnv) {
    this.path = path;
    this.env = env;
  }

  private emitLog(level: LogLevel, message: string, raw?: string) {
    this.emit("log", {
      timestamp: new Date().toISOString(),
      level,
      message,
      raw: raw || message,
    });
  }

  private emitError(error: Error | string, context?: string) {
    const message = error instanceof Error ? error.message : error;
    const raw = error instanceof Error ? error.toString() : error;
    this.emitLog("ERR", context ? `${context}: ${message}` : message, raw);
  }

  private emitStateChange(status: string) {
    this.emit("statusChange", status);
    this.emitLog("INF", `Service ${status}`);
  }

  async findProcess() {
    try {
      // Find all dotnet processes and their working directories in one command
      const { stdout } = await execAsync(`lsof -c dotnet -a -d cwd | grep cwd`);

      const results: { pid: string; cmd: string }[] = [];

      stdout.split("\n").forEach((line) => {
        if (!line.trim()) {
          return;
        }
        const [_, pid, __, ___, ____, _____, ______, _______, path] = line
          .trim()
          .split(/\s+/);
        if (path.toLowerCase().includes(this.path.toLowerCase())) {
          results.push({
            pid,
            cmd: this.path,
          });
        }
      });

      return results;
    } catch (error: any) {
      if (error.code === 1 && !error.stdout && !error.stderr) {
        return [];
      }
      console.error(`Process search error for ${this.path}:`, error);
      this.emitError(error, "Process search error");
      return [];
    }
  }

  async killProcess(pid: string): Promise<void> {
    try {
      await execAsync(`kill ${pid}`);
      await new Promise((resolve) => setTimeout(resolve, 1000));

      try {
        await execAsync(`ps -p ${pid}`);
        await execAsync(`kill -9 ${pid}`);
      } catch {
        console.log(`Process ${pid} terminated gracefully`);
      }
    } catch (error: any) {
      console.error(`Error killing process ${pid}:`, error);
      this.emitError(error, `Failed to kill process ${pid}`);
    }
  }

  async cleanup() {
    const runningProcesses = await this.findProcess();

    for (const proc of runningProcesses) {
      console.log(`Killing process ${proc.pid} (${proc.cmd})`);
      await this.killProcess(proc.pid);
    }
  }

  parseLog(entry: string): LogEntry | null {
    if (entry.includes("NETSDK1138")) {
      return null;
    }
    const match = entry.match(/^\[(.*?)\s+(ERR|INF|WARN|DBG).*?\]\s+(.*)/m);
    if (!match)
      return {
        timestamp: new Date().toISOString(),
        level: "INF",
        message: entry,
        raw: entry,
      };

    return {
      timestamp: match[1],
      level: match[2] as LogEntry["level"],
      message: match[3],
      raw: entry,
    };
  }

  async start() {
    if (this.currentProcess) {
      const error = new Error("Service is already running");
      this.emitError(error);
      throw error;
    }

    await this.cleanup();
    this.emitStateChange("starting");

    try {
      const proc = this.spawn();
      this.checkPid(proc);
      this.currentProcess = proc;
      this.emitStateChange("initializing");
      proc.stdout.on("data", (data) => this.onData(data));
      proc.stderr.on("data", (data) => this.onErrorData(data));
      proc.on("close", (code) => this.onClose(code));
    } catch (error: any) {
      console.error(`Error starting service:`, error);
      this.emitStateChange("error");
      this.emitError(error, "Failed to start service");
    }
  }

  private spawn() {
    try {
      const env = {
        ...process.env,
        DOTNET_ENVIRONMENT: "Development",
        ...this.env,
      };
      this.emitLog(
        "INF",
        `Starting service in ${this.path} with env: ${JSON.stringify(env)}`
      );
      return spawn("dotnet", ["run"], {
        cwd: this.path,
        shell: true,
        stdio: ["ignore", "pipe", "pipe"],
        env,
      });
    } catch (error: any) {
      this.emitStateChange("error");
      this.emitError(error, "Failed to start service");
      throw error;
    }
  }

  private checkPid(proc: ChildProcess) {
    if (!proc.pid) {
      this.emitStateChange("error");
      this.emitError("Failed to start process - no PID assigned");
      throw new Error("Failed to start process - no PID assigned");
    }
  }

  onData(data: any) {
    const output = data.toString().trim();
    if (!output) return;

    const urlMatch = output.match(/Now listening on:\s*(\S+)/i);
    if (urlMatch) {
      this.emit("url", urlMatch[1]);
    }

    const log = this.parseLog(output);
    if (log) {
      this.emit("log", log);
    }
  }

  private onErrorData(data: any) {
    this.emitLog("ERR", data.toString());
  }

  private onClose(code: number | null) {
    this.currentProcess = undefined;
    const status = code === 0 ? "stopped" : "error";
    this.emitStateChange(status);
    this.emitLog(
      status === "error" ? "ERR" : "INF",
      `Service ${status}${code !== null ? ` with exit code ${code}` : ""}`,
      `Process exited${code !== null ? ` (code ${code})` : ""}`
    );
  }

  async stop() {
    this.emitStateChange("stopping");

    if (!this.currentProcess) {
      this.emitStateChange("stopped");
      return;
    }

    this.currentProcess.kill();
  }
}
