import { mapValues } from "lodash-es";
import { defineStore } from "pinia";
import type { ServiceConfig } from "~/server/Service";
import type { ServiceInfo, WebSocketMessage } from "~/server/types";
import type { ClientServiceInfo, ClientLogEntry } from "~/types/client";

function parseReadLogs(serviceName: string): Set<string> {
  const stored = localStorage.getItem(`readLogs_${serviceName}`);
  return new Set(JSON.parse(stored || "[]"));
}

export const useServicesStore = defineStore("services", () => {
  const services = ref<Record<string, ClientServiceInfo>>({});
  const selectedServiceId = ref<string | null>(null);
  const selectedService = computed(() =>
    selectedServiceId.value ? services.value[selectedServiceId.value] : null
  );
  const ws = ref<WebSocket>();
  const readLogs = ref<Record<string, Set<string>>>({});

  function isLogRead(id: string, timestamp: string) {
    if (!readLogs.value[id]) {
      readLogs.value[id] = parseReadLogs(id);
    }
    return readLogs.value[id].has(timestamp);
  }

  function getUnreadErrorCount(id: string) {
    const service = services.value[id];
    if (!service) return 0;
    return service.logs.filter(
      (log) => log.level === "ERR" && !isLogRead(id, log.timestamp)
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

  function addServices(newServices: Record<string, ServiceInfo>) {
    services.value = {
      ...services.value,
      ...mapValues(newServices, mapToClientServiceInfo),
    };
  }

  async function addService(config: ServiceConfig) {
    const { id, service } = await $fetch("/api/services", {
      method: "POST",
      body: config,
    });

    const clientService = mapToClientServiceInfo(service);
    services.value[id] = clientService;
    if (!selectedService.value) {
      selectService(id);
    }
  }

  async function updateService(id: string, config: ServiceConfig) {
    const { service } = await $fetch<{
      service: ServiceInfo;
    }>(`/api/services/${id}`, {
      method: "PATCH",
      body: config,
    });

    services.value[id] = {
      ...service,
      logs: services.value[id].logs,
    };
  }

  function setupWebSocket() {
    ws.value = new WebSocket(`ws://${window.location.host}/api/socket`);

    ws.value.onmessage = (event) => {
      const msg = JSON.parse(event.data) as WebSocketMessage;

      switch (msg.type) {
        case "statusUpdate": {
          const service = services.value[msg.serviceId];
          if (service) {
            Object.assign(service, msg.data);
          }
          break;
        }
        case "newLog": {
          const service = services.value[msg.serviceId];
          if (service) {
            const clientLog: ClientLogEntry = {
              ...msg.data.log,
              read: false,
            };
            service.logs.push(clientLog);
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

  async function startService(id: string) {
    const serviceRef = services.value[id];
    if (serviceRef) {
      serviceRef.status = "starting";
    }

    try {
      await $fetch(`/api/services/${id}/start`, {
        method: "POST",
      });
    } catch (error) {
      if (serviceRef) {
        serviceRef.status = "error";
      }
      console.error("Failed to start service:", error);
    }
  }

  async function stopService(id: string) {
    const serviceRef = services.value[id];
    if (!serviceRef) {
      throw new Error(`Service ${id} not found`);
    }
    serviceRef.status = "stopping";

    try {
      await $fetch(`/api/services/${id}/stop`, {
        method: "POST",
      });
    } catch (error) {
      serviceRef.status = "error";
      console.error("Failed to stop service:", error);
    }
  }

  function selectService(id: string) {
    if (!services.value[id]) {
      throw new Error(`Service ${id} not found`);
    }
    selectedServiceId.value = id;
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
    updateService,
  };
});
