import { services } from "~/server/Service";

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, "id");
  if (!id) {
    throw createError({
      statusCode: 400,
      message: "Missing service id",
    });
  }

  const service = services[id];
  if (!service) {
    throw createError({
      statusCode: 404,
      message: "Service not found",
    });
  }

  service.clearLogs();

  return {
    success: true,
  };
});
