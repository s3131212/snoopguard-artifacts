const { SocketModeClient } = require("@slack/socket-mode");
const { WebClient } = require("@slack/web-api");

const appToken =
  "{TOKEN_REDACTED}";
const botToken = "{TOKEN_REDACTED}";

const socketModeClient = new SocketModeClient({ appToken });
const webClient = new WebClient(botToken);

socketModeClient.on("message", ({ body, ack }) => {
  console.log(body);
  ack();
  if (!("bot_id" in body.event) && body.event.type === 'message')
    webClient.chat.postMessage({
      text: "Received your message",
      response_type: "in_channel",
      channel: body.event.channel,
    });
});

(async () => {
  await socketModeClient.start();
})();
