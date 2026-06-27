package zhangzhishuo;

import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;

public interface FileService {

    FileVO upload(long uid, MultipartFile file) throws IOException;
}
