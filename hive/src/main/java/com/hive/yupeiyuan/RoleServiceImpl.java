package com.hive.yupeiyuan;

import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Map;

@Service
public class RoleServiceImpl implements RoleService {

    @Override
    public Map<String, Object> createRole(long operatorId, long hiveId, Map<String, Object> request) {
        return Map.of("id", 1L, "hiveId", hiveId, "name", request.getOrDefault("name", "draft-role"));
    }

    @Override
    public List<Map<String, Object>> listRoles(long operatorId, long hiveId) {
        return List.of(Map.of("id", 1L, "name", "member"));
    }

    @Override
    public void updateMemberRoles(long operatorId, long hiveId, long userId, List<Long> roleIds) {
    }
}
