package com.hive.controller;

import com.hive.common.ApiResponse;
import com.hive.config.CurrentUid;
import com.hive.model.dto.FriendRequestVO;
import com.hive.model.dto.FriendVO;
import com.hive.service.FriendService;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotBlank;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api/friends")
public class FriendController {

    /** 发送好友申请请求体 */
    public record AddFriendReq(@NotBlank(message = "请输入对方用户名") String username) {
    }

    private final FriendService friendService;

    public FriendController(FriendService friendService) {
        this.friendService = friendService;
    }

    @GetMapping
    public ApiResponse<List<FriendVO>> list(@CurrentUid long uid) {
        return ApiResponse.ok(friendService.listFriends(uid));
    }

    @PostMapping("/requests")
    public ApiResponse<Void> sendRequest(@CurrentUid long uid, @Valid @RequestBody AddFriendReq req) {
        friendService.sendRequest(uid, req.username());
        return ApiResponse.ok();
    }

    @GetMapping("/requests")
    public ApiResponse<List<FriendRequestVO>> incoming(@CurrentUid long uid) {
        return ApiResponse.ok(friendService.listIncoming(uid));
    }

    @PostMapping("/requests/{id}/accept")
    public ApiResponse<Void> accept(@CurrentUid long uid, @PathVariable long id) {
        friendService.accept(uid, id);
        return ApiResponse.ok();
    }

    @DeleteMapping("/requests/{id}")
    public ApiResponse<Void> decline(@CurrentUid long uid, @PathVariable long id) {
        friendService.declineOrCancel(uid, id);
        return ApiResponse.ok();
    }

    @DeleteMapping("/{userId}")
    public ApiResponse<Void> remove(@CurrentUid long uid, @PathVariable long userId) {
        friendService.removeFriend(uid, userId);
        return ApiResponse.ok();
    }
}
