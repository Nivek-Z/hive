package com.hive.zhangzhishuo;

import com.hive.jiangminzhi.CurrentUid;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Map;
import java.util.UUID;

/**
 * 图片上传：存磁盘（uploads/uuid.ext），元数据入库（files 表），
 * 返回 /uploads/xxx 静态地址（WebConfig 已做目录映射）。
 */
@RestController
@RequestMapping("/api/files")
public class FileController {

    /** 上传成功响应 */
    public record FileVO(String url, String originalName, long size) {
    }

    /** MIME 白名单 → 扩展名（不信任用户文件名） */
    private static final Map<String, String> ALLOWED_MIME = Map.of(
            "image/png", "png",
            "image/jpeg", "jpg",
            "image/gif", "gif",
            "image/webp", "webp");

    private final FileMapper fileMapper;
    private final Path uploadDir;

    public FileController(FileMapper fileMapper, @Value("${hive.upload-dir}") String uploadDir) {
        this.fileMapper = fileMapper;
        this.uploadDir = Paths.get(uploadDir).toAbsolutePath().normalize();
    }

    @PostMapping
    public ApiResponse<FileVO> upload(@CurrentUid long uid,
                                      @RequestParam("file") MultipartFile file) throws IOException {
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

        return ApiResponse.ok(new FileVO("/uploads/" + storedName, record.getOriginalName(), file.getSize()));
    }
}
