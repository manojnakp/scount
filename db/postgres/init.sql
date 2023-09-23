CREATE TABLE IF NOT EXISTS users
(
    uid      TEXT NOT NULL,
    email    TEXT NOT NULL,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    PRIMARY KEY (uid),
    UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS scounts
(
    sid         TEXT NOT NULL,
    owner       TEXT NOT NULL,
    title       TEXT NOT NULL,
    description TEXT NOT NULL,
    FOREIGN KEY (owner) REFERENCES users (uid),
    PRIMARY KEY (sid)
);

CREATE TABLE IF NOT EXISTS members (
    sid TEXT NOT NULL,
    uid TEXT NOT NULL,
    FOREIGN KEY (sid) REFERENCES scounts (sid),
    FOREIGN KEY (uid) REFERENCES users (uid),
    PRIMARY KEY (sid, uid)
);
