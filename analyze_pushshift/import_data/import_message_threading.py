import ndjson
import psycopg2
from psycopg2 import pool
from psycopg2.extras import execute_values
import json
from multiprocessing import Queue, Pool
from tqdm import tqdm

NDJSON_FILE_PATH = './messages.ndjson'
BATCH_SIZE = 10000
THREAD_COUNT = 30
DB_CONFIG = {
    "host": "localhost",
    "port": 30701,
    "dbname": "postgres",
    "user": "postgres",
    "password": ""
}

batch_queue = Queue(maxsize=5 * THREAD_COUNT)
error_queue = Queue()

def read_ndjson_in_batches():
    with open(NDJSON_FILE_PATH, 'r') as file:
        batch = []
        reader = ndjson.reader(file)
        for record in tqdm(reader, desc="Processing NDJSON", unit="lines"):
            if (record["_"] == "MessageService"): continue
            record["to_channel_id"] = record["to_id"]["channel_id"]
            batch.append(record)
            if len(batch) >= BATCH_SIZE:
                yield batch
                batch = []
        # Enqueue remaining records
        if batch:
            yield batch

def insert_batch(cur, conn, batch):
    query = """
        INSERT INTO "Message" (
            date, edit_date, from_id, from_scheduled, grouped_id,
            id, legacy, media_unread, mentioned, message, out,
            post, post_author, reply_to_msg_id, retrieved_utc,
            silent, to_channel_id, via_bot_id, views
        ) VALUES %s ON CONFLICT (id, to_channel_id) DO NOTHING
    """
    try:
        execute_values(cur, query, batch, """
            (%(date)s, %(edit_date)s, %(from_id)s, %(from_scheduled)s, %(grouped_id)s,
            %(id)s, %(legacy)s, %(media_unread)s, %(mentioned)s, %(message)s, %(out)s,
            %(post)s, %(post_author)s, %(reply_to_msg_id)s, TO_TIMESTAMP(%(retrieved_utc)s),
            %(silent)s, %(to_channel_id)s, %(via_bot_id)s, %(views)s)
        """)
        conn.commit()
    except Exception as e:
        conn.rollback()
        if (len(batch) == 1): return f"Error processing batch: {e}"
        else:
            error_message = ''
            for item in batch:
                error = insert_batch(cur, conn, [item])
                if error:
                    error_message = error_message + error + '\n'
            return error_message

def worker(queue):
    with psycopg2.connect(**DB_CONFIG) as conn:
        with conn.cursor() as cursor:
            while True:
                batch = queue.get()
                if batch is None:  # Sentinel to exit the thread
                    break
                error = insert_batch(cursor, conn, batch)
                if error:
                    error_queue.put(error)

def write_errors_to_file(error_queue, file_path):
    """
    Main process that writes errors from the error queue to a file.
    """
    with open(file_path, 'a') as f:  # Open in append mode
        while True:
            error_message = error_queue.get()
            print(error_message)
            if error_message == "DONE":
                break  # Sentinel to stop writing
            f.write(error_message + '\n')

def main():
    with Pool(THREAD_COUNT, worker, (batch_queue,)) as pool:
        with Pool(1, write_errors_to_file, (error_queue, 'errors.log')) as error_pool:
            for batch in read_ndjson_in_batches():
                batch_queue.put(batch)
            for _ in range(THREAD_COUNT):
                batch_queue.put(None)
            error_queue.put("DONE")
            pool.close()  # Prevent new tasks from being added to the pool
            pool.join()
            error_pool.close()
            error_pool.join()
    print("All tasks completed.")


if __name__ == "__main__":
    main()
