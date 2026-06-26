package com.hive.jiangminzhi;

import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Map;

@Service
public class AchievementServiceImpl implements AchievementService {

    @Override
    public List<Map<String, Object>> listAchievements(long userId) {
        return List.of(Map.of("code", "FIRST_LOGIN", "unlocked", false));
    }

    @Override
    public void handleDomainEvent(Object event) {
    }
}
