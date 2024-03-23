use("chatroom")
db.createCollection("chatroom_list")

db.chatroom_name.createIndex({"chatroom_id": 1}, {unique: true})
db.chatroom_name.createIndex({"name": 1}, {unique: true})

db.chatroom_list.insertOne({
    "name": "mainChat",
    "chatroom_id": UUID("00000000-0000-0000-0000-000000000001"),
    "is_active": true
})

db.createCollection("chatroom_users")

db.chatroom_name.createIndex({"chatroom_id": 1, "user_id": 1}, {unique: true})
