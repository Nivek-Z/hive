package com.hive.yupeiyuan;

import java.util.List;
import java.util.Map;

/**
 * 虞沛远负责：好友申请、好友列表和私聊频道。
 */
public interface FriendService {

    void sendFriendRequest(long requesterId, String targetUsername);

    void acceptFriendRequest(long userId, long requestId);

    List<Map<String, Object>> listFriends(long userId);

    Map<String, Object> openDm(long userId, long friendId);
}
