CREATE TABLE chatroom_logs (
	chatroom_id INTEGER,
	timestamp   TIMESTAMP,
	text        TEXT,
	clientID    UUID,
	PRIMARY KEY (chatroom_id, timestamp)
);


CREATE TABLE clients (
	clientID UUID,
	name     VARCHAR(64),
	PRIMARY KEY (clientID)
);