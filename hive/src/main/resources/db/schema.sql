-- 基础搭建阶段数据库草图，仅用于统一表边界；完整字段和约束在后续阶段细化。

CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(32) NOT NULL,
    nickname VARCHAR(32) NOT NULL
);

CREATE TABLE IF NOT EXISTS hives (
    id BIGINT PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    owner_id BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS channels (
    id BIGINT PRIMARY KEY,
    hive_id BIGINT,
    name VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
    id BIGINT PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    sender_id BIGINT,
    content TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS roles (
    id BIGINT PRIMARY KEY,
    hive_id BIGINT NOT NULL,
    name VARCHAR(32) NOT NULL,
    permissions BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS friendships (
    id BIGINT PRIMARY KEY,
    requester_id BIGINT NOT NULL,
    addressee_id BIGINT NOT NULL,
    status VARCHAR(16) NOT NULL
);

CREATE TABLE IF NOT EXISTS achievements (
    id BIGINT PRIMARY KEY,
    code VARCHAR(32) NOT NULL,
    name VARCHAR(32) NOT NULL
);
