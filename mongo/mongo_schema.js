use("chatroom")
db.createCollection("chatroom_name")

db.chatroom_name.createIndex({"chatroom_id": 1}, {unique: true})
db.chatroom_name.createIndex({"name": 1}, {unique: true})