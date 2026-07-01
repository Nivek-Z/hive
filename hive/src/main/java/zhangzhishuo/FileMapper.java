package zhangzhishuo;

import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Options;

@Mapper
public interface FileMapper {

    @Insert("INSERT INTO files(uploader_id, stored_name, original_name, mime, size_bytes) " +
            "VALUES(#{uploaderId}, #{storedName}, #{originalName}, #{mime}, #{sizeBytes})")
    @Options(useGeneratedKeys = true, keyProperty = "id", keyColumn = "id")
    int insert(StoredFile file);
}
