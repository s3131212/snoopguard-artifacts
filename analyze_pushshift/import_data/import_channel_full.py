import ndjson
import psycopg2
from psycopg2.extras import execute_values
from psycopg2 import sql

# PostgreSQL connection configuration
DB_CONFIG = {
    "host": "localhost",
    "port": 30701,
    "dbname": "postgres",
    "user": "postgres",
    "password": ""
}

# NDJSON file path
NDJSON_FILE_PATH = './channels.ndjson'

# Batch size for inserts
BATCH_SIZE = 1000

count = 0

# Insert data into PostgreSQL using execute_values
def insert_batch(cursor,conn, data):
    global count
    query = """
        INSERT INTO "ChannelFull" (
            about, admins_count, available_min_id, banned_count, can_set_location,
            can_set_stickers, can_set_username, can_view_participants, can_view_stats,
            folder_id, hidden_prehistory, id, kicked_count, linked_chat_id,
            migrated_from_chat_id, migrated_from_max_id, online_count,
            participants_count, pinned_msg_id, pts, read_inbox_max_id,
            read_outbox_max_id, unread_count
        ) VALUES %s ON CONFLICT (id) DO NOTHING
    """

    try:
        execute_values(cursor, query, data, """
            (%(about)s, %(admins_count)s, %(available_min_id)s, %(banned_count)s, %(can_set_location)s,
            %(can_set_stickers)s, %(can_set_username)s, %(can_view_participants)s, %(can_view_stats)s,
            %(folder_id)s, %(hidden_prehistory)s, %(id)s, %(kicked_count)s, %(linked_chat_id)s,
            %(migrated_from_chat_id)s, %(migrated_from_max_id)s, %(online_count)s,
            %(participants_count)s, %(pinned_msg_id)s, %(pts)s, %(read_inbox_max_id)s,
            %(read_outbox_max_id)s, %(unread_count)s)
        """)
        count += len(data)
        conn.commit()
        print(f"Inserted {count} records.")
    except Exception as e:
        print(f"Error inserting batch: {e}")

# Process NDJSON file in batches
def process_ndjson():
    try:
        # Establish PostgreSQL connection
        with psycopg2.connect(**DB_CONFIG) as conn:
            with conn.cursor() as cursor:
                batch = []
                with open(NDJSON_FILE_PATH, 'r') as file:
                    reader = ndjson.reader(file)

                    # Read and process each record
                    for record in reader:
                        batch.append(record["full_chat"])

                        # Insert batch if size is reached
                        if len(batch) >= BATCH_SIZE:
                            insert_batch(cursor,conn, batch)
                            batch.clear()  # Clear batch after insertion

                    # Insert any remaining records
                    if batch:
                        insert_batch(cursor,conn, batch)

            print("File processing completed.")
    except Exception as e:
        print(f"Error processing NDJSON: {e}")

# Run the script
if __name__ == "__main__":
    process_ndjson()
