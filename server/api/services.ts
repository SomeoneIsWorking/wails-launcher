import { services } from "../Service";
import { ServiceInfo } from "../types";
import { mapToServiceInfo } from "../utils/serviceMapper";

export default defineEventHandler((): ServiceInfo[] => {
  return Array.from(services.values()).map(mapToServiceInfo);
});
