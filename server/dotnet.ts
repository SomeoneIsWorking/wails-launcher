import { spawn, ChildProcess, exec } from "child_process";
import { promisify } from "util";
import type { LogEntry } from "./types";
import { Service } from "./Service";

const execAsync = promisify(exec);

export class DotnetService {
  async findProcess(
    servicePath: string
  ): Promise<{ pid: string; cmd: string }[]> {
    try {
      // Find all dotnet processes and their working directories in one command
      const { stdout } = await execAsync(`lsof -c dotnet -a -d cwd | grep cwd`);

      const results: { pid: string; cmd: string }[] = [];

      stdout.split("\n").forEach((line) => {
        if (!line.trim()) return;
        const [_, pid, __, ___, ____, _____, ______, _______, path] = line
          .trim()
          .split(/\s+/);
        if (path.toLowerCase().includes(servicePath.toLowerCase())) {
          results.push({
            pid,
            cmd: servicePath,
          });
        }
      });

      return results;
    } catch (error: any) {
      if (error.code === 1 && !error.stdout && !error.stderr) {
        return [];
      }
      console.error(`Process search error for ${servicePath}:`, error);
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
    } catch (error) {
      console.error(`Error killing process ${pid}:`, error);
    }
  }

  async cleanup(servicePath: string) {
    const runningProcesses = await this.findProcess(servicePath);

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

  async startService(service: Service) {
    if (service.getProcess()) {
      console.log(`Service ${service.name} is already running`);
      return null;
    }

    await this.cleanup(service.path);
    service.update({
      logs: [],
      status: "starting",
    });

    try {
      const proc = spawn("dotnet", ["run"], {
        cwd: service.path,
        shell: true,
        stdio: ["ignore", "pipe", "pipe"],
        env: { ...process.env, DOTNET_ENVIRONMENT: "Development" },
      });

      if (!proc.pid) {
        service.setStatus("error");
        return null;
      }

      service.setProcess(proc);
      service.setStatus("initializing");

      proc.stdout.on("data", (data) => {
        const output = data.toString().trim();
        if (!output) return;

        const urlMatch = output.match(/Now listening on:\s*(\S+)/i);
        if (urlMatch) {
          service.setUrl(urlMatch[1]);
        }

        const log = this.parseLog(output);
        if (log) {
          service.addLog(log);
        }
      });

      proc.stderr.on("data", (data) => {
        service.addLog({
          timestamp: new Date().toISOString(),
          level: "ERR",
          message: data.toString(),
          raw: data.toString(),
        });
      });

      proc.on("close", (code) => {
        service.setProcess(undefined);
        service.setStatus(code === 0 ? "stopped" : "error");
      });

      return proc;
    } catch (error) {
      console.error(`Error starting service ${service.name}:`, error);
      service.setStatus("error");
      return null;
    }
  }

  async stopService(service: Service) {
    service.setStatus("stopping");

    const proc = service.getProcess();
    if (proc) {
      proc.kill("SIGTERM");
      await new Promise((resolve) => setTimeout(resolve, 1000));
      if (!proc.killed) {
        proc.kill("SIGKILL");
      }
      service.setProcess(undefined);
    }

    await this.cleanup(service.path);
    service.setStatus("stopped");
  }
}

export const dotnetService = new DotnetService();
