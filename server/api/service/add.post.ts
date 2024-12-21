import { addService } from "~/server/Service";
import { mapToServiceInfo } from "../../utils/serviceMapper";

export default defineEventHandler(async (event) => {
  const { path } = await readBody(event);
  const service = addService(path);
  return { service: mapToServiceInfo(service) };
});
