import { ref, computed, onMounted } from 'vue'
import { mapValues } from "lodash-es";
import { defineStore } from "pinia";
import { MAX_LOGS } from "@/constants";
import type { ServiceConfig, ServiceInfo } from "@/types/service";
import type { ClientServiceInfo, ClientLogEntry, ScrollPosition } from "@/types/client";
import { GetServices, AddService, UpdateService, StartService, StopService, ClearLogs, ReloadServices } from '../../wailsjs/go/main/App.js'
import { EventsOn } from '../../wailsjs/runtime/runtime.js'

function parseReadLogs(serviceName: string): Set<string> {
  const stored = localStorage.getItem(`readLogs_${serviceName}`);
  return new Set(JSON.parse(stored || "[]"));
}

export const useServicesStore = defineStore("services", () => {
  const services = ref<Record<string, ClientServiceInfo>>({});
  const selectedServiceId = ref<string | null>(null);
  const scrollPositions = ref<Record<string, ScrollPosition | undefined>>({});
  const selectedService = computed(() =>
    selectedServiceId.value ? services.value[selectedServiceId.value] : null
  );
  const readLogs = ref<Record<string, Set<string>>>({});

  function mapToClientServiceInfo(service: ServiceInfo): ClientServiceInfo {
    return {
      ...service,
      logs: service.logs.map((log) => ({ ...log, read: false })),
      unreadErrors: 0,
    };
  }

  function isLogRead(id: string, timestamp: string) {
    if (!readLogs.value[id]) {
      readLogs.value[id] = parseReadLogs(id);
    }
    return readLogs.value[id].has(timestamp);
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

  function getUnreadErrorCount(id: string) {
    const service = services.value[id];
    if (!service) return 0;
    return service.logs.filter(
      (log) => log.level === "ERR" && !isLogRead(id, log.timestamp)
    ).length;
  }

  async function loadServices() {
    const newServices = await GetServices();
    addServices(newServices);
  }

  function addServices(newServices: Record<string, ServiceInfo>) {
    services.value = {
      ...services.value,
      ...mapValues(newServices, mapToClientServiceInfo),
    };
  }

  async function addService(config: ServiceConfig) {
    const service = await AddService(config);
    const id = service.ID;
    const clientService = mapToClientServiceInfo({
      name: service.Config.name,
      path: service.Config.path,
      status: service.Status,
      url: service.URL,
      logs: service.Logs,
      env: service.Config.env,
    });
    services.value[id] = clientService;
    if (!selectedService.value) {
      selectService(id);
    }
  }

  async function updateService(id: string, config: ServiceConfig) {
    await UpdateService(id, config);
    // Update local
    services.value[id] = {
      ...services.value[id],
      name: config.name,
      path: config.path,
      env: config.env,
    };
  }

  function setupEvents() {
    EventsOn("serviceEvent", (event: any) => {
      const msg = event;
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
            if (service.logs.length > MAX_LOGS) {
              service.logs.shift();
            }
          }
          break;
        }
      }
    });
  }

  async function startService(id: string) {
    const serviceRef = services.value[id];
    if (serviceRef) {
      serviceRef.status = "starting";
    }

    try {
      await StartService(id);
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
      await StopService(id);
    } catch (error) {
      serviceRef.status = "error";
      console.error("Failed to stop service:", error);
    }
  }

  async function restartService(id: string) {
    const serviceRef = services.value[id];
    if (!serviceRef) {
      throw new Error(`Service ${id} not found`);
    }

    try {
      await stopService(id);
      await startService(id);
    } catch (error) {
      serviceRef.status = "error";
      console.error("Failed to restart service:", error);
    }
  }

  function selectService(id: string) {
    if (!services.value[id]) {
      throw new Error(`Service ${id} not found`);
    }
    selectedServiceId.value = id;
  }

  onMounted(async () => {
    setupEvents();
    await loadServices();
  });

  async function clearLogs(id: string) {
    await ClearLogs(id);
    services.value[id].logs = [];
  }

  async function reloadConfig() {
    await ReloadServices();
    await loadServices();
  }

  function saveScrollPosition(serviceId: string, position: ScrollPosition | undefined) {
    scrollPositions.value[serviceId] = position;
  }

  function getScrollPosition(serviceId: string): ScrollPosition | undefined {
    return scrollPositions.value[serviceId];
  }

  return {
    services,
    selectedService,
    selectedServiceId,
    startService,
    stopService,
    restartService,
    selectService,
    addServices,
    isLogRead,
    markLogAsRead,
    getUnreadErrorCount,
    addService,
    updateService,
    clearLogs,
    reloadConfig,
    loadServices,
    saveScrollPosition,
    getScrollPosition,
  };
});

