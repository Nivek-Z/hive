package com.hive.zhangzhishuo;

import java.io.InputStream;
import java.util.Map;

/**
 * 张致硕负责：图片上传和文件元数据保存。
 */
public interface FileService {

    Map<String, Object> storeImage(long uploaderId, String originalName, String contentType, long size, InputStream content);
}
