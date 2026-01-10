import { ref, computed, onMounted } from 'vue'
import { defineStore } from "pinia";
import { MAX_LOGS } from "@/constants";
import type { ServiceConfig, ServiceInfo } from "@/types/service";
import type { ClientServiceInfo, ClientLogEntry, ScrollPosition, ClientGroupInfo } from "@/types/client";
import { GetServices, GetGroups, AddGroup, UpdateGroup, AddServiceToGroup, UpdateServiceInGroup, ImportSLN, ImportProject, AddService, UpdateService, StartService, StopService, ClearLogs, ReloadServices, DeleteService, StartGroup, Browse } from '../../wailsjs/go/main/App.js'
import { EventsOn } from '../../wailsjs/runtime/runtime.js'
import { process } from 'wailsjs/go/models.js';

function parseReadLogs(serviceName: string): Set<string> {
  const stored = localStorage.getItem(`readLogs_${serviceName}`);
  return new Set(JSON.parse(stored || "[]"));
}

function transformLogEntry(log: process.LogEntry): ClientLogEntry {
  const lines = log.message.split(/\n|\. /);
  return {
    ...log,
    read: false,
    lines: lines,
    height: lines.length * 20
  };
}

export const useServicesStore = defineStore("services", () => {
  const services = ref<Record<string, ClientServiceInfo>>({});
  const groups = ref<Record<string, ClientGroupInfo>>({});
  const selectedServiceId = ref<string | null>(null);
  const selectedGroupId = ref<string | null>(null);
  const scrollPositions = ref<Record<string, ScrollPosition | undefined>>({});
  const selectedService = computed(() =>
    selectedServiceId.value ? services.value[selectedServiceId.value] : null
  );
  const selectedGroup = computed(() =>
    selectedGroupId.value ? groups.value[selectedGroupId.value] : null
  );
  const readLogs = ref<Record<string, Set<string>>>({});

  function mapToClientServiceInfo(service: ServiceInfo): ClientServiceInfo {
    return {
      ...service,
      logs: service.logs.map((log: any) => transformLogEntry(log)),
      unreadErrors: 0,
    };
  }

  async function loadAll() {
    const newServices = await GetServices();
    const mappedServices: Record<string, ClientServiceInfo> = {};
    for (const [id, service] of Object.entries(newServices)) {
      mappedServices[id] = mapToClientServiceInfo(service);
    }
    services.value = mappedServices;

    const newGroups = await GetGroups();
    const mappedGroups: Record<string, ClientGroupInfo> = {};
    for (const [id, group] of Object.entries(newGroups)) {
      const groupServices: Record<string, ClientServiceInfo> = {};
      for (const serviceId of Object.keys(group.services)) {
        if (mappedServices[serviceId]) {
          groupServices[serviceId] = mappedServices[serviceId];
        }
      }
      mappedGroups[id] = {
        name: group.name,
        env: group.env,
        services: groupServices,
      };
    }
    groups.value = mappedGroups;
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

  async function addGroup(name: string, env: Record<string, string>) {
    await AddGroup(name, env);
    await loadAll();
  }

  async function updateGroup(id: string, name: string, env: Record<string, string>) {
    await UpdateGroup(id, name, env);
    await loadAll();
  }

  async function addServiceToGroup(groupId: string, config: ServiceConfig) {
    await AddServiceToGroup(groupId, config);
    await loadAll();
  }

  async function updateServiceInGroup(groupId: string, serviceId: string, config: ServiceConfig) {
    await UpdateServiceInGroup(groupId, serviceId, config);
    await loadAll();
  }

  async function importSLN(slnPath: string) {
    await ImportSLN(slnPath);
    await loadAll();
  }

  async function importProject(groupId: string, path: string, projectType: string) {
    await ImportProject(groupId, path, projectType);
    await loadAll();
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
      inheritedEnv: service.InheritedEnv,
      type: service.Config.type,
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
      type: config.type,
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
            const clientLog = transformLogEntry(msg.data.log);
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
    await loadAll();
  });

  async function clearLogs(id: string) {
    await ClearLogs(id);
    services.value[id].logs = [];
  }

  async function reloadConfig() {
    await ReloadServices();
    await loadAll();
  }

  async function deleteService(serviceId: string) {
    await DeleteService(serviceId);
    await loadAll();
  }

  async function startGroup(groupId: string) {
    await StartGroup(groupId);
  }

  async function browse(title: string, filterName: string, pattern: string): Promise<string> {
    return await Browse(title, filterName, pattern);
  }

  function saveScrollPosition(serviceId: string, position: ScrollPosition | undefined) {
    scrollPositions.value[serviceId] = position;
  }

  function getScrollPosition(serviceId: string): ScrollPosition | undefined {
    return scrollPositions.value[serviceId];
  }

  return {
    services,
    groups,
    selectedService,
    selectedServiceId,
    selectedGroup,
    selectedGroupId,
    startService,
    stopService,
    restartService,
    selectService,
    isLogRead,
    markLogAsRead,
    getUnreadErrorCount,
    addService,
    updateService,
    addGroup,
    updateGroup,
    addServiceToGroup,
    updateServiceInGroup,
    importSLN,
    clearLogs,
    reloadConfig,
    deleteService,
    startGroup,
    browse,
    loadAll,
    importProject,
    saveScrollPosition,
    getScrollPosition,
  };
});

