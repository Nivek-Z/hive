package com.hive.zhangzhishuo;

import com.hive.jiangminzhi.CurrentUid;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api/hives")
public class HiveController {

    private final HiveService hiveService;

    public HiveController(HiveService hiveService) {
        this.hiveService = hiveService;
    }

    @PostMapping
    public ApiResponse<HiveDetailVO> create(@CurrentUid long uid, @Valid @RequestBody HiveReq req) {
        return ApiResponse.ok(hiveService.create(uid, req));
    }

    @GetMapping
    public ApiResponse<List<HiveVO>> myHives(@CurrentUid long uid) {
        return ApiResponse.ok(hiveService.myHives(uid));
    }

    @GetMapping("/{id}")
    public ApiResponse<HiveDetailVO> detail(@CurrentUid long uid, @PathVariable long id) {
        return ApiResponse.ok(hiveService.detail(uid, id));
    }

    @PutMapping("/{id}")
    public ApiResponse<HiveVO> update(@CurrentUid long uid, @PathVariable long id,
                                      @Valid @RequestBody HiveReq req) {
        return ApiResponse.ok(hiveService.update(uid, id, req));
    }

    @DeleteMapping("/{id}")
    public ApiResponse<Void> delete(@CurrentUid long uid, @PathVariable long id) {
        hiveService.delete(uid, id);
        return ApiResponse.ok();
    }

    @PostMapping("/{id}/leave")
    public ApiResponse<Void> leave(@CurrentUid long uid, @PathVariable long id) {
        hiveService.leave(uid, id);
        return ApiResponse.ok();
    }

    // ---------- 成员 ----------

    @GetMapping("/{id}/members")
    public ApiResponse<List<MemberVO>> members(@CurrentUid long uid, @PathVariable long id) {
        return ApiResponse.ok(hiveService.members(uid, id));
    }

    @DeleteMapping("/{id}/members/{userId}")
    public ApiResponse<Void> kick(@CurrentUid long uid, @PathVariable long id,
                                  @PathVariable long userId) {
        hiveService.kick(uid, id, userId);
        return ApiResponse.ok();
    }

    @PostMapping("/{id}/members/{userId}/mute")
    public ApiResponse<Void> mute(@CurrentUid long uid, @PathVariable long id,
                                  @PathVariable long userId, @Valid @RequestBody MuteReq req) {
        hiveService.mute(uid, id, userId, req.minutes());
        return ApiResponse.ok();
    }

    @DeleteMapping("/{id}/members/{userId}/mute")
    public ApiResponse<Void> unmute(@CurrentUid long uid, @PathVariable long id,
                                    @PathVariable long userId) {
        hiveService.unmute(uid, id, userId);
        return ApiResponse.ok();
    }

    // ---------- 邀请码 ----------

    @PostMapping("/{id}/invites")
    public ApiResponse<InviteVO> createInvite(@CurrentUid long uid, @PathVariable long id,
                                              @Valid @RequestBody CreateInviteReq req) {
        return ApiResponse.ok(hiveService.createInvite(uid, id, req));
    }

    @GetMapping("/{id}/invites")
    public ApiResponse<List<InviteVO>> listInvites(@CurrentUid long uid, @PathVariable long id) {
        return ApiResponse.ok(hiveService.listInvites(uid, id));
    }
}
