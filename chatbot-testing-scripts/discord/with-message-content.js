const { Client, Events, GatewayIntentBits } = require("discord.js");

const client = new Client({
  intents: [
    GatewayIntentBits.Guilds,
    GatewayIntentBits.GuildMessages,
    GatewayIntentBits.MessageContent,
  ]
});

client.on(Events.MessageCreate, (message) => {
  if (!message.author.bot)
    message.channel.send(
      `Received <@${message.author.username}> message: "${message.content}"`
    );
});

client.login(
  "{TOKEN_REDACTED}"
);
