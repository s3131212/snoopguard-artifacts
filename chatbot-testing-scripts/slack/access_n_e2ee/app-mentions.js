const { SocketModeClient } = require('@slack/socket-mode');
const { WebClient } = require('@slack/web-api');

const appToken = "{TOKEN_REDACTED}";
const botToken = "{TOKEN_REDACTED}"

const socketModeClient = new SocketModeClient({appToken});
const webClient = new WebClient(botToken);

socketModeClient.on('app_mention', ({event, body, ack}) => {
  console.log(body)
  ack();
  webClient.chat.postMessage({
    text: "Received your message",
    response_type: "in_channel",
    channel: event.channel
  })
});

(async () => {
  await socketModeClient.start();
})();