package db

import (
	"chatroom_text/models"
	"chatroom_text/repo"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"
)

type chatroomDB struct {
}

const tsFormat = "2006-01-02T15:04:05"

var db *sqlx.DB

func InitDB() error {
	var err error
	// open and connect at the same time:
	// db, err = sqlx.Open("postgres", "user=pguser dbname=chatroom password=pgpass host=127.0.0.1 sslmode=disable")
	db, err = sqlx.Open("postgres", "user=pguser dbname=chatroom password=pgpass host=postgres sslmode=disable")
	if err != nil {
		return errors.Wrap(err, "failed to open db connection")
	}

	err = db.Ping()
	if err != nil {
		return errors.Wrap(err, "failed to initially ping db")
	}

	return nil
}

func GetChatroomLogger() repo.ChatroomLogger {
	return chatroomDB{}
}

func (c chatroomDB) GetChatroomLogs(params models.GetDBMessagesParams) ([]models.ChatroomLog, error) {
	// @todo add message to chatroom message history
	logs := []models.ChatroomLog{}

	rows, err := db.Queryx("SELECT chatroom_id, log_timestamp, log_text, client_id FROM chatroom_logs WHERE 1 = 1")
	if err != nil {
		return nil, errors.Wrap(err, "failed to query chatroom logs")
	}

	for rows.Next() {
		var log models.ChatroomLog
		if err := rows.StructScan(&log); err != nil {
			return nil, errors.Wrap(err, "failed to scan chatroom logs row")
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "received chatroom logs rows error")
	}

	return logs, nil
}

// @todo standardize timestamp
func (c chatroomDB) SetChatroomLogs(params models.SetDBMessagesParams) error {
	slog.Info(fmt.Sprintf("setting log: %v", params))
	// @todo add inserts from params
	query := fmt.Sprintf(`
	INSERT INTO chatroom_logs (chatroom_id, log_timestamp, log_text, client_id) 
	VALUES ('%s', '%s', '%s', '%s')`,
		params.ChatroomID.String(),
		params.Timestamp.Format(tsFormat),
		params.Text,
		params.ClientID,
	)
	if _, err := db.Exec(query); err != nil {
		return errors.Wrapf(err, "failed to insert log to chatroom %s", params.ChatroomID)
	}

	return nil
}
