package yupeiyuan;


/** 角色视图 */
public record RoleVO(Long id, String name, String color, long permissions,
                     int position, boolean isDefault) {

    public static RoleVO from(Role r) {
        return new RoleVO(r.getId(), r.getName(), r.getColor(),
                r.getPermissions() == null ? 0 : r.getPermissions(),
                r.getPosition() == null ? 0 : r.getPosition(),
                Boolean.TRUE.equals(r.getIsDefault()));
    }
}
