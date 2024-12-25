import { editService } from "~/server/Service";
import { mapToServiceInfo } from "../../utils/serviceMapper";
import type { ServiceConfig } from "~/server/Service";

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id');
  if (!id) throw createError({ statusCode: 400, message: 'Missing ID' });
  
  const config = await readBody<Partial<ServiceConfig>>(event);
  const service = editService(id, config);
  return { service: mapToServiceInfo(service) };
});
