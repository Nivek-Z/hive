package yupeiyuan;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.junit.jupiter.MockitoExtension;
import zhangkaiwen.WsPush;

import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class RoleServiceImplTest {

    private RoleMapper roleMapper;
    private MemberRoleMapper memberRoleMapper;
    private PermissionService permissionService;
    private WsPush push;
    private RoleServiceImpl service;

    @BeforeEach
    void setUp() {
        roleMapper = mock(RoleMapper.class);
        memberRoleMapper = mock(MemberRoleMapper.class);
        permissionService = mock(PermissionService.class);
        push = mock(WsPush.class);
        service = new RoleServiceImpl(roleMapper, memberRoleMapper, permissionService, push);
    }

    @Test
    void createSanitizesUnknownPermissionBitsAndAssignsNextPosition() {
        Role existing = role(1L, "existing", "#111111", Permissions.SEND_MESSAGES, 5, false);
        when(roleMapper.listByHive(10L)).thenReturn(List.of(existing));

        RoleVO result = service.create(42L, 10L,
                new RoleReq("mods", null, Permissions.ALL | (1L << 62)));

        ArgumentCaptor<Role> inserted = ArgumentCaptor.forClass(Role.class);
        verify(permissionService).require(10L, 42L, Permissions.MANAGE_ROLES);
        verify(roleMapper).insert(inserted.capture());
        assertEquals("mods", inserted.getValue().getName());
        assertEquals("#99AAB5", inserted.getValue().getColor());
        assertEquals(Permissions.ALL, inserted.getValue().getPermissions());
        assertEquals(6, inserted.getValue().getPosition());
        assertFalse(inserted.getValue().getIsDefault());
        assertEquals(Permissions.ALL, result.permissions());
        verify(push).toHive(10L, "HIVE_EVENT", java.util.Map.of("kind", "ROLES_CHANGED", "hiveId", 10L));
    }

    @Test
    void assignKeepsOnlyDistinctNonDefaultRolesFromSameHive() {
        Role member = role(11L, "member", "#111111", Permissions.SEND_MESSAGES, 1, false);
        Role defaultRole = role(12L, "default", "#222222", Permissions.DEFAULT_MEMBER, 0, true);
        when(roleMapper.listByHive(10L)).thenReturn(List.of(member, defaultRole));

        service.assign(42L, 10L, 77L, List.of(11L, 11L, 12L, 99L));

        verify(permissionService).require(10L, 42L, Permissions.MANAGE_ROLES);
        verify(permissionService).requireMember(10L, 77L);
        verify(memberRoleMapper).deleteByMember(10L, 77L);
        verify(memberRoleMapper).insert(10L, 77L, 11L);
        verify(push).toHive(10L, "HIVE_EVENT", java.util.Map.of("kind", "ROLES_CHANGED", "hiveId", 10L));
    }

    private static Role role(Long id, String name, String color, long permissions, int position, boolean isDefault) {
        Role role = new Role();
        role.setId(id);
        role.setHiveId(10L);
        role.setName(name);
        role.setColor(color);
        role.setPermissions(permissions);
        role.setPosition(position);
        role.setIsDefault(isDefault);
        return role;
    }
}
