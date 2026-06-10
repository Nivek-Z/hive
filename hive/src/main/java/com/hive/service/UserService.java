package com.hive.service;

import com.hive.common.BizException;
import com.hive.mapper.UserMapper;
import com.hive.model.User;
import com.hive.model.dto.ChangePasswordReq;
import com.hive.model.dto.UpdateProfileReq;
import com.hive.model.dto.UserVO;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * 用户资料相关业务。
 */
@Service
public class UserService {

    private final UserMapper userMapper;
    private final PasswordEncoder passwordEncoder;

    public UserService(UserMapper userMapper, PasswordEncoder passwordEncoder) {
        this.userMapper = userMapper;
        this.passwordEncoder = passwordEncoder;
    }

    /** 查询用户，不存在则抛业务异常 */
    public User require(long id) {
        User user = userMapper.findById(id);
        if (user == null) {
            throw BizException.notFound("用户");
        }
        return user;
    }

    public UserVO profile(long id) {
        return UserVO.from(require(id));
    }

    @Transactional
    public UserVO updateProfile(long uid, UpdateProfileReq req) {
        User user = require(uid);
        String bio = req.bio() == null ? user.getBio() : req.bio();
        String color = req.avatarColor() == null ? user.getAvatarColor() : req.avatarColor();
        userMapper.updateProfile(uid, req.nickname(), bio, color);
        return UserVO.from(require(uid));
    }

    @Transactional
    public void changePassword(long uid, ChangePasswordReq req) {
        User user = require(uid);
        if (!passwordEncoder.matches(req.oldPassword(), user.getPasswordHash())) {
            throw new BizException("原密码不正确");
        }
        userMapper.updatePassword(uid, passwordEncoder.encode(req.newPassword()));
    }
}
