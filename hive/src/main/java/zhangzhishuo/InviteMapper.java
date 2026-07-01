package zhangzhishuo;

import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Options;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

import java.util.List;

@Mapper
public interface InviteMapper {

    @Insert("INSERT INTO invites(code, hive_id, creator_id, max_uses, expires_at) " +
            "VALUES(#{code}, #{hiveId}, #{creatorId}, #{maxUses}, #{expiresAt})")
    @Options(useGeneratedKeys = true, keyProperty = "id", keyColumn = "id")
    int insert(Invite invite);

    @Select("SELECT * FROM invites WHERE code = #{code}")
    Invite findByCode(@Param("code") String code);

    @Select("SELECT * FROM invites WHERE hive_id = #{hiveId} ORDER BY id DESC")
    List<Invite> listByHive(@Param("hiveId") long hiveId);

    /**
     * 原子核销：次数与有效期校验放在 SQL 条件里，
     * 并发场景下也不会超发（影响行数=0 即失败）。
     */
    @Update("UPDATE invites SET used_count = used_count + 1 " +
            "WHERE id = #{id} " +
            "AND (max_uses = 0 OR used_count < max_uses) " +
            "AND (expires_at IS NULL OR expires_at > NOW())")
    int consume(@Param("id") long id);
}
