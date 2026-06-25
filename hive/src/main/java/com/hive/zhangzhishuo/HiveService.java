package com.hive.zhangzhishuo;

import java.util.List;
import java.util.Map;

/**
 * 张致硕负责：蜂巢创建、成员管理和邀请加入。
 */
public interface HiveService {

    Map<String, Object> createHive(long ownerId, Map<String, Object> request);

    List<Map<String, Object>> listJoinedHives(long userId);

    Map<String, Object> getHiveDetail(long userId, long hiveId);

    String createInvite(long userId, long hiveId, Map<String, Object> request);

    void joinByInvite(long userId, String inviteCode);
}
