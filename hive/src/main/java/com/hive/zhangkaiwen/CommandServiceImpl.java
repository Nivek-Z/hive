package com.hive.zhangkaiwen;

import org.springframework.stereotype.Service;

import java.util.Optional;

@Service
public class CommandServiceImpl implements CommandService {

    @Override
    public Optional<String> tryExecute(long userId, String rawContent) {
        return rawContent != null && rawContent.startsWith("/") ? Optional.of("draft command") : Optional.empty();
    }
}
