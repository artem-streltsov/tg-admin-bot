package db

import (
	"database/sql"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

type Message struct {
	ID       int
	UserID   int64
	Username string
	Message  string
	Answer   string
}

func New(databaseDSN string) *DB {
	conn, err := sql.Open("sqlite3", databaseDSN)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	return &DB{conn: conn}
}

func (db *DB) Close() {
	db.conn.Close()
}

func (db *DB) Migrate() {
	driver, err := sqlite3.WithInstance(db.conn, &sqlite3.Config{})
	if err != nil {
		log.Fatalf("Ошибка создания миграционного драйвера: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://migrations", "sqlite3", driver)
	if err != nil {
		log.Fatalf("Ошибка инициализации миграции: %v", err)
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Ошибка миграции: %v", err)
	}
}

func (db *DB) GetPendingMessages() ([]Message, error) {
	rows, err := db.conn.Query("SELECT id, username, message FROM messages WHERE answered = 0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.Username, &msg.Message); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (db *DB) GetAnsweredMessages() ([]Message, error) {
	rows, err := db.conn.Query("SELECT id, username, message, answer FROM messages WHERE answered = 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.Username, &msg.Message, &msg.Answer); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (db *DB) GetUserMessages(username string) ([]Message, error) {
	rows, err := db.conn.Query("SELECT message, answer FROM messages WHERE username=?", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.Message, &msg.Answer); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (db *DB) SaveMessage(userID int64, username, message string) (int64, error) {
	result, err := db.conn.Exec("INSERT INTO messages (user_id, username, message) VALUES (?, ?, ?)", userID, username, message)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) GetMessageByID(questionID int) (Message, error) {
	var msg Message
	err := db.conn.QueryRow("SELECT user_id, username, message FROM messages WHERE id = ?", questionID).Scan(&msg.UserID, &msg.Username, &msg.Message)
	return msg, err
}

func (db *DB) UpdateMessageAnswer(questionID int, answer string) error {
	_, err := db.conn.Exec("UPDATE messages SET answer=?, answered=1 WHERE id=?", answer, questionID)
	return err
}
