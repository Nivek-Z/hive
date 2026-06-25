package com.hive.zhangzhishuo;

import org.junit.jupiter.api.Test;

import java.util.HashSet;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class IdsTest {

    @Test
    void inviteCodeHasFixedLength() {
        assertEquals(Ids.INVITE_CODE_LENGTH, Ids.inviteCode().length());
    }

    @Test
    void inviteCodeAvoidsAmbiguousChars() {
        for (int i = 0; i < 200; i++) {
            String code = Ids.inviteCode();
            assertFalse(code.contains("0"), "不应包含数字 0");
            assertFalse(code.contains("O"), "不应包含字母 O");
            assertFalse(code.contains("1"), "不应包含数字 1");
            assertFalse(code.contains("I"), "不应包含字母 I");
            assertFalse(code.contains("L"), "不应包含字母 L");
            assertTrue(code.matches("[A-Z2-9]{8}"));
        }
    }

    @Test
    void inviteCodesAreEffectivelyUnique() {
        Set<String> codes = new HashSet<>();
        for (int i = 0; i < 1000; i++) {
            codes.add(Ids.inviteCode());
        }
        // 31^8 ≈ 8500 亿种组合，1000 次内出现重复几乎不可能
        assertEquals(1000, codes.size());
    }
}
