import { addService } from "~/server/Service";
import { mapToServiceInfo } from "../../utils/serviceMapper";
import type { ServiceConfig } from "~/server/Service";

export default defineEventHandler(async (event) => {
  const config = await readBody<ServiceConfig>(event);
  const service = addService(config);
  return { id: service.id, service: mapToServiceInfo(service) };
});
