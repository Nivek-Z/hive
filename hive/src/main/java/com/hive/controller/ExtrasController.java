package com.hive.controller;

import com.hive.common.ApiResponse;
import com.hive.config.CurrentUid;
import com.hive.mapper.MessageMapper;
import com.hive.model.dto.AchievementVO;
import com.hive.model.dto.HeatRow;
import com.hive.model.dto.SearchHit;
import com.hive.service.AchievementService;
import com.hive.service.PermissionService;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;
import java.util.Map;

/**
 * 成就 / 热力图 / 全文搜索 / 蜂巢统计 / 彩蛋入口。
 */
@RestController
@RequestMapping("/api")
public class ExtrasController {

    private final AchievementService achievementService;
    private final MessageMapper messageMapper;
    private final PermissionService permissionService;

    public ExtrasController(AchievementService achievementService, MessageMapper messageMapper,
                            PermissionService permissionService) {
        this.achievementService = achievementService;
        this.messageMapper = messageMapper;
        this.permissionService = permissionService;
    }

    /** 我的成就墙（隐藏成就未解锁时打码） */
    @GetMapping("/users/me/achievements")
    public ApiResponse<List<AchievementVO>> achievements(@CurrentUid long uid) {
        return ApiResponse.ok(achievementService.listFor(uid));
    }

    /** 个人聊天热力图（过去一年逐日消息数） */
    @GetMapping("/users/me/heatmap")
    public ApiResponse<List<HeatRow>> heatmap(@CurrentUid long uid) {
        return ApiResponse.ok(messageMapper.heatmap(uid));
    }

    /** Konami 秘技彩蛋：解锁隐藏成就 */
    @PostMapping("/eggs/konami")
    public ApiResponse<Void> konami(@CurrentUid long uid) {
        achievementService.unlockKonami(uid);
        return ApiResponse.ok();
    }

    /** 蜂巢内中文全文搜索（MySQL ngram 全文索引） */
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

    /** 蜂巢活跃统计：近7日消息量 + 发言排行 */
    @GetMapping("/hives/{id}/stats")
    public ApiResponse<Map<String, Object>> stats(@CurrentUid long uid, @PathVariable long id) {
        permissionService.requireHive(id);
        permissionService.requireMember(id, uid);
        return ApiResponse.ok(Map.of(
                "daily", messageMapper.hiveDaily(id),
                "topSpeakers", messageMapper.hiveTopSpeakers(id)));
    }
}
