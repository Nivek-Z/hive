package com.hive.yupeiyuan;

import com.hive.zhangzhishuo.BizException;
import com.hive.zhangzhishuo.HiveMapper;
import com.hive.zhangzhishuo.HiveMemberMapper;
import com.hive.zhangzhishuo.Hive;
import com.hive.zhangzhishuo.HiveMember;
import org.springframework.stereotype.Service;

/**
 * 权限守卫：所有"我能不能做这件事"的判定入口。
 * 生效权限 = 巢主全权 | (默认角色 ∪ 已分配角色) 位掩码按位 OR（单条 BIT_OR SQL）。
 */
@Service
public class PermissionServiceImpl implements PermissionService {

    private final HiveMapper hiveMapper;
    private final HiveMemberMapper memberMapper;
    private final RoleMapper roleMapper;

    public PermissionServiceImpl(HiveMapper hiveMapper, HiveMemberMapper memberMapper, RoleMapper roleMapper) {
        this.hiveMapper = hiveMapper;
        this.memberMapper = memberMapper;
        this.roleMapper = roleMapper;
    }

    public Hive requireHive(long hiveId) {
        Hive hive = hiveMapper.findById(hiveId);
        if (hive == null) {
            throw BizException.notFound("蜂巢");
        }
        return hive;
    }

    public HiveMember requireMember(long hiveId, long userId) {
        HiveMember member = memberMapper.find(hiveId, userId);
        if (member == null) {
            throw BizException.forbidden("你不在该蜂巢中");
        }
        return member;
    }

    /** 计算成员在蜂巢内的生效权限位 */
    public long effective(Hive hive, long userId) {
        if (hive.getOwnerId() == userId) {
            return Permissions.ALL;
        }
        requireMember(hive.getId(), userId);
        return roleMapper.effectivePermissions(hive.getId(), userId);
    }

    /** 校验权限位，不满足直接抛 403 业务异常；返回蜂巢实体方便调用方复用 */
    public Hive require(long hiveId, long userId, long permissionBit) {
        Hive hive = requireHive(hiveId);
        if (!Permissions.has(effective(hive, userId), permissionBit)) {
            throw BizException.forbidden("没有权限执行此操作");
        }
        return hive;
    }

    public Hive requireOwner(long hiveId, long userId) {
        Hive hive = requireHive(hiveId);
        if (hive.getOwnerId() != userId) {
            throw BizException.forbidden("只有巢主可以执行此操作");
        }
        return hive;
    }
}
