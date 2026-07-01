package jiangminzhi;

import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface AchievementMapper {

    @Select("SELECT * FROM achievements WHERE code = #{code}")
    AchievementVO findByCode(@Param("code") String code);

    /** 全部成就 + 当前用户解锁时间（未解锁为 NULL） */
    @Select("SELECT a.*, ua.unlocked_at FROM achievements a " +
            "LEFT JOIN user_achievements ua ON ua.achievement_id = a.id AND ua.user_id = #{uid} " +
            "ORDER BY a.id")
    List<AchievementVO> listWithStatus(@Param("uid") long uid);

    /** INSERT IGNORE：返回 1=首次解锁，0=已解锁过 */
    @Insert("INSERT IGNORE INTO user_achievements(user_id, achievement_id) VALUES(#{uid}, #{achievementId})")
    int unlock(@Param("uid") long uid, @Param("achievementId") long achievementId);
}
