package com.hive.zhangkaiwen;

import java.util.Optional;

/**
 * 张凯文负责：聊天斜杠命令的解析和结果生成。
 */
public interface CommandService {

    Optional<String> tryExecute(long userId, String rawContent);
}
