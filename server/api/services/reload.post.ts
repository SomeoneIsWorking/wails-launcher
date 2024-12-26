import { mapValues } from "lodash-es";
import { reloadServices } from "~/server/Service";

export default defineEventHandler(async () => {
  const newData = await reloadServices();
  return mapValues(newData, mapToServiceInfo);
});
