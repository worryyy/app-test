package com.jb.chat.controller;

import com.jb.chat.service.MessageService;
import com.jb.common.config.CustomConf;
import com.jb.common.result.Result;
import io.swagger.annotations.Api;
import io.swagger.annotations.ApiOperation;
import org.springframework.web.bind.annotation.*;

import javax.annotation.Nullable;
import javax.annotation.Resource;

@RestController
@RequestMapping("/api/message")
@Api(tags = "消息")
public class MessageController {

    @Resource
    private MessageService messageService;

    @Resource
    private CustomConf customConf;

    @ApiOperation("拉取离线消息")
    @GetMapping("/{last_message_id}")
    public Result<?> pullMessage(@PathVariable(value = "last_message_id") Long lastMessageId) {
        return messageService.getMessages(lastMessageId);
    }

    @ApiOperation("查询历史会话消息")
    @GetMapping("/history_messages")
    public Result<?> getHistoryMessages(@RequestParam(value = "conversation_id") String conversationId,
                                        @RequestParam(value = "oldest_message_id",required = false) Long oldestMessageId,
                                        @RequestParam(value = "page",defaultValue = "1") Integer page,
                                        @RequestParam(value = "size",defaultValue = "0") Integer size) {
        if(page < 0){
            page = 1;
        }
        if (size <= 0){
            size = customConf.getPageSize();
        }else if(size > customConf.getMaxSize()){
            size = customConf.getMaxSize();
        }
        return messageService.getHistoryMessages(page,size,conversationId,oldestMessageId);
    }

    @ApiOperation("未读消息渲染")
    @GetMapping("/unread_messages")
    public Result<?> getUnreadMessages(){
        return messageService.getUnread();
    }

}
