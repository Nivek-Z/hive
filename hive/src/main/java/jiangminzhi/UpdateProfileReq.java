package jiangminzhi;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;

/** 修改个人资料请求 */
public record UpdateProfileReq(
        @NotBlank(message = "昵称不能为空")
        @Size(max = 16, message = "昵称最长 16 个字符")
        String nickname,

        @Size(max = 100, message = "签名最长 100 个字符")
        String bio,

        @Pattern(regexp = "^#[0-9a-fA-F]{6}$", message = "颜色格式不正确")
        String avatarColor) {
}
