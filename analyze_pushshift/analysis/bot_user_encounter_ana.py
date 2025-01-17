import pickle
from pathlib import Path
from collections import defaultdict
from tqdm import tqdm
from collections import Counter

bot_users = pickle.loads(Path('bot_users.pickle').read_bytes())
bot_encounters = pickle.loads(Path('bot_encounters.pickle').read_bytes())
user_to_channels = pickle.loads(Path('user_to_channels.pickle').read_bytes())
channel_to_users = pickle.loads(Path('channel_to_users.pickle').read_bytes())

bot_can_read_history = [bot_user['id'] for bot_user in bot_users if bot_user['bot_chat_history']]

# (user, chatbot) pair
encounter_distribution = Counter()
for bot_id, users in bot_encounters.items():
    if bot_id in bot_can_read_history:
        for user_id, channels in users.items():
            encounter_count = len(channels)
            encounter_distribution[encounter_count] += 1

# print("=== Users encounter the same chatbot in N channels ===")
# for encounter_count, frequency in sorted(encounter_distribution.items()):
#     print(f"Encountered in {encounter_count} channels: {frequency} users")

total_users = sum(encounter_distribution.values())
cumulative_count = 0
cumulative_distribution = []
larger_than_ten = 0
print("Encountered in X channels: Frequency | Cumulative Count | Cumulative Proportion | Proportion of Total")
print("---------------------------------------------------------------------------------------------")

for encounter_count, frequency in sorted(encounter_distribution.items()):
    cumulative_count += frequency
    cumulative_proportion = cumulative_count / total_users
    proportion_of_total = frequency / total_users
    cumulative_distribution.append((encounter_count, frequency, cumulative_count, cumulative_proportion, proportion_of_total))

    print(f"{encounter_count:>19} : {frequency:>8} | {cumulative_count:>15} | {cumulative_proportion:>19.2%} | {proportion_of_total:>19.2%}")

    if encounter_count > 10:
        larger_than_ten += frequency

print(f"Larger than ten: {larger_than_ten}")

# Calculate the amount of users that encounter the same bot in different groups.
encounter_bots_multiple_groups = defaultdict(set) # {user: {bot1, bot2, ...}}, where bot1, bot2, ... are the bots that user encountered in multiple groups
for bot, users in bot_encounters.items():
    if bot not in bot_can_read_history:
        continue
    for user, groups in users.items():
        if len(groups) > 1:
            encounter_bots_multiple_groups[user].add(bot)

# Calculate the distribution of the number of bots that a user encounters in multiple groups
print("=== The distribution of the number of bots that a user encounters in multiple groups ===")
encounter_distribution = Counter(len(bots) for bots in encounter_bots_multiple_groups.values())
print(f"User Encounter Distribution: {encounter_distribution}")
print(f"Total users: {len(user_to_channels)}")
# Generate the distribution table
print("Has Multiple Encounter: Frequency | Cumulative Count | Cumulative Proportion | Proportion of Total")
print("---------------------------------------------------------------------------------------------")
total_users = len(user_to_channels)
cumulative_count = 0
cumulative_distribution = []
for encounter_count, frequency in sorted(encounter_distribution.items()):
    cumulative_count += frequency
    cumulative_proportion = cumulative_count / total_users
    proportion_of_total = frequency / total_users
    cumulative_distribution.append((encounter_count, frequency, cumulative_count, cumulative_proportion, proportion_of_total))

    print(f"{encounter_count:>19} : {frequency:>8} | {cumulative_count:>15} | {cumulative_proportion:>19.2%} | {proportion_of_total:>19.2%}")

print("Users encountering the same chatbots in multiple groups: ", sum(encounter_distribution.values()), sum(encounter_distribution.values()) / total_users)

# Percentage of Chatbots with Multiple Group Interactions
chatbots_with_multiple_group_interactions = 0
total_chatbots = len(bot_encounters)

for bot_id, user_dict in bot_encounters.items():
    if bot_id not in bot_can_read_history:
        continue

    found_multiple_interactions = False
    for user_id, user_channels in user_dict.items():
        if len(user_channels) > 1:
            found_multiple_interactions = True
            break
    if found_multiple_interactions:
        chatbots_with_multiple_group_interactions += 1

# Calculate the percentage on average
percentage_with_multiple_interactions = (chatbots_with_multiple_group_interactions / total_chatbots) * 100 if total_chatbots > 0 else 0
print(f"Percentage of Chatbots with Multiple Group Interactions: {percentage_with_multiple_interactions:.2f}%")


# Calculate the percentage of users encountering the same chatbot in more than one group
total_users = 0
users_with_multiple_encounters = 0
user_encounters_distribution = []

for bot, users in bot_encounters.items():
    if bot not in bot_can_read_history:
        continue
    for user, groups in users.items():
        total_users += 1
        if len(groups) > 1:
            users_with_multiple_encounters += 1
            user_encounters_distribution.append(len(groups))

repeat_interaction_rate = (users_with_multiple_encounters / total_users) * 100
print(f"User-chatbot encountered multiple times: {users_with_multiple_encounters}, {repeat_interaction_rate:.2f}%")

# Calculate the distribution of user encounters across different numbers of groups
distribution = Counter(user_encounters_distribution)
print(f"User Encounters Distribution: {distribution}")