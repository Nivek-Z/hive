package yupeiyuan;

import org.apache.ibatis.annotations.Delete;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Options;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

import java.util.List;

@Mapper
public interface FriendshipMapper {

    @Insert("INSERT INTO friendships(requester_id, addressee_id) VALUES(#{requesterId}, #{addresseeId})")
    @Options(useGeneratedKeys = true, keyProperty = "id", keyColumn = "id")
    int insert(Friendship friendship);

    @Select("SELECT * FROM friendships WHERE id = #{id}")
    Friendship findById(@Param("id") long id);

    /** 查找两人之间的关系记录（不区分谁发起） */
    @Select("SELECT * FROM friendships " +
            "WHERE (requester_id = #{a} AND addressee_id = #{b}) " +
            "   OR (requester_id = #{b} AND addressee_id = #{a}) LIMIT 1")
    Friendship findPair(@Param("a") long a, @Param("b") long b);

    @Update("UPDATE friendships SET status = 'ACCEPTED' WHERE id = #{id}")
    int accept(@Param("id") long id);

    @Delete("DELETE FROM friendships WHERE id = #{id}")
    int delete(@Param("id") long id);

    @Delete("DELETE FROM friendships " +
            "WHERE (requester_id = #{a} AND addressee_id = #{b}) " +
            "   OR (requester_id = #{b} AND addressee_id = #{a})")
    int deletePair(@Param("a") long a, @Param("b") long b);

    /** 我的好友列表（双向关系归一化为对方的用户信息） */
    @Select("SELECT u.id AS user_id, u.username, u.nickname, u.avatar_color, u.avatar_url, u.bio " +
            "FROM friendships f " +
            "JOIN users u ON u.id = IF(f.requester_id = #{uid}, f.addressee_id, f.requester_id) " +
            "WHERE f.status = 'ACCEPTED' AND (f.requester_id = #{uid} OR f.addressee_id = #{uid}) " +
            "ORDER BY u.nickname")
    List<FriendVO> listFriends(@Param("uid") long uid);

    /** 收到的待处理好友申请 */
    @Select("SELECT f.id, u.id AS user_id, u.username, u.nickname, " +
            "       u.avatar_color, u.avatar_url, f.created_at " +
            "FROM friendships f JOIN users u ON u.id = f.requester_id " +
            "WHERE f.addressee_id = #{uid} AND f.status = 'PENDING' ORDER BY f.id DESC")
    List<FriendRequestVO> listIncoming(@Param("uid") long uid);

    @Select("SELECT COUNT(*) FROM friendships " +
            "WHERE status = 'ACCEPTED' AND (requester_id = #{uid} OR addressee_id = #{uid})")
    int countFriends(@Param("uid") long uid);
}
