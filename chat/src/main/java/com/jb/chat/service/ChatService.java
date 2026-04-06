package com.jb.chat.service;


import com.jb.common.result.Result;
import org.springframework.web.socket.TextMessage;

public interface ChatService {

    void handleMessage(TextMessage message);

}
