package com.hive.jiangminzhi;

import java.util.List;
import java.util.Map;

/**
 * 江民智负责：成就规则、解锁记录和通知。
 */
public interface AchievementService {

    List<Map<String, Object>> listAchievements(long userId);

    void handleDomainEvent(Object event);
}
