package com.hive.yupeiyuan;

/**
 * 虞沛远负责：蜂巢内权限判断。
 */
public interface PermissionService {

    long effectivePermissions(long hiveId, long userId);

    boolean hasPermission(long hiveId, long userId, long permissionBit);

    void requirePermission(long hiveId, long userId, long permissionBit);
}
