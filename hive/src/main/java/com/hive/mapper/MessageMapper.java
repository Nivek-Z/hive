package com.hive.mapper;

import com.hive.model.Message;
import com.hive.model.dto.MessageVO;
import com.hive.model.dto.UnreadRow;
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
    @Options(useGeneratedKeys = true, keyProperty = "id")
    int insert(Message message);

    @Select("SELECT * FROM messages WHERE id = #{id}")
    Message findById(long id);

    @Select(BASE_SELECT + "WHERE m.id = #{id}")
    MessageVO findVOById(long id);

    /** 游标分页：取 before 之前最新 limit 条（DESC，service 层反转为正序） */
    @Select(BASE_SELECT +
            "WHERE m.channel_id = #{channelId} AND m.deleted = 0 AND m.id < #{before} " +
            "ORDER BY m.id DESC LIMIT #{limit}")
    List<MessageVO> history(@Param("channelId") long channelId,
                            @Param("before") long before,
                            @Param("limit") int limit);

    @Update("UPDATE messages SET deleted = 1 WHERE id = #{id}")
    int softDelete(long id);

    /** 一条 SQL 统计某用户在某蜂巢内每个频道的未读数 */
    @Select("SELECT m.channel_id AS channelId, COUNT(*) AS count " +
            "FROM messages m " +
            "JOIN channels c ON c.id = m.channel_id AND c.hive_id = #{hiveId} " +
            "LEFT JOIN read_states rs ON rs.channel_id = m.channel_id AND rs.user_id = #{userId} " +
            "WHERE m.deleted = 0 AND m.id > COALESCE(rs.last_read_message_id, 0) " +
            "GROUP BY m.channel_id")
    List<UnreadRow> unreadCounts(@Param("hiveId") long hiveId, @Param("userId") long userId);
}
