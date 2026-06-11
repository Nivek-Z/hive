package com.hive.mapper;

import com.hive.model.HiveMember;
import com.hive.model.dto.MemberVO;
import org.apache.ibatis.annotations.Delete;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

import java.util.List;

@Mapper
public interface HiveMemberMapper {

    @Insert("INSERT INTO hive_members(hive_id, user_id) VALUES(#{hiveId}, #{userId})")
    int insert(@Param("hiveId") long hiveId, @Param("userId") long userId);

    @Select("SELECT * FROM hive_members WHERE hive_id = #{hiveId} AND user_id = #{userId}")
    HiveMember find(@Param("hiveId") long hiveId, @Param("userId") long userId);

    @Select("SELECT COUNT(*) FROM hive_members WHERE hive_id = #{hiveId}")
    int countByHive(long hiveId);

    /** 成员列表（连表带出用户资料 + 是否巢主标记） */
    @Select("SELECT m.user_id, u.username, u.nickname, m.hive_nickname, " +
            "       u.avatar_color, u.avatar_url, m.muted_until, m.joined_at, " +
            "       (m.user_id = h.owner_id) AS owner " +
            "FROM hive_members m " +
            "JOIN users u ON u.id = m.user_id " +
            "JOIN hives h ON h.id = m.hive_id " +
            "WHERE m.hive_id = #{hiveId} " +
            "ORDER BY owner DESC, m.joined_at")
    List<MemberVO> listByHive(long hiveId);

    /** 蜂巢全部成员 id（WebSocket 广播用） */
    @Select("SELECT user_id FROM hive_members WHERE hive_id = #{hiveId}")
    List<Long> listUserIds(long hiveId);

    @Update("UPDATE hive_members SET muted_until = #{until} " +
            "WHERE hive_id = #{hiveId} AND user_id = #{userId}")
    int updateMute(@Param("hiveId") long hiveId,
                   @Param("userId") long userId,
                   @Param("until") java.time.LocalDateTime until);

    @Delete("DELETE FROM hive_members WHERE hive_id = #{hiveId} AND user_id = #{userId}")
    int delete(@Param("hiveId") long hiveId, @Param("userId") long userId);
}
