CREATE TABLE IF NOT EXISTS messages_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    username TEXT,
    message TEXT,
    answered INTEGER DEFAULT 0,
    answer TEXT DEFAULT ""
);

INSERT INTO messages_old (id, user_id, username, message, answered, answer)
SELECT id, user_id, username, question, answered, answer FROM messages;

DROP TABLE messages;

ALTER TABLE messages_old RENAME TO messages;
