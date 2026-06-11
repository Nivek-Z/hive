package com.hive.mapper;

import com.hive.model.Channel;
import org.apache.ibatis.annotations.Delete;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Options;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

import java.util.List;

@Mapper
public interface ChannelMapper {

    @Insert("INSERT INTO channels(hive_id, parent_id, type, name, topic, position) " +
            "VALUES(#{hiveId}, #{parentId}, #{type}, #{name}, #{topic}, #{position})")
    @Options(useGeneratedKeys = true, keyProperty = "id")
    int insert(Channel channel);

    @Select("SELECT * FROM channels WHERE id = #{id}")
    Channel findById(long id);

    @Select("SELECT * FROM channels WHERE hive_id = #{hiveId} " +
            "ORDER BY COALESCE(parent_id, 0), position, id")
    List<Channel> listByHive(long hiveId);

    @Update("UPDATE channels SET name = #{name}, topic = #{topic}, position = #{position} " +
            "WHERE id = #{id}")
    int update(@Param("id") long id,
               @Param("name") String name,
               @Param("topic") String topic,
               @Param("position") int position);

    /** 删除分区前：把子频道挂到上一级（避免级联误删整棵子树） */
    @Update("UPDATE channels SET parent_id = #{newParentId} WHERE parent_id = #{oldParentId}")
    int reparentChildren(@Param("oldParentId") long oldParentId,
                         @Param("newParentId") Long newParentId);

    @Delete("DELETE FROM channels WHERE id = #{id}")
    int delete(long id);
}
