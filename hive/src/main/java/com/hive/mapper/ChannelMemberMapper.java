package com.hive.mapper;

import com.hive.model.Channel;
import com.hive.model.dto.DmVO;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

/** 私聊（DM）频道参与者表访问 */
@Mapper
public interface ChannelMemberMapper {

    @Insert("INSERT IGNORE INTO channel_members(channel_id, user_id) VALUES(#{channelId}, #{userId})")
    int insert(@Param("channelId") long channelId, @Param("userId") long userId);

    @Select("SELECT COUNT(*) FROM channel_members WHERE channel_id = #{channelId} AND user_id = #{userId}")
    int countMember(@Param("channelId") long channelId, @Param("userId") long userId);

    @Select("SELECT user_id FROM channel_members WHERE channel_id = #{channelId}")
    List<Long> listUserIds(long channelId);

    /** 两人之间已存在的 DM 频道 */
    @Select("SELECT c.* FROM channels c " +
            "JOIN channel_members m1 ON m1.channel_id = c.id AND m1.user_id = #{a} " +
            "JOIN channel_members m2 ON m2.channel_id = c.id AND m2.user_id = #{b} " +
            "WHERE c.type = 'DM' LIMIT 1")
    Channel findDmChannel(@Param("a") long a, @Param("b") long b);

    /**
     * 我的私聊会话列表：对方信息 + 最后一条消息 + 未读数，
     * 按最新消息排序（子查询展示，课程报告可重点讲解）。
     */
    @Select("SELECT c.id AS channel_id, " +
            "       u.id AS user_id, u.username, u.nickname, u.avatar_color, u.avatar_url, " +
            "       (SELECT m.content FROM messages m WHERE m.channel_id = c.id AND m.deleted = 0 " +
            "        ORDER BY m.id DESC LIMIT 1) AS last_content, " +
            "       (SELECT m.created_at FROM messages m WHERE m.channel_id = c.id AND m.deleted = 0 " +
            "        ORDER BY m.id DESC LIMIT 1) AS last_at, " +
            "       (SELECT COUNT(*) FROM messages m WHERE m.channel_id = c.id AND m.deleted = 0 " +
            "        AND m.sender_id IS NOT NULL AND m.sender_id <> #{uid} " +
            "        AND m.id > COALESCE((SELECT r.last_read_message_id FROM read_states r " +
            "                             WHERE r.user_id = #{uid} AND r.channel_id = c.id), 0)) AS unread " +
            "FROM channels c " +
            "JOIN channel_members me ON me.channel_id = c.id AND me.user_id = #{uid} " +
            "JOIN channel_members o  ON o.channel_id = c.id AND o.user_id <> #{uid} " +
            "JOIN users u ON u.id = o.user_id " +
            "WHERE c.type = 'DM' " +
            "ORDER BY COALESCE((SELECT MAX(m2.id) FROM messages m2 WHERE m2.channel_id = c.id), 0) DESC")
    List<DmVO> listDms(long uid);
}
