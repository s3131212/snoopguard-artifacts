const { Client, Events, GatewayIntentBits } = require("discord.js");

const client = new Client({
  intents: [GatewayIntentBits.Guilds, GatewayIntentBits.GuildMessages],
});

client.on(Events.InteractionCreate, (interaction) => {
  if (!interaction.isChatInputCommand()) return;
  interaction.reply({ content: `Received <@${interaction.user.username}> message: "${interaction.options.getString("input")}"`});
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
