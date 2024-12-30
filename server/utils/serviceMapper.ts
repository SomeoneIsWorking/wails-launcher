import type { Service } from "../Service";
import type { ServiceInfo } from "../types";

export function mapToServiceInfo(service: Service): ServiceInfo {
  return {
    ...service.config,
    status: service.status,
    url: service.url,
    logs: service.logs,
  };
}
