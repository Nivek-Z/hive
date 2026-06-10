package com.hive.controller;

import com.hive.common.ApiResponse;
import com.hive.config.CurrentUid;
import com.hive.model.dto.ChangePasswordReq;
import com.hive.model.dto.UpdateProfileReq;
import com.hive.model.dto.UserVO;
import com.hive.service.UserService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/users")
public class UserController {

    private final UserService userService;

    public UserController(UserService userService) {
        this.userService = userService;
    }

    @GetMapping("/me")
    public ApiResponse<UserVO> me(@CurrentUid long uid) {
        return ApiResponse.ok(userService.profile(uid));
    }

    @PutMapping("/me")
    public ApiResponse<UserVO> updateProfile(@CurrentUid long uid,
                                             @Valid @RequestBody UpdateProfileReq req) {
        return ApiResponse.ok(userService.updateProfile(uid, req));
    }

    @PutMapping("/me/password")
    public ApiResponse<Void> changePassword(@CurrentUid long uid,
                                            @Valid @RequestBody ChangePasswordReq req) {
        userService.changePassword(uid, req);
        return ApiResponse.ok();
    }

    @GetMapping("/{id}")
    public ApiResponse<UserVO> profile(@PathVariable long id) {
        return ApiResponse.ok(userService.profile(id));
    }
}
