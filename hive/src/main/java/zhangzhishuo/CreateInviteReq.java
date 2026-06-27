package zhangzhishuo;

import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Min;

/** 创建邀请码请求：maxUses=0 不限次数；expiresHours=0 永不过期 */
public record CreateInviteReq(
        @Min(value = 0, message = "使用次数不能为负")
        @Max(value = 1000, message = "使用次数最多 1000")
        Integer maxUses,

        @Min(value = 0, message = "有效期不能为负")
        @Max(value = 720, message = "有效期最长 720 小时")
        Integer expiresHours) {
}
