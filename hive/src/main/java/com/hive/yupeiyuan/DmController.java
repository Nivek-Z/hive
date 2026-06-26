package com.hive.yupeiyuan;

import com.hive.common.ApiResponse;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

@RestController
@RequestMapping("/api/dms")
public class DmController {

    private final FriendService friendService;

    public DmController(FriendService friendService) {
        this.friendService = friendService;
    }

    @PostMapping("/{friendId}")
    public ApiResponse<Map<String, Object>> open(@PathVariable long friendId) {
        return ApiResponse.ok(friendService.openDm(0L, friendId));
    }
}
