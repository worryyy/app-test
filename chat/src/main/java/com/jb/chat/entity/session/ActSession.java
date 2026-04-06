package com.jb.chat.entity.session;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;
import org.springframework.web.socket.WebSocketSession;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class ActSession {
    private WebSocketSession session;
    private String userId;
    private Long updateLastTime;
}
