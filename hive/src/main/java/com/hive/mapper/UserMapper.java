package com.hive.mapper;

import com.hive.model.User;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Options;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

@Mapper
public interface UserMapper {

    @Insert("INSERT INTO users(username, password_hash, nickname, avatar_color) " +
            "VALUES(#{username}, #{passwordHash}, #{nickname}, #{avatarColor})")
    @Options(useGeneratedKeys = true, keyProperty = "id")
    int insert(User user);

    @Select("SELECT * FROM users WHERE username = #{username}")
    User findByUsername(String username);

    @Select("SELECT * FROM users WHERE id = #{id}")
    User findById(long id);

    @Select("SELECT COUNT(*) FROM users")
    long count();

    @Update("UPDATE users SET nickname = #{nickname}, bio = #{bio}, avatar_color = #{avatarColor} " +
            "WHERE id = #{id}")
    int updateProfile(@Param("id") long id,
                      @Param("nickname") String nickname,
                      @Param("bio") String bio,
                      @Param("avatarColor") String avatarColor);

    @Update("UPDATE users SET avatar_url = #{avatarUrl} WHERE id = #{id}")
    int updateAvatar(@Param("id") long id, @Param("avatarUrl") String avatarUrl);

    @Update("UPDATE users SET password_hash = #{hash} WHERE id = #{id}")
    int updatePassword(@Param("id") long id, @Param("hash") String hash);

    @Update("UPDATE users SET last_seen_at = NOW() WHERE id = #{id}")
    int touchLastSeen(long id);
}
