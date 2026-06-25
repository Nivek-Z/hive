package com.hive.jiangminzhi;

import com.hive.zhangzhishuo.ApiResponse;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

/**
 * 成就系统入口。
 */
@RestController
@RequestMapping("/api")
public class AchievementController {

    private final AchievementService achievementService;

    public AchievementController(AchievementService achievementService) {
        this.achievementService = achievementService;
    }

    /** 我的成就墙。 */
    @GetMapping("/users/me/achievements")
    public ApiResponse<List<AchievementVO>> achievements(@CurrentUid long uid) {
        return ApiResponse.ok(achievementService.listFor(uid));
    }

    /** Konami 彩蛋：解锁隐藏成就。 */
    @PostMapping("/eggs/konami")
    public ApiResponse<Void> konami(@CurrentUid long uid) {
        achievementService.unlockKonami(uid);
        return ApiResponse.ok();
    }
}
