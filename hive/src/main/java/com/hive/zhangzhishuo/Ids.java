package com.hive.zhangzhishuo;

import java.security.SecureRandom;

/**
 * 随机标识生成工具。
 */
public final class Ids {

    /** 邀请码字符集：去掉易混淆的 0/O/1/I/L */
    private static final char[] INVITE_ALPHABET =
            "ABCDEFGHJKMNPQRSTUVWXYZ23456789".toCharArray();
    public static final int INVITE_CODE_LENGTH = 8;

    private static final SecureRandom RANDOM = new SecureRandom();

    private Ids() {
    }

    /** 生成 8 位邀请码，如 K7XW2MQ9 */
    public static String inviteCode() {
        StringBuilder sb = new StringBuilder(INVITE_CODE_LENGTH);
        for (int i = 0; i < INVITE_CODE_LENGTH; i++) {
            sb.append(INVITE_ALPHABET[RANDOM.nextInt(INVITE_ALPHABET.length)]);
        }
        return sb.toString();
    }
}
