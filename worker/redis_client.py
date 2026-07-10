import redis

client = redis.Redis(
    host="localhost",
    port=6379,
    decode_responses=True,
)

print("Connected to Redis:", client.ping())

try:
    client.xgroup_create(
        name="relay-stream",
        groupname="relay-workers",
        id="0",
        mkstream=True,
    )
    print("Consumer group created")
    
except redis.exceptions.ResponseError:
    print("Consumer group already exists")