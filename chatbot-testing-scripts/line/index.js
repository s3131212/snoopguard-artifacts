const line = require("@line/bot-sdk");
const express = require('express')

const config = {
  channelSecret: "{SECRET_REDACTED}",
};

const client = new line.messagingApi.MessagingApiClient({
  channelAccessToken: "{TOKEN_REDACTED}"
});

const app = express();

app.post('/', line.middleware(config), (req, res) => {
  Promise
    .all(req.body.events.map(handleEvent))
    .then((result) => res.json(result))
    .catch((err) => {
      console.error(err);
      res.status(500).end();
    });
});

function handleEvent(event) {
  if (event.replyToken) return client.replyMessage({
    replyToken: event.replyToken,
    messages: [ { type: 'text', text: 'Recieved your message.' } ],
  })
  else return null;
}

const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(`listening on ${port}`);
});