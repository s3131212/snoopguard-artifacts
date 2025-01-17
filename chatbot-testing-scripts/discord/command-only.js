const { Client, Events, GatewayIntentBits } = require("discord.js");

const client = new Client({
  intents: [GatewayIntentBits.Guilds],
});

client.on(Events.InteractionCreate, (interaction) => {
  if (!interaction.isChatInputCommand()) return;
  interaction.reply({ content: `Received <@${interaction.user.username}> message: "${interaction.options.getString("input")}"`});
});

client.login(
  "{TOKEN_REDACTED}"
);
