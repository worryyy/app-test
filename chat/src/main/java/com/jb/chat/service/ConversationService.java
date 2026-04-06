package com.jb.chat.service;

import com.jb.common.result.Result;

public interface ConversationService {
    Result<?> getConversations();

    Result<?> unreadCount(String conversationId);

    Result<?> enterConversation(String conversationId, String lastMessageId);

    Result<?> getCommonConversation(String targetUserId);

    Result<?> getUserProfile(String conversationId);

    Result<?> deleteConversation(String conversationId);
}
