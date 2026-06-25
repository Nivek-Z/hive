package com.hive.jiangminzhi;


import java.time.LocalDateTime;

/** 用户公开信息视图（不含密码哈希等敏感字段） */
public record UserVO(
        Long id,
        String username,
        String nickname,
        String avatarColor,
        String avatarUrl,
        String bio,
        LocalDateTime createdAt) {

    public static UserVO from(User u) {
        return new UserVO(u.getId(), u.getUsername(), u.getNickname(),
                u.getAvatarColor(), u.getAvatarUrl(), u.getBio(), u.getCreatedAt());
    }
}
