const clientId = "{ID_REDACTED}";
const guildId = "{ID_REDACTED}";
const token =
  "{TOKEN_REDACTED}";

const { REST, Routes, SlashCommandBuilder } = require("discord.js");

const rest = new REST().setToken(token);

rest.put(Routes.applicationGuildCommands(clientId, guildId), {
  body: [
    new SlashCommandBuilder()
      .setName("echo")
      .setDescription("Just repeat your word.")
      .addStringOption((option) =>
        option.setName("input").setDescription("The input to echo back")
      )
      .toJSON(),
  ],
});
