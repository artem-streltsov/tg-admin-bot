CREATE TABLE IF NOT EXISTS messages_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    username TEXT DEFAULT "",
    question TEXT,
    answered INTEGER DEFAULT 0,
    answer TEXT DEFAULT ""
);

INSERT INTO messages_new (id, user_id, username, question, answered, answer)
SELECT id, user_id, username, message, answered, answer FROM messages;

DROP TABLE messages;

ALTER TABLE messages_new RENAME TO messages;
