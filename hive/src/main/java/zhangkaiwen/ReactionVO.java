package zhangkaiwen;

import java.util.List;

/** 单条消息上某个表情的聚合：谁回应了、共几个 */
public record ReactionVO(String emoji, int count, List<Long> userIds) {
}
