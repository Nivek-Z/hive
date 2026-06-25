package com.hive.jiangminzhi;

import com.hive.zhangzhishuo.BizException;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

public interface UserService {

    User require(long id);

    UserVO profile(long id);

    UserVO updateProfile(long uid, UpdateProfileReq req);

    void changePassword(long uid, ChangePasswordReq req);

}
