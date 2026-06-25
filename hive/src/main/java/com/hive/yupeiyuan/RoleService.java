package com.hive.yupeiyuan;

import java.util.List;
import java.util.Map;

/**
 * 虞沛远负责：角色创建、修改和成员角色分配。
 */
public interface RoleService {

    Map<String, Object> createRole(long operatorId, long hiveId, Map<String, Object> request);

    List<Map<String, Object>> listRoles(long operatorId, long hiveId);

    void updateMemberRoles(long operatorId, long hiveId, long userId, List<Long> roleIds);
}
