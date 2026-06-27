package zhangzhishuo;

import jiangminzhi.CurrentUid;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/invites")
public class InviteController {

    private final HiveService hiveService;

    public InviteController(HiveService hiveService) {
        this.hiveService = hiveService;
    }

    /** 凭邀请码加入蜂巢 */
    @PostMapping("/{code}/join")
    public ApiResponse<HiveVO> join(@CurrentUid long uid, @PathVariable String code) {
        return ApiResponse.ok(hiveService.join(uid, code));
    }
}
