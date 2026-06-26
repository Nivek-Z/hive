package com.hive.zhangzhishuo;

import org.springframework.stereotype.Service;

import java.io.InputStream;
import java.util.Map;

@Service
public class FileServiceImpl implements FileService {

    @Override
    public Map<String, Object> storeImage(long uploaderId, String originalName, String contentType, long size, InputStream content) {
        return Map.of("url", "/uploads/draft.png", "originalName", originalName);
    }
}
