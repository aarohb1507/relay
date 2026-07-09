import redis

client = redis.Redis(
    host="localhost",
    port=6379,
    decode_responses=True,
)

print("Connected to Redis:", client.ping())