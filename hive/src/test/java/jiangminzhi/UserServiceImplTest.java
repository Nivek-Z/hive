package jiangminzhi;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.security.crypto.password.PasswordEncoder;
import zhangzhishuo.BizException;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class UserServiceImplTest {

    private UserMapper userMapper;
    private PasswordEncoder passwordEncoder;
    private UserServiceImpl service;

    @BeforeEach
    void setUp() {
        userMapper = mock(UserMapper.class);
        passwordEncoder = mock(PasswordEncoder.class);
        service = new UserServiceImpl(userMapper, passwordEncoder);
    }

    @Test
    void updateProfileKeepsExistingOptionalFieldsWhenRequestOmitsThem() {
        User existing = user(7L, "alice", "Alice", "#123456", "old bio");
        User updated = user(7L, "alice", "Neo", "#123456", "old bio");
        when(userMapper.findById(7L)).thenReturn(existing, updated);

        UserVO result = service.updateProfile(7L, new UpdateProfileReq("Neo", null, null));

        assertEquals("Neo", result.nickname());
        assertEquals("old bio", result.bio());
        assertEquals("#123456", result.avatarColor());
        verify(userMapper).updateProfile(7L, "Neo", "old bio", "#123456");
    }

    @Test
    void changePasswordRejectsWrongOldPasswordWithoutUpdatingHash() {
        User existing = user(7L, "alice", "Alice", "#123456", "old bio");
        existing.setPasswordHash("encoded-old");
        when(userMapper.findById(7L)).thenReturn(existing);
        when(passwordEncoder.matches("wrong", "encoded-old")).thenReturn(false);

        assertThrows(BizException.class,
                () -> service.changePassword(7L, new ChangePasswordReq("wrong", "new-secret")));

        verify(userMapper, never()).updatePassword(7L, "encoded-new");
    }

    @Test
    void requireThrowsNotFoundWhenUserDoesNotExist() {
        when(userMapper.findById(404L)).thenReturn(null);

        BizException error = assertThrows(BizException.class, () -> service.require(404L));

        assertEquals(BizException.CODE_NOT_FOUND, error.getCode());
    }

    private static User user(Long id, String username, String nickname, String avatarColor, String bio) {
        User user = new User();
        user.setId(id);
        user.setUsername(username);
        user.setNickname(nickname);
        user.setAvatarColor(avatarColor);
        user.setBio(bio);
        return user;
    }
}
