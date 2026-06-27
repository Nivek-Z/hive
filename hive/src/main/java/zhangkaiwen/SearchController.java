package zhangkaiwen;

import jiangminzhi.CurrentUid;
import yupeiyuan.PermissionService;
import zhangzhishuo.ApiResponse;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;
import java.util.Map;

/**
 * 全文检索和消息统计入口。
 */
@RestController
@RequestMapping("/api")
public class SearchController {

    private final MessageMapper messageMapper;
    private final PermissionService permissionService;

    public SearchController(MessageMapper messageMapper, PermissionService permissionService) {
        this.messageMapper = messageMapper;
        this.permissionService = permissionService;
    }

    /** 个人聊天热力图：过去一年逐日消息数。 */
    @GetMapping("/users/me/heatmap")
    public ApiResponse<List<HeatRow>> heatmap(@CurrentUid long uid) {
        return ApiResponse.ok(messageMapper.heatmap(uid));
    }

    /** 蜂巢内中文全文搜索。 */
    @GetMapping("/search/messages")
    public ApiResponse<List<SearchHit>> search(@CurrentUid long uid,
                                               @RequestParam long hiveId,
                                               @RequestParam String q) {
        permissionService.requireHive(hiveId);
        permissionService.requireMember(hiveId, uid);
        String keyword = q == null ? "" : q.strip();
        if (keyword.isEmpty()) {
            return ApiResponse.ok(List.of());
        }
        return ApiResponse.ok(messageMapper.search(hiveId, keyword));
    }

    /** 蜂巢活跃统计：近 7 日消息量 + 发言排行。 */
    @GetMapping("/hives/{id}/stats")
    public ApiResponse<Map<String, Object>> stats(@CurrentUid long uid, @PathVariable long id) {
        permissionService.requireHive(id);
        permissionService.requireMember(id, uid);
        return ApiResponse.ok(Map.of(
                "daily", messageMapper.hiveDaily(id),
                "topSpeakers", messageMapper.hiveTopSpeakers(id)));
    }
}
