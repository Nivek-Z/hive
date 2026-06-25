package com.hive.yupeiyuan;

import org.apache.ibatis.annotations.Delete;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface MemberRoleMapper {

    @Insert("INSERT IGNORE INTO member_roles(hive_id, user_id, role_id) " +
            "VALUES(#{hiveId}, #{userId}, #{roleId})")
    int insert(@Param("hiveId") long hiveId,
               @Param("userId") long userId,
               @Param("roleId") long roleId);

    @Select("SELECT * FROM member_roles WHERE hive_id = #{hiveId}")
    List<MemberRole> listByHive(long hiveId);

    /** 成员退出/被踢时清理其在该蜂巢的所有角色 */
    @Delete("DELETE FROM member_roles WHERE hive_id = #{hiveId} AND user_id = #{userId}")
    int deleteByMember(@Param("hiveId") long hiveId, @Param("userId") long userId);

    @Delete("DELETE FROM member_roles WHERE hive_id = #{hiveId} AND user_id = #{userId} AND role_id = #{roleId}")
    int delete(@Param("hiveId") long hiveId,
               @Param("userId") long userId,
               @Param("roleId") long roleId);
}
