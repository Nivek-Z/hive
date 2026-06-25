package com.hive.yupeiyuan;

import com.hive.zhangzhishuo.BizException;
import com.hive.zhangkaiwen.WsPush;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

/**
 * 角色管理：CRUD 与成员角色分配。
 * 生效权限的计算在 PermissionService（BIT_OR 聚合），此处只管"角色本身"。
 */
@Service
public class RoleService {

    private static final int MAX_ROLES_PER_HIVE = 20;

    private final RoleMapper roleMapper;
    private final MemberRoleMapper memberRoleMapper;
    private final PermissionService permissionService;
    private final WsPush push;

    public RoleService(RoleMapper roleMapper, MemberRoleMapper memberRoleMapper,
                       PermissionService permissionService, WsPush push) {
        this.roleMapper = roleMapper;
        this.memberRoleMapper = memberRoleMapper;
        this.permissionService = permissionService;
        this.push = push;
    }

    public List<RoleVO> list(long uid, long hiveId) {
        permissionService.requireHive(hiveId);
        permissionService.requireMember(hiveId, uid);
        return roleMapper.listByHive(hiveId).stream().map(RoleVO::from).toList();
    }

    @Transactional
    public RoleVO create(long uid, long hiveId, RoleReq req) {
        permissionService.require(hiveId, uid, Permissions.MANAGE_ROLES);
        List<Role> existing = roleMapper.listByHive(hiveId);
        if (existing.size() >= MAX_ROLES_PER_HIVE) {
            throw new BizException("角色数量已达上限（" + MAX_ROLES_PER_HIVE + " 个）");
        }
        int maxPosition = existing.stream()
                .mapToInt(r -> r.getPosition() == null ? 0 : r.getPosition())
                .max().orElse(0);
        Role role = new Role();
        role.setHiveId(hiveId);
        role.setName(req.name());
        role.setColor(req.color() == null ? "#99AAB5" : req.color());
        role.setPermissions(sanitize(req.permissions()));
        role.setPosition(maxPosition + 1);
        role.setIsDefault(false);
        roleMapper.insert(role);
        notifyChanged(hiveId);
        return RoleVO.from(role);
    }

    @Transactional
    public RoleVO update(long uid, long roleId, RoleReq req) {
        Role role = requireRole(roleId);
        permissionService.require(role.getHiveId(), uid, Permissions.MANAGE_ROLES);
        String color = req.color() == null ? role.getColor() : req.color();
        roleMapper.update(roleId, req.name(), color, sanitize(req.permissions()));
        notifyChanged(role.getHiveId());
        return RoleVO.from(roleMapper.findById(roleId));
    }

    @Transactional
    public void delete(long uid, long roleId) {
        Role role = requireRole(roleId);
        permissionService.require(role.getHiveId(), uid, Permissions.MANAGE_ROLES);
        if (Boolean.TRUE.equals(role.getIsDefault())) {
            throw new BizException("默认角色不能删除");
        }
        roleMapper.delete(roleId); // member_roles 级联删除
        notifyChanged(role.getHiveId());
    }

    /** 重设某成员的角色集合（全量替换） */
    @Transactional
    public void assign(long uid, long hiveId, long targetId, List<Long> roleIds) {
        permissionService.require(hiveId, uid, Permissions.MANAGE_ROLES);
        permissionService.requireMember(hiveId, targetId);
        Set<Long> valid = roleMapper.listByHive(hiveId).stream()
                .filter(r -> !Boolean.TRUE.equals(r.getIsDefault()))
                .map(Role::getId)
                .collect(Collectors.toSet());
        memberRoleMapper.deleteByMember(hiveId, targetId);
        if (roleIds != null) {
            roleIds.stream().distinct().filter(valid::contains)
                    .forEach(roleId -> memberRoleMapper.insert(hiveId, targetId, roleId));
        }
        notifyChanged(hiveId);
    }

    // ---------- 内部 ----------

    /** 只保留已定义的权限位，防止写入未知位 */
    private long sanitize(Long permissions) {
        return (permissions == null ? 0 : permissions) & Permissions.ALL;
    }

    private Role requireRole(long roleId) {
        Role role = roleMapper.findById(roleId);
        if (role == null) {
            throw BizException.notFound("角色");
        }
        return role;
    }

    private void notifyChanged(long hiveId) {
        push.toHive(hiveId, "HIVE_EVENT", Map.of("kind", "ROLES_CHANGED", "hiveId", hiveId));
    }
}
