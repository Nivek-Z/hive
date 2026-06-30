package zhangkaiwen;

import jiangminzhi.UserMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.context.ApplicationEventPublisher;
import yupeiyuan.PermissionService;
import zhangzhishuo.HiveMember;

import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class MessageServiceImplTest {

    private MessageMapper messageMapper;
    private ReactionMapper reactionMapper;
    private PermissionService permissionService;
    private MessageServiceImpl service;

    @BeforeEach
    void setUp() {
        messageMapper = mock(MessageMapper.class);
        reactionMapper = mock(ReactionMapper.class);
        ReadStateMapper readStateMapper = mock(ReadStateMapper.class);
        ChannelMapper channelMapper = mock(ChannelMapper.class);
        ChannelMemberMapper channelMemberMapper = mock(ChannelMemberMapper.class);
        UserMapper userMapper = mock(UserMapper.class);
        permissionService = mock(PermissionService.class);
        CommandService commandService = mock(CommandService.class);
        ApplicationEventPublisher events = mock(ApplicationEventPublisher.class);
        WsPush push = mock(WsPush.class);
        service = new MessageServiceImpl(messageMapper, reactionMapper, readStateMapper, channelMapper,
                channelMemberMapper, userMapper, permissionService, commandService, events, push);

        Channel channel = new Channel();
        channel.setId(5L);
        channel.setHiveId(10L);
        channel.setType(Channel.TYPE_TEXT);
        when(channelMapper.findById(5L)).thenReturn(channel);
    }

    @Test
    void historyClampsLimitReversesToChronologicalOrderAndFillsReactions() {
        when(permissionService.requireMember(10L, 42L)).thenReturn(new HiveMember());
        MessageVO newest = message(3L);
        MessageVO oldest = message(2L);
        when(messageMapper.history(5L, Long.MAX_VALUE, 100)).thenReturn(List.of(newest, oldest));
        when(reactionMapper.listByRange(5L, 2L, 3L)).thenReturn(List.of(
                reaction(2L, "thumbs-up", 7L),
                reaction(2L, "thumbs-up", 8L),
                reaction(3L, "rocket", 9L)));

        List<MessageVO> result = service.history(42L, 5L, null, 500);

        assertEquals(List.of(2L, 3L), result.stream().map(MessageVO::getId).toList());
        assertEquals("thumbs-up", result.get(0).getReactions().get(0).emoji());
        assertEquals(2, result.get(0).getReactions().get(0).count());
        assertEquals(List.of(7L, 8L), result.get(0).getReactions().get(0).userIds());
        assertEquals("rocket", result.get(1).getReactions().get(0).emoji());
        verify(messageMapper).history(5L, Long.MAX_VALUE, 100);
    }

    private static MessageVO message(long id) {
        MessageVO vo = new MessageVO();
        vo.setId(id);
        vo.setChannelId(5L);
        return vo;
    }

    private static ReactionRow reaction(long messageId, String emoji, long userId) {
        ReactionRow row = new ReactionRow();
        row.setMessageId(messageId);
        row.setEmoji(emoji);
        row.setUserId(userId);
        return row;
    }
}
