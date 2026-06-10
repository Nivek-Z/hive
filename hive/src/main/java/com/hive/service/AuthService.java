package com.hive.service;

import com.hive.common.BizException;
import com.hive.mapper.UserMapper;
import com.hive.model.User;
import com.hive.model.dto.LoginReq;
import com.hive.model.dto.LoginResp;
import com.hive.model.dto.RegisterReq;
import com.hive.model.dto.UserVO;
import com.hive.util.JwtUtil;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.concurrent.ThreadLocalRandom;

/**
 * 注册与登录。密码使用 BCrypt 加盐哈希存储，登录成功签发 JWT。
 */
@Service
public class AuthService {

    /** 新用户随机分配的头像底色调色板 */
    private static final String[] AVATAR_PALETTE = {
            "#FFB300", "#FF7043", "#EC407A", "#AB47BC",
            "#5C6BC0", "#29B6F6", "#26A69A", "#9CCC65"};

    private final UserMapper userMapper;
    private final JwtUtil jwtUtil;
    private final PasswordEncoder passwordEncoder;

    public AuthService(UserMapper userMapper, JwtUtil jwtUtil, PasswordEncoder passwordEncoder) {
        this.userMapper = userMapper;
        this.jwtUtil = jwtUtil;
        this.passwordEncoder = passwordEncoder;
    }

    @Transactional
    public LoginResp register(RegisterReq req) {
        if (userMapper.findByUsername(req.username()) != null) {
            throw new BizException("用户名已被占用");
        }
        User user = new User();
        user.setUsername(req.username());
        user.setPasswordHash(passwordEncoder.encode(req.password()));
        user.setNickname(req.nickname());
        user.setAvatarColor(AVATAR_PALETTE[ThreadLocalRandom.current().nextInt(AVATAR_PALETTE.length)]);
        userMapper.insert(user);
        // 注册成功直接登录
        return issueToken(userMapper.findById(user.getId()));
    }

    public LoginResp login(LoginReq req) {
        User user = userMapper.findByUsername(req.username());
        if (user == null || !passwordEncoder.matches(req.password(), user.getPasswordHash())) {
            // 统一提示，不暴露"用户是否存在"
            throw new BizException("用户名或密码错误");
        }
        userMapper.touchLastSeen(user.getId());
        return issueToken(user);
    }

    private LoginResp issueToken(User user) {
        String token = jwtUtil.create(user.getId(), user.getUsername());
        return new LoginResp(token, UserVO.from(user));
    }
}
