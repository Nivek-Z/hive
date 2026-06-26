package com.hive.zhangzhishuo;

import com.hive.common.ApiResponse;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.io.ByteArrayInputStream;
import java.util.Map;

@RestController
@RequestMapping("/api/files")
public class FileController {

    private final FileService fileService;

    public FileController(FileService fileService) {
        this.fileService = fileService;
    }

    @PostMapping
    public ApiResponse<Map<String, Object>> uploadDraft() {
        return ApiResponse.ok(fileService.storeImage(0L, "draft.png", "image/png", 0L, new ByteArrayInputStream(new byte[0])));
    }
}
