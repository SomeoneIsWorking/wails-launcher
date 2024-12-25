import type { Service } from "../Service";
import type { ServiceInfo } from "../types";

export function mapToServiceInfo(service: Service): ServiceInfo {
  return {
    name: service.name,
    path: service.path,
    status: service.status,
    url: service.url,
    logs: service.logs,
    env: service.env,
  };
}
