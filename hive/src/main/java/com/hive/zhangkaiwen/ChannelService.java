package com.hive.zhangkaiwen;

import com.hive.zhangzhishuo.BizException;
import com.hive.yupeiyuan.PermissionService;
import com.hive.yupeiyuan.Permissions;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.Map;

/**
 * 频道树业务："群中群"= 分区(CATEGORY)可嵌套分区，文字频道为叶子。
 */
@Service
public class ChannelService {

    /** 频道树最大深度，防止无限套娃 */
    private static final int MAX_DEPTH = 5;

    private final ChannelMapper channelMapper;
    private final PermissionService permissionService;
    private final WsPush push;

    public ChannelService(ChannelMapper channelMapper, PermissionService permissionService, WsPush push) {
        this.channelMapper = channelMapper;
        this.permissionService = permissionService;
        this.push = push;
    }

    /** 通知全巢在线成员刷新频道树 */
    private void notifyChanged(long hiveId) {
        push.toHive(hiveId, "HIVE_EVENT", Map.of("kind", "CHANNELS_CHANGED", "hiveId", hiveId));
    }

    @Transactional
    public ChannelVO create(long uid, long hiveId, CreateChannelReq req) {
        permissionService.require(hiveId, uid, Permissions.MANAGE_CHANNELS);

        Long parentId = req.parentId();
        if (parentId != null) {
            Channel parent = requireChannel(parentId);
            if (parent.getHiveId() == null || parent.getHiveId() != hiveId) {
                throw new BizException("父频道不属于该蜂巢");
            }
            if (!Channel.TYPE_CATEGORY.equals(parent.getType())) {
                throw new BizException("只有分区下才能创建子频道");
            }
            if (depthOf(parent) + 1 >= MAX_DEPTH) {
                throw new BizException("频道层级最多嵌套 " + MAX_DEPTH + " 层");
            }
        }

        Channel c = new Channel();
        c.setHiveId(hiveId);
        c.setParentId(parentId);
        c.setType(req.type() == null ? Channel.TYPE_TEXT : req.type());
        c.setName(req.name());
        c.setTopic(req.topic() == null ? "" : req.topic());
        c.setPosition(0);
        channelMapper.insert(c);
        notifyChanged(hiveId);
        return ChannelVO.from(c);
    }

    @Transactional
    public ChannelVO update(long uid, long channelId, UpdateChannelReq req) {
        Channel channel = requireHiveChannel(channelId);
        permissionService.require(channel.getHiveId(), uid, Permissions.MANAGE_CHANNELS);
        String topic = req.topic() == null ? channel.getTopic() : req.topic();
        int position = req.position() == null ? channel.getPosition() : req.position();
        channelMapper.update(channelId, req.name(), topic, position);
        notifyChanged(channel.getHiveId());
        return ChannelVO.from(channelMapper.findById(channelId));
    }

    /** 删除频道：若是分区，子频道整体上移一层，不连坐删除 */
    @Transactional
    public void delete(long uid, long channelId) {
        Channel channel = requireHiveChannel(channelId);
        permissionService.require(channel.getHiveId(), uid, Permissions.MANAGE_CHANNELS);
        channelMapper.reparentChildren(channelId, channel.getParentId());
        channelMapper.delete(channelId);
        notifyChanged(channel.getHiveId());
    }

    public Channel requireChannel(long channelId) {
        Channel channel = channelMapper.findById(channelId);
        if (channel == null) {
            throw BizException.notFound("频道");
        }
        return channel;
    }

    /** 仅限蜂巢内频道（私聊频道不允许这些管理操作） */
    private Channel requireHiveChannel(long channelId) {
        Channel channel = requireChannel(channelId);
        if (channel.getHiveId() == null) {
            throw new BizException("私聊会话不支持此操作");
        }
        return channel;
    }

    private int depthOf(Channel channel) {
        int depth = 1;
        Channel current = channel;
        while (current.getParentId() != null && depth < MAX_DEPTH + 1) {
            current = channelMapper.findById(current.getParentId());
            if (current == null) {
                break;
            }
            depth++;
        }
        return depth;
    }
}
