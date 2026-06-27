package yupeiyuan;

import zhangzhishuo.BizException;
import zhangzhishuo.HiveMapper;
import zhangzhishuo.HiveMemberMapper;
import zhangzhishuo.Hive;
import zhangzhishuo.HiveMember;
import org.springframework.stereotype.Service;

public interface PermissionService {

    Hive requireHive(long hiveId);

    HiveMember requireMember(long hiveId, long userId);

    long effective(Hive hive, long userId);

    Hive require(long hiveId, long userId, long permissionBit);

    Hive requireOwner(long hiveId, long userId);

}
