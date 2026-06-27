package zhangkaiwen;

import org.apache.ibatis.annotations.Delete;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface ReactionMapper {

    /** 唯一键(message_id,user_id,emoji)防重复，INSERT IGNORE 幂等 */
    @Insert("INSERT IGNORE INTO reactions(message_id, user_id, emoji) " +
            "VALUES(#{messageId}, #{userId}, #{emoji})")
    int add(@Param("messageId") long messageId,
            @Param("userId") long userId,
            @Param("emoji") String emoji);

    @Delete("DELETE FROM reactions WHERE message_id = #{messageId} AND user_id = #{userId} AND emoji = #{emoji}")
    int remove(@Param("messageId") long messageId,
               @Param("userId") long userId,
               @Param("emoji") String emoji);

    @Select("SELECT message_id, emoji, user_id FROM reactions WHERE message_id = #{messageId} ORDER BY id")
    List<ReactionRow> listByMessage(long messageId);

    /** 历史分页批量取回应：用消息 id 区间避免动态 IN 拼接 */
    @Select("SELECT r.message_id, r.emoji, r.user_id FROM reactions r " +
            "JOIN messages m ON m.id = r.message_id " +
            "WHERE m.channel_id = #{channelId} AND m.id BETWEEN #{minId} AND #{maxId} " +
            "ORDER BY r.id")
    List<ReactionRow> listByRange(@Param("channelId") long channelId,
                                  @Param("minId") long minId,
                                  @Param("maxId") long maxId);

    @Select("SELECT COUNT(*) FROM reactions WHERE message_id = #{messageId}")
    int countOnMessage(long messageId);

    @Select("SELECT COUNT(*) FROM reactions WHERE user_id = #{uid}")
    long countByUser(long uid);
}
