package com.jb.chat.service;

import com.jb.common.result.Result;

public interface MessageService {
    Result<?> getMessages(Long lastMessageId);

    Result<?> getHistoryMessages(Integer page,Integer size,String conversationId,Long oldestMessageId);

    Result<?> getUnread();
}
