package com.hive.zhangzhishuo;

import com.hive.common.ApiResponse;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

@RestController
@RequestMapping("/api/invites")
public class InviteController {

    private final HiveService hiveService;

    public InviteController(HiveService hiveService) {
        this.hiveService = hiveService;
    }

    @PostMapping("/{code}/join")
    public ApiResponse<Map<String, Object>> join(@PathVariable String code) {
        hiveService.joinByInvite(0L, code);
        return ApiResponse.ok(Map.of("joined", true));
    }
}
