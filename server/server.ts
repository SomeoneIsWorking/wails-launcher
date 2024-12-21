import { dotnetService } from "./dotnet";
import { services } from "./Service";

async function startService(servicePath: string) {
  const service = services.get(servicePath)!;

  try {
    await dotnetService.startService(service);
  } catch (error) {
    console.error(`Error in startService for ${name}:`, error);
    service.setStatus("error");
  }
}

async function stopService(servicePath: string) {
  const service = services.get(servicePath)!;
  await dotnetService.stopService(service);
}

export { startService, stopService };
