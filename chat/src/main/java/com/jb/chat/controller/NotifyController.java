package com.jb.chat.controller;

import com.baomidou.mybatisplus.extension.api.R;
import com.jb.chat.service.NotifyService;
import com.jb.common.config.CustomConf;
import com.jb.common.result.Result;
import io.swagger.annotations.Api;
import io.swagger.annotations.ApiOperation;
import org.springframework.web.bind.annotation.*;

import javax.annotation.Resource;

@RestController
@RequestMapping("/api/notify")
@Api(tags = "通知")
public class NotifyController {

    @Resource
    private NotifyService notifyService;

    @Resource
    private CustomConf customConf;

    @GetMapping()
    @ApiOperation("获取收到的通知")
    public Result<?> getCommentNotifications(@RequestParam(value = "page",defaultValue = "1") Integer page,
                                             @RequestParam(value = "size",defaultValue = "0") Integer size,
                                             @RequestParam(value = "type") String type){
        if(page < 0){
            page = 1;
        }
        if (size <= 0){
            size = customConf.getPageSize();
        }else if(size > customConf.getMaxSize()){
            size = customConf.getMaxSize();
        }
        return notifyService.getNotifications(page,size,type);
    }


    @GetMapping("/{type}/haveUnread")
    @ApiOperation("是否有未读通知")
    public Result<?> haveUnreadNotifications(@PathVariable(value = "type") String type) {
        return notifyService.haveUnreadNotifications(type);
    }

    @GetMapping("/{type}")
    @ApiOperation("最新消息摘要")
    public Result<?> getLatestNotifications(@PathVariable(value = "type") String type) {
        return notifyService.getNotification(type);
    }

}
