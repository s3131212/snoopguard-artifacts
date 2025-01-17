const Bot = require("keybase-bot");

async function main() {
  const bot = new Bot();
  try {
    const username = process.env.KB_USERNAME;
    const paperkey = process.env.KB_PAPERKEY;
    await bot.init(username, paperkey, { verbose: true, adminDebugDirectory: './logs' });
    console.log(
      `Your bot is initialized. It is logged in as ${bot.myInfo().username}`
    );
    await bot.chat.advertiseCommands({
      advertisements: [
        {
          type: "public",
          commands: [
            {
              name: "echo",
              description: "Just repeat your word.",
              usage: "[your text]",
            },
          ],
        },
      ],
    });
    await bot.chat.watchAllChannelsForNewMessages((message) => {
      console.log(message);
      console.log(new Error().stack);
      bot.chat.send(message.conversationId, { body: "Received your message." });
    }, (error) => {
      console.log(error)
    });
  } catch (error) {
    console.error(error);
  } finally {
    await bot.deinit();
  }
}

main();
