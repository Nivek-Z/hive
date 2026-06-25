package com.hive.yupeiyuan;

import org.apache.ibatis.annotations.Delete;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Options;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

import java.util.List;

@Mapper
public interface RoleMapper {

    @Insert("INSERT INTO roles(hive_id, name, color, permissions, position, is_default) " +
            "VALUES(#{hiveId}, #{name}, #{color}, #{permissions}, #{position}, #{isDefault})")
    @Options(useGeneratedKeys = true, keyProperty = "id")
    int insert(Role role);

    @Select("SELECT * FROM roles WHERE id = #{id}")
    Role findById(long id);

    @Select("SELECT * FROM roles WHERE hive_id = #{hiveId} ORDER BY position DESC, id")
    List<Role> listByHive(long hiveId);

    /**
     * 一条 SQL 计算成员生效权限：默认角色 ∪ 已分配角色 的 permissions 按位 OR。
     * BIT_OR 为 MySQL 聚合函数；无任何角色时返回 0。
     */
    @Select("SELECT COALESCE(BIT_OR(permissions), 0) FROM roles " +
            "WHERE hive_id = #{hiveId} " +
            "AND (is_default = 1 OR id IN " +
            "     (SELECT role_id FROM member_roles WHERE hive_id = #{hiveId} AND user_id = #{userId}))")
    long effectivePermissions(@Param("hiveId") long hiveId, @Param("userId") long userId);

    @Update("UPDATE roles SET name = #{name}, color = #{color}, permissions = #{permissions} " +
            "WHERE id = #{id}")
    int update(@Param("id") long id,
               @Param("name") String name,
               @Param("color") String color,
               @Param("permissions") long permissions);

    @Delete("DELETE FROM roles WHERE id = #{id}")
    int delete(long id);
}
