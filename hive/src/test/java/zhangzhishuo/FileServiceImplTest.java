package zhangzhishuo;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.mock.web.MockMultipartFile;

import java.nio.file.Files;
import java.nio.file.Path;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;

@ExtendWith(MockitoExtension.class)
class FileServiceImplTest {

    @TempDir
    private Path uploadDir;

    private FileMapper fileMapper;
    private FileServiceImpl service;

    @BeforeEach
    void setUp() {
        fileMapper = mock(FileMapper.class);
        service = new FileServiceImpl(fileMapper, uploadDir.toString());
    }

    @Test
    void uploadStoresAllowedImageAndRecordsMetadata() throws Exception {
        MockMultipartFile file = new MockMultipartFile(
                "file", "avatar.jpg", "image/jpeg", new byte[]{1, 2, 3});

        FileVO result = service.upload(9L, file);

        assertEquals("avatar.jpg", result.originalName());
        assertEquals(3L, result.size());
        assertTrue(result.url().startsWith("/uploads/"));
        assertTrue(result.url().endsWith(".jpg"));

        ArgumentCaptor<StoredFile> stored = ArgumentCaptor.forClass(StoredFile.class);
        verify(fileMapper).insert(stored.capture());
        assertEquals(9L, stored.getValue().getUploaderId());
        assertEquals("avatar.jpg", stored.getValue().getOriginalName());
        assertEquals("image/jpeg", stored.getValue().getMime());
        assertEquals(3L, stored.getValue().getSizeBytes());
        assertTrue(Files.exists(uploadDir.resolve(stored.getValue().getStoredName())));
    }

    @Test
    void uploadRejectsUnsupportedMimeWithoutWritingRecord() {
        MockMultipartFile file = new MockMultipartFile(
                "file", "note.txt", "text/plain", new byte[]{1});

        assertThrows(BizException.class, () -> service.upload(9L, file));

        verifyNoInteractions(fileMapper);
    }
}
