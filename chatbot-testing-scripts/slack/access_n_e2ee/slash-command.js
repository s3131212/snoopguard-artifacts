const { SocketModeClient } = require('@slack/socket-mode');
const appToken = "{TOKEN_REDACTED}";

const socketModeClient = new SocketModeClient({appToken});

socketModeClient.on('slash_commands', async ({ body, ack }) => {
    console.log(body)
    if (body.command === "/echo") {
        await ack({
          text: "Received your message",
          response_type: "in_channel"
        });
    }
});

(async () => {
  await socketModeClient.start();
})();