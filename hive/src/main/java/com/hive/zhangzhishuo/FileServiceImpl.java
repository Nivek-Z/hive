package com.hive.zhangzhishuo;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Map;
import java.util.UUID;

@Service
public class FileServiceImpl implements FileService {

    private static final Map<String, String> ALLOWED_MIME = Map.of(
            "image/png", "png",
            "image/jpeg", "jpg",
            "image/gif", "gif",
            "image/webp", "webp");

    private final FileMapper fileMapper;
    private final Path uploadDir;

    public FileServiceImpl(FileMapper fileMapper, @Value("${hive.upload-dir}") String uploadDir) {
        this.fileMapper = fileMapper;
        this.uploadDir = Paths.get(uploadDir).toAbsolutePath().normalize();
    }

    @Override
    public FileVO upload(long uid, MultipartFile file) throws IOException {
        if (file.isEmpty()) {
            throw new BizException("文件为空");
        }
        String ext = ALLOWED_MIME.get(file.getContentType());
        if (ext == null) {
            throw new BizException("仅支持 PNG / JPG / GIF / WebP 图片");
        }
        String storedName = UUID.randomUUID().toString().replace("-", "") + "." + ext;
        Files.createDirectories(uploadDir);
        file.transferTo(uploadDir.resolve(storedName));

        StoredFile record = new StoredFile();
        record.setUploaderId(uid);
        record.setStoredName(storedName);
        String original = file.getOriginalFilename();
        record.setOriginalName(original == null ? storedName : original);
        record.setMime(file.getContentType());
        record.setSizeBytes(file.getSize());
        fileMapper.insert(record);

        return new FileVO("/uploads/" + storedName, record.getOriginalName(), file.getSize());
    }
}