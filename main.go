package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/artem-streltsov/tg-admin-bot/config"
	"github.com/artem-streltsov/tg-admin-bot/database"
)

func main() {
	cfg := config.LoadConfig()

	database := db.New(cfg.DatabasePath)
	defer database.Close()
	database.Migrate()

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("Error initializing bot: %v", err)
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
		isAdmin := chatID == cfg.AdminID
		text := update.Message.Text

		if isAdmin {
			handleAdminMessage(bot, database, userStates, chatID, text)
		} else {
			handleUserMessage(bot, database, userStates, update, cfg.AdminID)
		}
	}
}

func handleAdminMessage(bot *tgbotapi.BotAPI, database *db.DB, userStates map[int64]string, chatID int64, text string) {
	switch {
	case text == "/start":
		msg := "Добро пожаловать, Вы - администратор. Вы можете использовать команды:\n/see_questions - посмотреть вопросы\n/answer - ответить на вопрос\n/see_answers - посмотреть Ваши ответы."
		bot.Send(tgbotapi.NewMessage(chatID, msg))
	case text == "/see_questions":
		messages, err := database.GetPendingMessages()
		if err != nil {
			log.Printf("Error reading messages: %v", err)
			return
		}
		sendMessagesList(bot, chatID, messages, "Нет новых вопросов.")
	case text == "/see_answers":
		messages, err := database.GetAnsweredMessages()
		if err != nil {
			log.Printf("Error reading messages: %v", err)
			return
		}
		sendAnsweredList(bot, chatID, messages, "Нет отвеченных вопросов.")
	case text == "/answer":
		userStates[chatID] = "awaiting_question_id"
		bot.Send(tgbotapi.NewMessage(chatID, "Пожалуйста, укажите ID вопроса, на который хотите ответить."))
	case userStates[chatID] == "awaiting_question_id" || strings.HasPrefix(text, "/answer_"):
		questionID, err := parseQuestionID(text)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Пожалуйста, введите корректный числовой ID."))
			return
		}
		_, err = database.GetMessageByID(questionID)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Вопрос с таким ID не найден или уже был отвечен."))
			userStates[chatID] = ""
			return
		}
		userStates[chatID] = fmt.Sprintf("answering_%d", questionID)
		bot.Send(tgbotapi.NewMessage(chatID, "Пожалуйста, введите сообщение для отправки пользователю."))
	case strings.HasPrefix(userStates[chatID], "answering_"):
		questionID, _ := strconv.Atoi(strings.TrimPrefix(userStates[chatID], "answering_"))
		answer := text
		if err := database.UpdateMessageAnswer(questionID, answer); err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при обновлении ответа в базе данных."))
			userStates[chatID] = ""
			return
		}
		msg, err := database.GetMessageByID(questionID)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении информации о пользователе."))
			userStates[chatID] = ""
			return
		}
		sendAnswerToUser(bot, msg.UserID, msg.Question, answer)
		bot.Send(tgbotapi.NewMessage(chatID, "Сообщение отправлено пользователю."))
		userStates[chatID] = ""
	default:
		bot.Send(tgbotapi.NewMessage(chatID, "Неизвестная команда.\nДоступные команды:\n/see_questions\n/see_answers\n/answer"))
	}
}

func handleUserMessage(bot *tgbotapi.BotAPI, database *db.DB, userStates map[int64]string, update tgbotapi.Update, adminID int64) {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	switch text {
	case "/start":
		bot.Send(tgbotapi.NewMessage(chatID, "Здравствуйте! Вы можете использовать команды:\n/contact - связаться с администратором\n/see_questions - посмотреть на ваши вопросы."))
	case "/contact":
		userStates[chatID] = "awaiting_message"
		bot.Send(tgbotapi.NewMessage(chatID, "Напишите мне сообщение, которое нужно отправить администратору."))
	case "/see_questions":
		messages, err := database.GetUserMessages(update.Message.From.UserName)
		if err != nil {
			log.Printf("Ошибка чтения сообщений: %v", err)
			return
		}
		sendUserMessagesList(bot, chatID, messages)
	default:
		if userStates[chatID] == "awaiting_message" {
			id, err := database.SaveMessage(chatID, update.Message.From.UserName, text)
			if err != nil {
				log.Printf("Ошибка сохранения сообщения в базу данных: %v", err)
				bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при сохранении вашего сообщения."))
				return
			}
			notifyAdmin(bot, adminID, update.Message.From.UserName, text, id)
			bot.Send(tgbotapi.NewMessage(chatID, "Ваше сообщение отправлено администратору."))
			userStates[chatID] = ""
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, "Неизвестная команда. Вы можете использовать команды:\n/contact - связаться с администратором\n/see_questions - посмотреть на ваши вопросы."))
		}
	}
}

func parseQuestionID(text string) (int, error) {
	if strings.HasPrefix(text, "/answer_") {
		return strconv.Atoi(strings.TrimPrefix(text, "/answer_"))
	}
	return strconv.Atoi(text)
}

func sendMessagesList(bot *tgbotapi.BotAPI, chatID int64, messages []db.Message, emptyMsg string) {
	var response strings.Builder
	for _, msg := range messages {
		response.WriteString(fmt.Sprintf("ID: %d\nПользователь: %s\nСообщение: %s\nОтветить: /answer_%d\n", msg.ID, msg.Username, msg.Question, msg.ID))
	}
	if response.Len() == 0 {
		response.WriteString(emptyMsg)
	}
	bot.Send(tgbotapi.NewMessage(chatID, response.String()))
}

func sendAnsweredList(bot *tgbotapi.BotAPI, chatID int64, messages []db.Message, emptyMsg string) {
	var response strings.Builder
	for _, msg := range messages {
		response.WriteString(fmt.Sprintf("ID: %d\nПользователь: %s\nВопрос:\n%s\nОтвет:\n%s\n\n", msg.ID, msg.Username, msg.Question, msg.Answer))
	}
	if response.Len() == 0 {
		response.WriteString(emptyMsg)
	}
	bot.Send(tgbotapi.NewMessage(chatID, response.String()))
}

func sendUserMessagesList(bot *tgbotapi.BotAPI, chatID int64, messages []db.Message) {
	var response strings.Builder
	response.WriteString("Ваши вопросы\n\n")
	for _, msg := range messages {
		response.WriteString(fmt.Sprintf("Вопрос:\n%s\nОтвет:\n%s\n\n", msg.Question, msg.Answer))
	}
	if len(messages) == 0 {
		response.WriteString("У Вас нет вопросов.")
	}
	bot.Send(tgbotapi.NewMessage(chatID, response.String()))
}

func sendAnswerToUser(bot *tgbotapi.BotAPI, userID int64, question, answer string) {
	responseMsg := fmt.Sprintf("Вы получили новый ответ!\nВаш вопрос:\n%v\nОтвет администратора:\n%v\n", question, answer)
	bot.Send(tgbotapi.NewMessage(userID, responseMsg))
}

func notifyAdmin(bot *tgbotapi.BotAPI, adminID int64, username, message string, id int64) {
	adminMsg := fmt.Sprintf("Новый вопрос\nОт: @%s\nВопрос:\n%s\nОтветить: /answer_%d", username, message, id)
	bot.Send(tgbotapi.NewMessage(adminID, adminMsg))
}
