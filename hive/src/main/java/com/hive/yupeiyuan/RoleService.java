package com.hive.yupeiyuan;

import com.hive.zhangzhishuo.BizException;
import com.hive.zhangkaiwen.WsPush;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

public interface RoleService {

    List<RoleVO> list(long uid, long hiveId);

    RoleVO create(long uid, long hiveId, RoleReq req);

    RoleVO update(long uid, long roleId, RoleReq req);

    void delete(long uid, long roleId);

    void assign(long uid, long hiveId, long targetId, List<Long> roleIds);

}
