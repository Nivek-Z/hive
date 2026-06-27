package jiangminzhi;

import java.time.LocalDateTime;

/** 成就视图（含当前用户解锁状态；POJO 供 MyBatis 连表自动映射） */
public class AchievementVO {

    private Long id;
    private String code;
    private String name;
    private String description;
    private String emoji;
    private Boolean secret;
    private Integer points;
    private LocalDateTime unlockedAt;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getCode() { return code; }
    public void setCode(String code) { this.code = code; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }
    public String getEmoji() { return emoji; }
    public void setEmoji(String emoji) { this.emoji = emoji; }
    public Boolean getSecret() { return secret; }
    public void setSecret(Boolean secret) { this.secret = secret; }
    public Integer getPoints() { return points; }
    public void setPoints(Integer points) { this.points = points; }
    public LocalDateTime getUnlockedAt() { return unlockedAt; }
    public void setUnlockedAt(LocalDateTime unlockedAt) { this.unlockedAt = unlockedAt; }
}
