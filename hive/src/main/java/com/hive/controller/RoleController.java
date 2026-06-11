package com.hive.controller;

import com.hive.common.ApiResponse;
import com.hive.config.CurrentUid;
import com.hive.model.dto.RoleReq;
import com.hive.model.dto.RoleVO;
import com.hive.service.RoleService;
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
@RequestMapping("/api")
public class RoleController {

    /** 重设成员角色请求 */
    public record AssignRolesReq(List<Long> roleIds) {
    }

    private final RoleService roleService;

    public RoleController(RoleService roleService) {
        this.roleService = roleService;
    }

    @GetMapping("/hives/{hiveId}/roles")
    public ApiResponse<List<RoleVO>> list(@CurrentUid long uid, @PathVariable long hiveId) {
        return ApiResponse.ok(roleService.list(uid, hiveId));
    }

    @PostMapping("/hives/{hiveId}/roles")
    public ApiResponse<RoleVO> create(@CurrentUid long uid, @PathVariable long hiveId,
                                      @Valid @RequestBody RoleReq req) {
        return ApiResponse.ok(roleService.create(uid, hiveId, req));
    }

    @PutMapping("/roles/{id}")
    public ApiResponse<RoleVO> update(@CurrentUid long uid, @PathVariable long id,
                                      @Valid @RequestBody RoleReq req) {
        return ApiResponse.ok(roleService.update(uid, id, req));
    }

    @DeleteMapping("/roles/{id}")
    public ApiResponse<Void> delete(@CurrentUid long uid, @PathVariable long id) {
        roleService.delete(uid, id);
        return ApiResponse.ok();
    }

    @PutMapping("/hives/{hiveId}/members/{userId}/roles")
    public ApiResponse<Void> assign(@CurrentUid long uid, @PathVariable long hiveId,
                                    @PathVariable long userId, @RequestBody AssignRolesReq req) {
        roleService.assign(uid, hiveId, userId, req.roleIds());
        return ApiResponse.ok();
    }
}
