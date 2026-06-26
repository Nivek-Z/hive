package com.hive.yupeiyuan;

import org.springframework.stereotype.Service;

@Service
public class PermissionServiceImpl implements PermissionService {

    @Override
    public long effectivePermissions(long hiveId, long userId) {
        return Permissions.SEND_MESSAGES;
    }

    @Override
    public boolean hasPermission(long hiveId, long userId, long permissionBit) {
        return (effectivePermissions(hiveId, userId) & permissionBit) != 0;
    }

    @Override
    public void requirePermission(long hiveId, long userId, long permissionBit) {
        if (!hasPermission(hiveId, userId, permissionBit)) {
            throw new IllegalStateException("permission denied");
        }
    }
}
