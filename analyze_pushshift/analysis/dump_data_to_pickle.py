import psycopg2
from collections import defaultdict
import pickle
import json
from pathlib import Path
import psycopg2.extras

# Dump user and channel mapping
conn = psycopg2.connect(
    dbname="postgres",
    user="postgres",
    password="",
    host="localhost",
    port="30701"
)

cur = conn.cursor()

user_to_channels = defaultdict(set)
channel_to_users = defaultdict(set)

cur.execute("""
    SELECT DISTINCT from_id, to_channel_id
    FROM "Message"
    WHERE from_id IS NOT NULL AND to_channel_id IS NOT NULL;
""")

for from_id, channel_id in cur.fetchall():
    user_to_channels[from_id].add(channel_id)
    channel_to_users[channel_id].add(from_id)

Path('user_to_channels.pickle').write_bytes(pickle.dumps(user_to_channels))
Path('channel_to_users.pickle').write_bytes(pickle.dumps(channel_to_users))

# Path('user_to_channels.json').write_text(json.dumps({k: list(v) for k, v in user_to_channels.items()}, indent=4))
# Path('channel_to_users.json').write_text(json.dumps({k: list(v) for k, v in channel_to_users.items()}, indent=4))

cur.close()

# Dump chatbot id list
cur = conn.cursor(cursor_factory=psycopg2.extras.DictCursor)
cur.execute("""
    SELECT *
    FROM "User"
    WHERE bot IS TRUE;
""")

bot_users = [dict(row) for row in cur.fetchall()]

Path('bot_users.pickle').write_bytes(pickle.dumps(bot_users))
# Path('bot_users.json').write_text(json.dumps(bot_users, indent=4, default=str))

cur.close()
conn.close()

# Dump bot-user encounters
bot_user_ids = [bot_user['id'] for bot_user in bot_users if bot_user['bot_chat_history']]
print(f'{len(bot_user_ids)=}')

bot_encounters = defaultdict(lambda: defaultdict(set))

for bot_id in bot_user_ids:
    if bot_id not in user_to_channels:
        continue
    channels = user_to_channels[bot_id]
    for channel_id in channels:
        for user_id in channel_to_users[channel_id]:
            if user_id != bot_id:
                bot_encounters[bot_id][user_id].add(channel_id)

bot_encounters = {k: v for k, v in bot_encounters.items() if v}

Path('bot_encounters.pickle').write_bytes(pickle.dumps({k: {user: list(channels) for user, channels in v.items()} for k, v in bot_encounters.items()}))
# Path('bot_encounters.json').write_text(
#     json.dumps({k: {user: list(channels) for user, channels in v.items()} for k, v in bot_encounters.items()}, indent=4, default=str)
# )