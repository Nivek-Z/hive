-- ============================================================
-- Hive 蜂巢 · 数据库结构 (MySQL 8, utf8mb4 支持 emoji)
-- 全部 IF NOT EXISTS：应用每次启动自动执行，幂等
-- ============================================================

-- 1. 用户
CREATE TABLE IF NOT EXISTS users (
    id            BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    username      VARCHAR(32)  NOT NULL COMMENT '登录名，唯一',
    password_hash VARCHAR(100) NOT NULL COMMENT 'BCrypt 加盐哈希，不存明文',
    nickname      VARCHAR(32)  NOT NULL COMMENT '显示昵称',
    avatar_color  CHAR(7)      NOT NULL DEFAULT '#FFB300' COMMENT '六边形头像底色',
    avatar_url    VARCHAR(255) DEFAULT NULL COMMENT '上传头像后覆盖底色头像',
    bio           VARCHAR(200) NOT NULL DEFAULT '' COMMENT '个性签名',
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at  DATETIME     DEFAULT NULL COMMENT '最后在线时间',
    UNIQUE KEY uk_users_username (username)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '用户表';

-- 2. 蜂巢（社区/服务器）
CREATE TABLE IF NOT EXISTS hives (
    id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name        VARCHAR(50)  NOT NULL,
    description VARCHAR(200) NOT NULL DEFAULT '',
    icon_color  CHAR(7)      NOT NULL DEFAULT '#FFB300',
    owner_id    BIGINT       NOT NULL COMMENT '巢主(蜂后)，拥有全部权限',
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_hives_owner (owner_id),
    CONSTRAINT fk_hives_owner FOREIGN KEY (owner_id) REFERENCES users (id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '蜂巢(群组)表';

-- 3. 邀请码
CREATE TABLE IF NOT EXISTS invites (
    id         BIGINT   NOT NULL AUTO_INCREMENT PRIMARY KEY,
    code       CHAR(8)  NOT NULL COMMENT '8位随机邀请码',
    hive_id    BIGINT   NOT NULL,
    creator_id BIGINT   NOT NULL,
    max_uses   INT      NOT NULL DEFAULT 0 COMMENT '最大使用次数，0=不限',
    used_count INT      NOT NULL DEFAULT 0,
    expires_at DATETIME DEFAULT NULL COMMENT 'NULL=永不过期',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_invites_code (code),
    KEY idx_invites_hive (hive_id),
    CONSTRAINT fk_invites_hive FOREIGN KEY (hive_id) REFERENCES hives (id) ON DELETE CASCADE,
    CONSTRAINT fk_invites_creator FOREIGN KEY (creator_id) REFERENCES users (id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '邀请码表';

-- 4. 蜂巢成员
CREATE TABLE IF NOT EXISTS hive_members (
    id            BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    hive_id       BIGINT      NOT NULL,
    user_id       BIGINT      NOT NULL,
    hive_nickname VARCHAR(32) DEFAULT NULL COMMENT '巢内昵称，NULL则用全局昵称',
    muted_until   DATETIME    DEFAULT NULL COMMENT '禁言截止时间',
    joined_at     DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_member (hive_id, user_id),
    KEY idx_member_user (user_id),
    CONSTRAINT fk_member_hive FOREIGN KEY (hive_id) REFERENCES hives (id) ON DELETE CASCADE,
    CONSTRAINT fk_member_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '蜂巢成员表';

-- 5. 频道（树形结构：parent_id 自引用实现"群中群"；DM=私聊频道）
CREATE TABLE IF NOT EXISTS channels (
    id         BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    hive_id    BIGINT      DEFAULT NULL COMMENT '私聊频道为 NULL',
    parent_id  BIGINT      DEFAULT NULL COMMENT '父频道/分区，自引用=群中群',
    type       ENUM ('CATEGORY','TEXT','DM') NOT NULL DEFAULT 'TEXT',
    name       VARCHAR(50) NOT NULL,
    topic      VARCHAR(200) NOT NULL DEFAULT '' COMMENT '频道主题',
    position   INT         NOT NULL DEFAULT 0 COMMENT '排序',
    created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_channels_hive (hive_id),
    KEY idx_channels_parent (parent_id),
    CONSTRAINT fk_channels_hive FOREIGN KEY (hive_id) REFERENCES hives (id) ON DELETE CASCADE,
    CONSTRAINT fk_channels_parent FOREIGN KEY (parent_id) REFERENCES channels (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '频道表(树形)';

-- 6. 频道成员（仅私聊 DM 频道使用）
CREATE TABLE IF NOT EXISTS channel_members (
    id         BIGINT   NOT NULL AUTO_INCREMENT PRIMARY KEY,
    channel_id BIGINT   NOT NULL,
    user_id    BIGINT   NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_cm (channel_id, user_id),
    KEY idx_cm_user (user_id),
    CONSTRAINT fk_cm_channel FOREIGN KEY (channel_id) REFERENCES channels (id) ON DELETE CASCADE,
    CONSTRAINT fk_cm_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '私聊频道参与者';

-- 7. 消息（软删除；毫秒时间戳；ngram 中文全文索引）
CREATE TABLE IF NOT EXISTS messages (
    id          BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    channel_id  BIGINT      NOT NULL,
    sender_id   BIGINT      DEFAULT NULL COMMENT 'NULL=系统消息',
    type        ENUM ('TEXT','IMAGE','SYSTEM') NOT NULL DEFAULT 'TEXT',
    content     TEXT        NOT NULL,
    reply_to_id BIGINT      DEFAULT NULL COMMENT '回复引用的消息id',
    deleted     TINYINT(1)  NOT NULL DEFAULT 0 COMMENT '软删除(撤回)',
    created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_msg_channel (channel_id, id),
    KEY idx_msg_sender_time (sender_id, created_at),
    FULLTEXT KEY ft_msg_content (content) WITH PARSER ngram,
    CONSTRAINT fk_msg_channel FOREIGN KEY (channel_id) REFERENCES channels (id) ON DELETE CASCADE,
    CONSTRAINT fk_msg_sender FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '消息表';

-- 8. 表情回应
CREATE TABLE IF NOT EXISTS reactions (
    id         BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    message_id BIGINT      NOT NULL,
    user_id    BIGINT      NOT NULL,
    emoji      VARCHAR(16) NOT NULL,
    created_at DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_reaction (message_id, user_id, emoji),
    KEY idx_reaction_user (user_id),
    CONSTRAINT fk_react_msg FOREIGN KEY (message_id) REFERENCES messages (id) ON DELETE CASCADE,
    CONSTRAINT fk_react_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '消息表情回应';

-- 9. 好友关系（requester 发起申请，accepted 后互为好友）
CREATE TABLE IF NOT EXISTS friendships (
    id           BIGINT   NOT NULL AUTO_INCREMENT PRIMARY KEY,
    requester_id BIGINT   NOT NULL,
    addressee_id BIGINT   NOT NULL,
    status       ENUM ('PENDING','ACCEPTED') NOT NULL DEFAULT 'PENDING',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_friend_pair (requester_id, addressee_id),
    KEY idx_friend_addressee (addressee_id, status),
    CONSTRAINT fk_friend_req FOREIGN KEY (requester_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_friend_addr FOREIGN KEY (addressee_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '好友关系表';

-- 10. 角色（permissions 为权限位掩码，Discord 同款设计）
CREATE TABLE IF NOT EXISTS roles (
    id          BIGINT      NOT NULL AUTO_INCREMENT PRIMARY KEY,
    hive_id     BIGINT      NOT NULL,
    name        VARCHAR(32) NOT NULL,
    color       CHAR(7)     NOT NULL DEFAULT '#99AAB5',
    permissions BIGINT      NOT NULL DEFAULT 0 COMMENT '权限位掩码(按位OR)',
    position    INT         NOT NULL DEFAULT 0 COMMENT '越大层级越高',
    is_default  TINYINT(1)  NOT NULL DEFAULT 0 COMMENT '默认角色(所有成员自动拥有)',
    CONSTRAINT fk_roles_hive FOREIGN KEY (hive_id) REFERENCES hives (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '角色表';

-- 11. 成员-角色（多对多）
CREATE TABLE IF NOT EXISTS member_roles (
    id      BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    hive_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    UNIQUE KEY uk_member_role (user_id, role_id),
    KEY idx_mr_hive_user (hive_id, user_id),
    CONSTRAINT fk_mr_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
    CONSTRAINT fk_mr_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '成员角色关联表';

-- 12. 已读状态（未读红点：频道最新消息id - 已读id）
CREATE TABLE IF NOT EXISTS read_states (
    id                   BIGINT   NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id              BIGINT   NOT NULL,
    channel_id           BIGINT   NOT NULL,
    last_read_message_id BIGINT   NOT NULL DEFAULT 0,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_read (user_id, channel_id),
    CONSTRAINT fk_read_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_read_channel FOREIGN KEY (channel_id) REFERENCES channels (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '频道已读状态';

-- 13. 成就定义
CREATE TABLE IF NOT EXISTS achievements (
    id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    code        VARCHAR(32)  NOT NULL COMMENT '程序内引用的唯一编码',
    name        VARCHAR(32)  NOT NULL,
    description VARCHAR(120) NOT NULL,
    emoji       VARCHAR(8)   NOT NULL DEFAULT '🏆',
    secret      TINYINT(1)   NOT NULL DEFAULT 0 COMMENT '隐藏成就(解锁前显示???)',
    points      INT          NOT NULL DEFAULT 10 COMMENT '成就点数',
    UNIQUE KEY uk_ach_code (code)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '成就定义表';

-- 14. 用户成就（解锁记录）
CREATE TABLE IF NOT EXISTS user_achievements (
    id             BIGINT   NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id        BIGINT   NOT NULL,
    achievement_id BIGINT   NOT NULL,
    unlocked_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_ach (user_id, achievement_id),
    CONSTRAINT fk_ua_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_ua_ach FOREIGN KEY (achievement_id) REFERENCES achievements (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '用户成就解锁记录';

-- 15. 上传文件记录
CREATE TABLE IF NOT EXISTS files (
    id            BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    uploader_id   BIGINT       NOT NULL,
    stored_name   VARCHAR(64)  NOT NULL COMMENT '磁盘存储文件名(uuid)',
    original_name VARCHAR(255) NOT NULL,
    mime          VARCHAR(64)  NOT NULL,
    size_bytes    BIGINT       NOT NULL,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_files_stored (stored_name),
    CONSTRAINT fk_files_user FOREIGN KEY (uploader_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT '上传文件表';
