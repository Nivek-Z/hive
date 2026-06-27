package zhangzhishuo;

import org.apache.ibatis.annotations.Delete;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Options;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

import java.util.List;

@Mapper
public interface HiveMapper {

    @Insert("INSERT INTO hives(name, description, icon_color, owner_id) " +
            "VALUES(#{name}, #{description}, #{iconColor}, #{ownerId})")
    @Options(useGeneratedKeys = true, keyProperty = "id")
    int insert(Hive hive);

    @Select("SELECT * FROM hives WHERE id = #{id}")
    Hive findById(long id);

    /** 我加入的所有蜂巢（按加入时间排序） */
    @Select("SELECT h.* FROM hives h " +
            "JOIN hive_members m ON m.hive_id = h.id " +
            "WHERE m.user_id = #{userId} ORDER BY m.joined_at")
    List<Hive> listByUserId(long userId);

    @Update("UPDATE hives SET name = #{name}, description = #{description}, icon_color = #{iconColor} " +
            "WHERE id = #{id}")
    int update(@Param("id") long id,
               @Param("name") String name,
               @Param("description") String description,
               @Param("iconColor") String iconColor);

    /** 删除蜂巢（频道/消息/成员/角色/邀请码全部级联删除） */
    @Delete("DELETE FROM hives WHERE id = #{id}")
    int delete(long id);
}
