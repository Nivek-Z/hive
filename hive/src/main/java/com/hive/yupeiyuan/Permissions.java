package com.hive.yupeiyuan;

/**
 * 权限位掩码定义（Discord 同款设计思路）。
 * 一个成员的生效权限 = 其所有角色 permissions 字段按位 OR；
 * 巢主(owner)拥有全部权限；ADMINISTRATOR 位等同于全部权限。
 */
public final class Permissions {

    public static final long ADMINISTRATOR    = 1L;        // 1<<0 管理员：等同所有权限
    public static final long MANAGE_HIVE      = 1L << 1;   // 修改蜂巢资料
    public static final long MANAGE_CHANNELS  = 1L << 2;   // 频道的增删改
    public static final long MANAGE_ROLES     = 1L << 3;   // 角色管理与分配
    public static final long KICK_MEMBERS     = 1L << 4;   // 踢出成员
    public static final long MUTE_MEMBERS     = 1L << 5;   // 禁言成员
    public static final long DELETE_MESSAGES  = 1L << 6;   // 删除他人消息
    public static final long CREATE_INVITE    = 1L << 7;   // 创建邀请码
    public static final long MENTION_EVERYONE = 1L << 8;   // @全体成员
    public static final long SEND_MESSAGES    = 1L << 9;   // 发送消息
    public static final long ATTACH_FILES     = 1L << 10;  // 发送图片
    public static final long ADD_REACTIONS    = 1L << 11;  // 添加表情回应

    /** 普通成员默认权限（建巢时赋给默认角色"工蜂"） */
    public static final long DEFAULT_MEMBER =
            CREATE_INVITE | SEND_MESSAGES | ATTACH_FILES | ADD_REACTIONS;

    /** 管理员预设权限（建巢时自动创建的"管理员"角色） */
    public static final long PRESET_ADMIN =
            DEFAULT_MEMBER | MANAGE_CHANNELS | KICK_MEMBERS | MUTE_MEMBERS
                    | DELETE_MESSAGES | MENTION_EVERYONE;

    /** 全部权限（巢主） */
    public static final long ALL = (1L << 12) - 1;

    private Permissions() {
    }

    /** 判断权限集 perms 是否包含 bit 权限（ADMINISTRATOR 覆盖一切） */
    public static boolean has(long perms, long bit) {
        return (perms & ADMINISTRATOR) != 0 || (perms & bit) == bit;
    }
}
