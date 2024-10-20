package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Ошибка загрузки файла с перменными среды: %v", err)
	}

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("Переменная среды BOT_TOKEN не установлена")
	}

	adminIDStr := os.Getenv("ADMIN_CHAT_ID")
	if adminIDStr == "" {
		log.Fatal("Переменная среды ADMIN_CHAT_ID не установлена")
	}
	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		log.Fatalf("ADMIN_CHAT_ID должен быть числом: %v", err)
	}

	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	createTableQuery := `
    CREATE TABLE IF NOT EXISTS messages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER,
        username TEXT,
        message TEXT,
        answered INTEGER DEFAULT 0
    );`
	if _, err := db.Exec(createTableQuery); err != nil {
		log.Fatalf("Ошибка создания базы данных: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("Ошибка инициализации бота: %v", err)
	}

	bot.Debug = false
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := bot.GetUpdatesChan(updateConfig)

	userStates := make(map[int64]string)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		isAdmin := chatID == adminID
		text := update.Message.Text

		if isAdmin {
			switch {
			case text == "/see_queries":
				rows, err := db.Query("SELECT id, username, message FROM messages WHERE answered = 0")
				if err != nil {
					log.Printf("Ошибка чтения сообщений: %v", err)
					continue
				}
				defer rows.Close()

				var response strings.Builder
				for rows.Next() {
					var id int
					var username, message string
					if err := rows.Scan(&id, &username, &message); err != nil {
						log.Printf("Ошибка сканирования сообщения: %v", err)
						continue
					}
					response.WriteString(fmt.Sprintf("ID: %d\nПользователь: %s\nСообщение: %s\n\n", id, username, message))
				}

				if response.Len() == 0 {
					response.WriteString("Нет новых вопросов.")
				}

				msg := tgbotapi.NewMessage(chatID, response.String())
				bot.Send(msg)
			case text == "/answer":
				userStates[chatID] = "awaiting_question_id"
				msg := tgbotapi.NewMessage(chatID, "Пожалуйста, укажите ID вопроса, на который хотите ответить.")
				bot.Send(msg)
			}
		} else {
			switch text {
			case "/contact":
				userStates[chatID] = "awaiting_message"
				msg := tgbotapi.NewMessage(chatID, "Напишите мне сообщение, которое нужно отправить администратору.")
				bot.Send(msg)
			case "/start":
				msg := tgbotapi.NewMessage(chatID, "Здравствуйте! Используйте /contact, чтобы связаться с администратором.")
				bot.Send(msg)
			default:
				if userStates[chatID] == "awaiting_message" {
					_, err := db.Exec(
						"INSERT INTO messages (user_id, username, message) VALUES (?, ?, ?)",
						chatID, update.Message.From.UserName, text,
					)
					if err != nil {
						log.Printf("Ошибка сохранения сообщения в базу данных: %v", err)
						msg := tgbotapi.NewMessage(chatID, "Ошибка при сохранении вашего сообщения.")
						bot.Send(msg)
						continue
					}

					adminMsg := tgbotapi.NewMessage(adminID, fmt.Sprintf("Новый вопрос от @%s:\n%s", update.Message.From.UserName, text))
					bot.Send(adminMsg)

					msg := tgbotapi.NewMessage(chatID, "Ваше сообщение отправлено администратору.")
					bot.Send(msg)
					userStates[chatID] = ""
				} else {
					msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте /contact, чтобы связаться с администратором.")
					bot.Send(msg)
				}
			}
		}
	}
}
