import { services } from "~/server/Service";
import { mapToServiceInfo } from "../../utils/serviceMapper";
import { mapValues } from "lodash-es";

export default defineEventHandler(async () => {
  return mapValues(services, mapToServiceInfo);
});
