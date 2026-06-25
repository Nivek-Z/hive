package com.hive.yupeiyuan;

/**
 * 虞沛远负责：权限位规划。
 */
public final class Permissions {

    public static final long MANAGE_HIVE = 1L << 0;
    public static final long MANAGE_CHANNELS = 1L << 1;
    public static final long MANAGE_ROLES = 1L << 2;
    public static final long CREATE_INVITE = 1L << 3;
    public static final long SEND_MESSAGES = 1L << 4;
    public static final long DELETE_MESSAGES = 1L << 5;
    public static final long MUTE_MEMBERS = 1L << 6;
    public static final long KICK_MEMBERS = 1L << 7;

    private Permissions() {
    }
}
