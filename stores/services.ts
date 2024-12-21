import { defineStore } from "pinia";
import type { ServiceInfo, WebSocketMessage } from "~/server/types";
import type { ClientServiceInfo, ClientLogEntry } from "~/types/client";

function parseReadLogs(serviceName: string): Set<string> {
  const stored = localStorage.getItem(`readLogs_${serviceName}`);
  return new Set(JSON.parse(stored || "[]"));
}

export const useServicesStore = defineStore("services", () => {
  const services = ref<ClientServiceInfo[]>([]);
  const selectedService = ref<ClientServiceInfo | null>(null);
  const ws = ref<WebSocket>();
  const readLogs = ref<Record<string, Set<string>>>({});

  function isLogRead(serviceName: string, timestamp: string) {
    if (!readLogs.value[serviceName]) {
      readLogs.value[serviceName] = parseReadLogs(serviceName);
    }
    return readLogs.value[serviceName].has(timestamp);
  }

  function getUnreadErrorCount(serviceName: string) {
    const service = services.value.find((s) => s.name === serviceName);
    if (!service) return 0;
    return service.logs.filter(
      (log) => log.level === "ERR" && !isLogRead(serviceName, log.timestamp)
    ).length;
  }

  function markLogAsRead(serviceName: string, timestamp: string) {
    if (!readLogs.value[serviceName]) {
      readLogs.value[serviceName] = parseReadLogs(serviceName);
    }
    readLogs.value[serviceName].add(timestamp);
    localStorage.setItem(
      `readLogs_${serviceName}`,
      JSON.stringify(Array.from(readLogs.value[serviceName]))
    );
  }

  function mapToClientServiceInfo(service: ServiceInfo): ClientServiceInfo {
    return {
      ...service,
      logs: service.logs.map((log) => ({ ...log, read: false })),
      unreadErrors: 0,
    };
  }

  function addServices(newServices: ServiceInfo[]) {
    services.value.push(...newServices.map(mapToClientServiceInfo));
  }

  async function addService(path: string) {
    const { service } = await $fetch<{
      success: boolean;
      service: ServiceInfo;
    }>("/api/service/add", {
      method: "POST",
      body: { path },
    });
    const clientService = mapToClientServiceInfo(service);
    services.value.push(clientService);
    if (!selectedService.value) {
      selectedService.value = clientService;
    }
  }

  function setupWebSocket() {
    ws.value = new WebSocket(`ws://${window.location.host}/api/socket`);

    ws.value.onmessage = (event) => {
      const msg = JSON.parse(event.data) as WebSocketMessage;

      switch (msg.type) {
        case "statusUpdate": {
          const service = services.value.find(
            (s) => s.name === msg.data.service
          );
          if (service) {
            Object.assign(service, msg.data);
            if (selectedService.value?.name === msg.data.service) {
              selectedService.value = service as ClientServiceInfo;
            }
          }
          break;
        }
        case "newLog": {
          const service = services.value.find(
            (s) => s.name === msg.data.service
          );
          if (service) {
            const clientLog: ClientLogEntry = {
              ...msg.data.log,
              read: false,
            };
            service.logs.push(clientLog);
          }
          break;
        }
        case "initialState": {
          services.value = msg.data.services.map((service) => ({
            ...service,
            logs: service.logs.map((log) => ({ ...log, read: false })),
          }));
          if (!selectedService.value && services.value.length > 0) {
            selectedService.value = services.value[0];
          }
          break;
        }
      }
    };

    ws.value.onclose = () => {
      console.log("WebSocket connection closed, reconnecting...");
      setTimeout(setupWebSocket, 1000);
    };
  }

  async function startService(service: ClientServiceInfo) {
    const serviceRef = services.value.find((s) => s.name === service.name);
    if (serviceRef) {
      serviceRef.status = "starting";
    }

    try {
      await $fetch("/api/service/start", {
        method: "POST",
        body: { service: service.path },
      });
    } catch (error) {
      if (serviceRef) {
        serviceRef.status = "error";
      }
      console.error("Failed to start service:", error);
    }
  }

  async function stopService(service: ClientServiceInfo) {
    const serviceRef = services.value.find((s) => s.name === service.name);
    if (serviceRef) {
      serviceRef.status = "stopping";
    }

    try {
      await $fetch("/api/service/stop", {
        method: "POST",
        body: { service: service.path },
      });
    } catch (error) {
      if (serviceRef) {
        serviceRef.status = "error";
      }
      console.error("Failed to stop service:", error);
    }
  }

  function selectService(service: ClientServiceInfo) {
    selectedService.value = service;
  }

  onMounted(() => {
    setupWebSocket();
  });

  return {
    services,
    selectedService,
    startService,
    stopService,
    selectService,
    addServices,
    isLogRead,
    markLogAsRead,
    getUnreadErrorCount,
    addService,
  };
});
