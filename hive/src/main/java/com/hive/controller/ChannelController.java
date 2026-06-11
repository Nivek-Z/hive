package com.hive.controller;

import com.hive.common.ApiResponse;
import com.hive.config.CurrentUid;
import com.hive.model.dto.ChannelVO;
import com.hive.model.dto.CreateChannelReq;
import com.hive.model.dto.UpdateChannelReq;
import com.hive.service.ChannelService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api")
public class ChannelController {

    private final ChannelService channelService;

    public ChannelController(ChannelService channelService) {
        this.channelService = channelService;
    }

    @PostMapping("/hives/{hiveId}/channels")
    public ApiResponse<ChannelVO> create(@CurrentUid long uid, @PathVariable long hiveId,
                                         @Valid @RequestBody CreateChannelReq req) {
        return ApiResponse.ok(channelService.create(uid, hiveId, req));
    }

    @PutMapping("/channels/{id}")
    public ApiResponse<ChannelVO> update(@CurrentUid long uid, @PathVariable long id,
                                         @Valid @RequestBody UpdateChannelReq req) {
        return ApiResponse.ok(channelService.update(uid, id, req));
    }

    @DeleteMapping("/channels/{id}")
    public ApiResponse<Void> delete(@CurrentUid long uid, @PathVariable long id) {
        channelService.delete(uid, id);
        return ApiResponse.ok();
    }
}
