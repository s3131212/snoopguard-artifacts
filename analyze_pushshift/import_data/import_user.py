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
NDJSON_FILE_PATH = './accounts.ndjson'

# Batch size for inserts
BATCH_SIZE = 10000

count = 0

# Insert data into PostgreSQL using execute_values
def insert_batch(cursor,conn, data):
    global count
    query = """
        INSERT INTO "User" (
            access_hash, bot, bot_chat_history, bot_info_version, bot_inline_geo,
            bot_inline_placeholder, bot_nochats, contact, deleted, first_name,
            id, is_self, lang_code, last_name, min, mutual_contact, phone,
            restricted, retrieved_utc, scam, support, username, verified
        ) VALUES %s ON CONFLICT (username) DO NOTHING
    """

    try:
        execute_values(cursor, query, data, """
            (%(access_hash)s, %(bot)s, %(bot_chat_history)s, %(bot_info_version)s,
            %(bot_inline_geo)s, %(bot_inline_placeholder)s, %(bot_nochats)s,
            %(contact)s, %(deleted)s, %(first_name)s, %(id)s, %(is_self)s,
            %(lang_code)s, %(last_name)s, %(min)s, %(mutual_contact)s, %(phone)s,
            %(restricted)s, TO_TIMESTAMP(%(retrieved_utc)s), %(scam)s, %(support)s, %(username)s, %(verified)s)
        """)
        count += len(data)
        conn.commit()
        print(f"Inserted {count} records.")
    except Exception as e:
        print(f"Error inserting batch: {e}")

# Process NDJSON file in batches
def process_ndjson():
    global count
    try:
        # Establish PostgreSQL connection
        with psycopg2.connect(**DB_CONFIG) as conn:
            with conn.cursor() as cursor:
                batch = []
                with open(NDJSON_FILE_PATH, 'r') as file:
                    reader = ndjson.reader(file)

                    # Read and process each record
                    for record in reader:
                        batch.append(record)

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
