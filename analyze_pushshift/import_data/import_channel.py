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
        INSERT INTO "Channel" (
            access_hash, broadcast, creator, date, has_geo, has_link, id,
            megagroup, min, participants_count, restricted, scam, signatures, title,
            username, verified, version
        ) VALUES %s ON CONFLICT (id) DO NOTHING
    """

    try:
        execute_values(cursor, query, data, """
            (%(access_hash)s, %(broadcast)s, %(creator)s, %(date)s,
            %(has_geo)s, %(has_link)s, %(id)s,
            %(megagroup)s, %(min)s, %(participants_count)s, %(restricted)s, %(scam)s,
            %(signatures)s, %(title)s, %(username)s, %(verified)s, %(version)s)
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
                        for chat in record["chats"]:
                            batch.append(chat)
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
