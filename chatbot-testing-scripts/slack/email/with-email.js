const { WebClient } = require("@slack/web-api");

const botToken = "{TOKEN_REDACTED}";
const userId = "{SAMPLE_USER_ID_REDACTED}";

const webClient = new WebClient(botToken);

async function fetch() {
  console.log(
    (
      await webClient.users.info({
        user: userId,
      })
    ).user.profile.email
  );
}

fetch();
