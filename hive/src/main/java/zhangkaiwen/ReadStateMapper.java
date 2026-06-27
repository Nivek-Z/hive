package zhangkaiwen;

import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;

@Mapper
public interface ReadStateMapper {

    /** UPSERT：GREATEST 防止已读指针回退（多端并发安全） */
    @Insert("INSERT INTO read_states(user_id, channel_id, last_read_message_id) " +
            "VALUES(#{userId}, #{channelId}, #{lastReadMessageId}) " +
            "ON DUPLICATE KEY UPDATE last_read_message_id = GREATEST(last_read_message_id, #{lastReadMessageId})")
    int upsert(@Param("userId") long userId,
               @Param("channelId") long channelId,
               @Param("lastReadMessageId") long lastReadMessageId);
}
