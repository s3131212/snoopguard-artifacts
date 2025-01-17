import pickle
from pathlib import Path
from collections import defaultdict, Counter
from tqdm import tqdm
import json

bot_users = pickle.loads(Path('bot_users.pickle').read_bytes())

print(f'chatbot len={len(bot_users)}')

bot_cannot_read_history = [bot_user['id'] for bot_user in bot_users if not bot_user['bot_chat_history']]
print(f'bot_cannot_read_history len = {len(bot_cannot_read_history)} ({len(bot_cannot_read_history)/len(bot_users)})')

bot_can_read_history = [bot_user['id'] for bot_user in bot_users if bot_user['bot_chat_history']]
print(f'bot_can_read_history len = {len(bot_can_read_history)} ({len(bot_can_read_history)/len(bot_users)})')

user_to_channels = pickle.loads(Path('user_to_channels.pickle').read_bytes())

chatbot_channel_counts = []
for bot_id in bot_can_read_history:
    chatbot_channel_counts.append(len(user_to_channels.get(bot_id, set())))

chatbot_channel_counts = Counter(chatbot_channel_counts)
more_than_ten = 0
for chatbot_channel_count, frequency in sorted(chatbot_channel_counts.items()):
    print(f"Encountered in {chatbot_channel_count} channels: {frequency} chatbots")
    if chatbot_channel_count > 10:
        more_than_ten += frequency
print(f"more_than_ten: {more_than_ten}")