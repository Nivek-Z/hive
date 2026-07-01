package zhangkaiwen;

import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Options;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

import java.util.List;

@Mapper
public interface MessageMapper {

    /** 公共连表：发送者快照 + 被回复消息摘要（被回复消息已删则摘要为 NULL） */
    String BASE_SELECT =
            "SELECT m.id, m.channel_id, m.sender_id, m.type, m.content, m.reply_to_id, m.created_at, " +
            "       u.nickname AS sender_nickname, u.avatar_color AS sender_avatar_color, " +
            "       u.avatar_url AS sender_avatar_url, " +
            "       ru.nickname AS reply_sender_nickname, " +
            "       CASE WHEN rm.deleted = 1 THEN NULL ELSE rm.content END AS reply_content " +
            "FROM messages m " +
            "LEFT JOIN users u ON u.id = m.sender_id " +
            "LEFT JOIN messages rm ON rm.id = m.reply_to_id " +
            "LEFT JOIN users ru ON ru.id = rm.sender_id ";

    @Insert("INSERT INTO messages(channel_id, sender_id, type, content, reply_to_id) " +
            "VALUES(#{channelId}, #{senderId}, #{type}, #{content}, #{replyToId})")
    @Options(useGeneratedKeys = true, keyProperty = "id", keyColumn = "id")
    int insert(Message message);

    @Select("SELECT * FROM messages WHERE id = #{id}")
    Message findById(@Param("id") long id);

    @Select(BASE_SELECT + "WHERE m.id = #{id}")
    MessageVO findVOById(@Param("id") long id);

    /** 游标分页：取 before 之前最新 limit 条（DESC，service 层反转为正序） */
    @Select(BASE_SELECT +
            "WHERE m.channel_id = #{channelId} AND m.deleted = 0 AND m.id < #{before} " +
            "ORDER BY m.id DESC LIMIT #{limit}")
    List<MessageVO> history(@Param("channelId") long channelId,
                            @Param("before") long before,
                            @Param("limit") int limit);

    @Update("UPDATE messages SET deleted = 1 WHERE id = #{id}")
    int softDelete(@Param("id") long id);

    /** 一条 SQL 统计某用户在某蜂巢内每个频道的未读数 */
    @Select("SELECT m.channel_id AS channelId, COUNT(*) AS count " +
            "FROM messages m " +
            "JOIN channels c ON c.id = m.channel_id AND c.hive_id = #{hiveId} " +
            "LEFT JOIN read_states rs ON rs.channel_id = m.channel_id AND rs.user_id = #{userId} " +
            "WHERE m.deleted = 0 AND m.id > COALESCE(rs.last_read_message_id, 0) " +
            "GROUP BY m.channel_id")
    List<UnreadRow> unreadCounts(@Param("hiveId") long hiveId, @Param("userId") long userId);

    // ---------- 成就判定用计数 ----------

    @Select("SELECT COUNT(*) FROM messages WHERE sender_id = #{uid} AND deleted = 0")
    long countBySender(@Param("uid") long uid);

    @Select("SELECT COUNT(*) FROM messages WHERE sender_id = #{uid} AND deleted = 0 " +
            "AND created_at >= CURDATE()")
    long countBySenderToday(@Param("uid") long uid);

    /** 最近 7 个自然日中有发言的天数（=7 即连续一周打卡） */
    @Select("SELECT COUNT(DISTINCT DATE(created_at)) FROM messages " +
            "WHERE sender_id = #{uid} AND deleted = 0 " +
            "AND created_at >= CURDATE() - INTERVAL 6 DAY")
    int countActiveDaysLast7(@Param("uid") long uid);

    // ---------- 数据可视化 ----------

    /** 个人聊天热力图：过去一年逐日消息数 */
    @Select("SELECT DATE(created_at) AS date, COUNT(*) AS count FROM messages " +
            "WHERE sender_id = #{uid} AND deleted = 0 " +
            "AND created_at >= CURDATE() - INTERVAL 364 DAY " +
            "GROUP BY DATE(created_at) ORDER BY date")
    List<HeatRow> heatmap(@Param("uid") long uid);

    /** 蜂巢近 7 日逐日消息量 */
    @Select("SELECT DATE(m.created_at) AS date, COUNT(*) AS count FROM messages m " +
            "JOIN channels c ON c.id = m.channel_id AND c.hive_id = #{hiveId} " +
            "WHERE m.deleted = 0 AND m.created_at >= CURDATE() - INTERVAL 6 DAY " +
            "GROUP BY DATE(m.created_at) ORDER BY date")
    List<HeatRow> hiveDaily(@Param("hiveId") long hiveId);

    /** 蜂巢发言排行（前 5） */
    @Select("SELECT u.nickname AS name, COUNT(*) AS count FROM messages m " +
            "JOIN channels c ON c.id = m.channel_id AND c.hive_id = #{hiveId} " +
            "JOIN users u ON u.id = m.sender_id " +
            "WHERE m.deleted = 0 GROUP BY m.sender_id, u.nickname " +
            "ORDER BY count DESC LIMIT 5")
    List<NameCount> hiveTopSpeakers(@Param("hiveId") long hiveId);

    /** ngram 中文全文检索（NATURAL LANGUAGE MODE） */
    @Select("SELECT m.id, m.channel_id, c.name AS channel_name, u.nickname AS sender_nickname, " +
            "       m.content, m.created_at " +
            "FROM messages m " +
            "JOIN channels c ON c.id = m.channel_id AND c.hive_id = #{hiveId} " +
            "LEFT JOIN users u ON u.id = m.sender_id " +
            "WHERE m.deleted = 0 AND m.type = 'TEXT' " +
            "AND MATCH(m.content) AGAINST(#{q} IN NATURAL LANGUAGE MODE) " +
            "ORDER BY m.id DESC LIMIT 30")
    List<SearchHit> search(@Param("hiveId") long hiveId, @Param("q") String q);
}
