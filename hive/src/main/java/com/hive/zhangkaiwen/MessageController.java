package com.hive.zhangkaiwen;

import com.hive.zhangzhishuo.ApiResponse;
import com.hive.jiangminzhi.CurrentUid;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api")
public class MessageController {

    /** 标记已读请求 */
    public record ReadReq(@NotNull(message = "缺少消息id") Long lastMessageId) {
    }

    /** 添加表情回应请求 */
    public record ReactReq(@NotBlank(message = "表情不能为空") String emoji) {
    }

    private final MessageService messageService;

    public MessageController(MessageService messageService) {
        this.messageService = messageService;
    }

    /** 历史消息（游标分页：?before=消息id&limit=50，不传 before 取最新一页） */
    @GetMapping("/channels/{channelId}/messages")
    public ApiResponse<List<MessageVO>> history(@CurrentUid long uid,
                                                @PathVariable long channelId,
                                                @RequestParam(required = false) Long before,
                                                @RequestParam(required = false) Integer limit) {
        return ApiResponse.ok(messageService.history(uid, channelId, before, limit));
    }

    @PostMapping("/channels/{channelId}/read")
    public ApiResponse<Void> markRead(@CurrentUid long uid, @PathVariable long channelId,
                                      @Valid @RequestBody ReadReq req) {
        messageService.markRead(uid, channelId, req.lastMessageId());
        return ApiResponse.ok();
    }

    /** 撤回（软删除）：自己的消息或持有"删除他人消息"权限 */
    @DeleteMapping("/messages/{id}")
    public ApiResponse<Void> delete(@CurrentUid long uid, @PathVariable long id) {
        messageService.delete(uid, id);
        return ApiResponse.ok();
    }

    @PostMapping("/messages/{id}/reactions")
    public ApiResponse<List<ReactionVO>> addReaction(@CurrentUid long uid, @PathVariable long id,
                                                     @Valid @RequestBody ReactReq req) {
        return ApiResponse.ok(messageService.react(uid, id, req.emoji(), true));
    }

    @DeleteMapping("/messages/{id}/reactions/{emoji}")
    public ApiResponse<List<ReactionVO>> removeReaction(@CurrentUid long uid, @PathVariable long id,
                                                        @PathVariable String emoji) {
        return ApiResponse.ok(messageService.react(uid, id, emoji, false));
    }
}
