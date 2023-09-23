CREATE TABLE IF NOT EXISTS users
(
    uid      TEXT NOT NULL,
    email    TEXT NOT NULL,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    PRIMARY KEY (uid),
    UNIQUE (email)
);
