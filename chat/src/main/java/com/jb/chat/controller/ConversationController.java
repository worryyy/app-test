package com.jb.chat.controller;

import com.jb.chat.service.ConversationService;
import com.jb.common.result.Result;
import io.swagger.annotations.Api;
import io.swagger.annotations.ApiOperation;
import org.apache.commons.lang.StringUtils;
import org.springframework.util.Assert;
import org.springframework.web.bind.annotation.*;

import javax.annotation.Nullable;
import javax.annotation.Resource;

@RestController
@RequestMapping("/api/conversation")
@Api(tags = "会话")
public class ConversationController {

    @Resource
    private ConversationService conversationService;

    @ApiOperation("获取会话列表")
    @GetMapping
    public Result<?> getConversations() {
        return conversationService.getConversations();
    }

    /**
     * 进入会话时需要更新用户的会话状态
     * @param conversationId
     * @return
     */
    @ApiOperation("进入会话")
    @PutMapping("/conversation_enter")
    public Result<?> enterConversation(
                @RequestParam(value = "conversation_id") String conversationId,
                @RequestParam(value = "last_message_id",required = false)  String lastMessageId) {
        return conversationService.enterConversation(conversationId,lastMessageId);
    }

    @ApiOperation("获取未读消息数")
    @GetMapping("/{conversation_id}/unread_count")
    public Result<?> getConversationMessages(@PathVariable(value = "conversation_id") String conversationId) {
        return conversationService.unreadCount(conversationId);
    }

    @ApiOperation("查询会话")
    @GetMapping("/conversation_query")
    public Result<?> getConversationByTargetUserId(@RequestParam("target_user_id") String targetUserId) {
        return conversationService.getCommonConversation(targetUserId);
    }

    @ApiOperation("聊天会话对象渲染")
    @GetMapping("/profile_by_conversation_id")
    public Result<?> getUserProfileByConversationId(@RequestParam(value = "conversation_id") String conversationId){
        return conversationService.getUserProfile(conversationId);
    }
    @ApiOperation("删除会话")
    @DeleteMapping("/{conversation_id}")
    public Result<?> deleteConversation(@PathVariable(value = "conversation_id") String conversationId) {
        Assert.isTrue(StringUtils.isNotBlank(conversationId), "conversation_id 不能为空");
        return conversationService.deleteConversation(conversationId);
    }
}
