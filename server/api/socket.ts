import { WebSocketServer } from "ws";
import type { LogType, LogOptions } from "../types";

const socket = new WebSocketServer({ noServer: true });

export function broadcast<T extends LogType>(type: T, data: LogOptions[T]) {
  for (const client of socket.clients) {
    if (client.readyState === client.OPEN) {
      client.send(JSON.stringify({ type, data }));
    }
  }
}

// upgrade request to web socket
export default defineEventHandler((event) => {
  if (event.node.req.headers.upgrade?.toLowerCase() !== "websocket") {
    throw createError({
      statusCode: 400,
      message: "Failed to upgrade connection",
    });
  }

  return new Promise<void>((resolve) => {
    socket.handleUpgrade(
      event.node.req,
      event.node.req.socket,
      Buffer.alloc(0),
      (ws) => {
        console.log("socket connected");
        socket.emit("connection", ws, event.node.req);

        ws.on("close", () => {
          console.log("socket disconnected");
          resolve();
        });
      }
    );
  });
});
