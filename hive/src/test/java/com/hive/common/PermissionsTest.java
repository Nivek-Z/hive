package com.hive.common;

import org.junit.jupiter.api.Test;

import static com.hive.common.Permissions.ADD_REACTIONS;
import static com.hive.common.Permissions.ADMINISTRATOR;
import static com.hive.common.Permissions.ALL;
import static com.hive.common.Permissions.ATTACH_FILES;
import static com.hive.common.Permissions.CREATE_INVITE;
import static com.hive.common.Permissions.DEFAULT_MEMBER;
import static com.hive.common.Permissions.DELETE_MESSAGES;
import static com.hive.common.Permissions.KICK_MEMBERS;
import static com.hive.common.Permissions.MANAGE_ROLES;
import static com.hive.common.Permissions.SEND_MESSAGES;
import static com.hive.common.Permissions.has;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class PermissionsTest {

    @Test
    void defaultMemberCanChatButNotManage() {
        assertTrue(has(DEFAULT_MEMBER, SEND_MESSAGES));
        assertTrue(has(DEFAULT_MEMBER, ATTACH_FILES));
        assertTrue(has(DEFAULT_MEMBER, ADD_REACTIONS));
        assertTrue(has(DEFAULT_MEMBER, CREATE_INVITE));
        assertFalse(has(DEFAULT_MEMBER, KICK_MEMBERS));
        assertFalse(has(DEFAULT_MEMBER, DELETE_MESSAGES));
        assertFalse(has(DEFAULT_MEMBER, MANAGE_ROLES));
    }

    @Test
    void administratorBitOverridesEverything() {
        assertTrue(has(ADMINISTRATOR, KICK_MEMBERS));
        assertTrue(has(ADMINISTRATOR, DELETE_MESSAGES));
        assertTrue(has(ADMINISTRATOR, SEND_MESSAGES));
    }

    @Test
    void multipleRolesCombineWithBitwiseOr() {
        long roleA = SEND_MESSAGES;
        long roleB = KICK_MEMBERS;
        long effective = roleA | roleB;
        assertTrue(has(effective, SEND_MESSAGES));
        assertTrue(has(effective, KICK_MEMBERS));
        assertFalse(has(effective, MANAGE_ROLES));
    }

    @Test
    void allContainsEveryDefinedBit() {
        assertTrue(has(ALL, ADMINISTRATOR));
        assertTrue(has(ALL, MANAGE_ROLES));
        assertTrue(has(ALL, ADD_REACTIONS));
    }
}
