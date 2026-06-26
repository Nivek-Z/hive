package com.hive.yupeiyuan;

import com.hive.zhangzhishuo.ApiResponse;
import com.hive.jiangminzhi.CurrentUid;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api/dms")
public class DmController {

    /** 打开私聊响应：DM 频道 id */
    public record OpenDmResp(long channelId) {
    }

    private final FriendService friendService;

    public DmController(FriendService friendService) {
        this.friendService = friendService;
    }

    /** 打开（不存在则创建）与某好友的私聊会话 */
    @PostMapping("/{userId}")
    public ApiResponse<OpenDmResp> open(@CurrentUid long uid, @PathVariable long userId) {
        return ApiResponse.ok(new OpenDmResp(friendService.openDm(uid, userId)));
    }

    /** 我的私聊会话列表（带最后一条消息与未读数） */
    @GetMapping
    public ApiResponse<List<DmVO>> list(@CurrentUid long uid) {
        return ApiResponse.ok(friendService.listDms(uid));
    }
}
