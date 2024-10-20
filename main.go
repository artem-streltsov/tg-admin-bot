package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

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

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		isAdmin := chatID == adminID
		var msg tgbotapi.MessageConfig

		if isAdmin {
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вы администратор")
		} else {
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вы пользователь")
		}

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}
	}
}
