# tg-admin-bot

В данном боте есть 2 роли: администратор и пользователь
Цель бота: перенаправлять вопросы пользователей администратору и ответы администратора пользователям

Доступные команды для пользователя:
`/start` - Начать бота
`/contact` - Написать вопрос администратору
`/see_questions` - Посмотреть на вопросы и ответы

Доступные команды для администратора:
`/start` - Начать бота
`/see_questions` - Посмотреть на неотвеченные вопросы
`/answer` - Позволяет ответить на вопрос
`/see_answers` - Посмотреть на отвеченные вопросы

Программа использует .env файл для хранения токена для бота, chatID администратора, путь к базе данных.

База данных - sqlite3, с применением миграций.
