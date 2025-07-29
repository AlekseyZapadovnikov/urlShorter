-- создет роль пользователя urlshortner
CREATE ROLE urlshortner WITH LOGIN PASSWORD '123';

-- создаёт БД и назначить владельцем пользователя urlshortner
CREATE DATABASE "url-shrtner" OWNER urlshortner;

-- подключиться к только что созданной базе
\connect url-shrtner;

-- сам скрипт для создания таблиц
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    mail TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS urls (
    id SERIAL PRIMARY KEY,
    short_code TEXT UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- даем нашей роли права на использование
GRANT ALL PRIVILEGES ON TABLE users TO urlshortner;
GRANT ALL PRIVILEGES ON TABLE urls TO urlshortner;

GRANT USAGE, SELECT ON SEQUENCE users_id_seq TO urlshortner;
GRANT USAGE, SELECT ON SEQUENCE urls_id_seq TO urlshortner;