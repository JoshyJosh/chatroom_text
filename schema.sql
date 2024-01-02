CREATE TABLE chatroom_logs (
	chatroom_id   VARCHAR(128),
	log_timestamp TIMESTAMP,
	log_text      TEXT,
	client_id     VARCHAR(128),
	PRIMARY KEY (chatroom_id, log_timestamp)
);


CREATE TABLE clients (
	client_id   VARCHAR(128),
	client_name VARCHAR(64),
	PRIMARY KEY (client_id)
);

CREATE DATABASE supertokens;