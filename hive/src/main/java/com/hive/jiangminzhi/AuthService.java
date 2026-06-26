package com.hive.jiangminzhi;

import com.hive.zhangzhishuo.BizException;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.util.concurrent.ThreadLocalRandom;

public interface AuthService {

    LoginResp register(RegisterReq req);

    LoginResp login(LoginReq req);

}
