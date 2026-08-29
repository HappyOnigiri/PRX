import { createConnection } from "node:net";
import { setTimeout } from "node:timers";

const host = "127.0.0.1";
const port = 7331;
const retryDelayMs = 100;

await new Promise((resolve) => {
  const connect = () => {
    const socket = createConnection({ host, port });
    let retryScheduled = false;

    const retry = () => {
      if (retryScheduled) return;
      retryScheduled = true;
      socket.destroy();
      setTimeout(connect, retryDelayMs);
    };

    socket.once("connect", () => {
      socket.destroy();
      resolve();
    });
    socket.once("error", retry);
  };

  connect();
});
