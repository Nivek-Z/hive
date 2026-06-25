package com.hive.zhangzhishuo;

import com.hive.jiangminzhi.CurrentUid;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;

@RestController
@RequestMapping("/api/files")
public class FileController {

    private final FileService fileService;

    public FileController(FileService fileService) {
        this.fileService = fileService;
    }

    @PostMapping
    public ApiResponse<FileVO> upload(@CurrentUid long uid,
                                      @RequestParam("file") MultipartFile file) throws IOException {
        return ApiResponse.ok(fileService.upload(uid, file));
    }
}