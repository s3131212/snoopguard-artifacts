const TelegramBot = require('node-telegram-bot-api');

const privacyOnBotToken = '{TOKEN_REDACTED}';
const privacyOffBotToken = '{TOKEN_REDACTED}';

setTimeout(() => {
    const privacyOnBot = new TelegramBot(privacyOnBotToken, {polling: true});
    const privacyOffBot = new TelegramBot(privacyOffBotToken, {polling: true});

    privacyOnBot.on('message', (msg) => {
        const chatId = msg.chat.id;
        privacyOnBot.sendMessage(chatId, 'Received your message in privacy mode');
    });

    privacyOffBot.on('message', (msg) => {
        const chatId = msg.chat.id;
        privacyOffBot.sendMessage(chatId, 'Received your message in non-privacy mode');
    });

}, 3000)