package com.hive.zhangzhishuo;

import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Map;

@Service
public class HiveServiceImpl implements HiveService {

    @Override
    public Map<String, Object> createHive(long ownerId, Map<String, Object> request) {
        return Map.of("id", 1L, "ownerId", ownerId, "stage", "foundation");
    }

    @Override
    public List<Map<String, Object>> listJoinedHives(long userId) {
        return List.of(Map.of("id", 1L, "name", "draft-hive"));
    }

    @Override
    public Map<String, Object> getHiveDetail(long userId, long hiveId) {
        return Map.of("id", hiveId, "member", userId);
    }

    @Override
    public String createInvite(long userId, long hiveId, Map<String, Object> request) {
        return "DRAFT001";
    }

    @Override
    public void joinByInvite(long userId, String inviteCode) {
    }
}
