package com.jb.chat.handler;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.jb.chat.entity.session.ActSession;
import com.jb.common.utils.JwtHelper;
import com.jb.common.utils.RedisUtils;
import com.jb.common.utils.TokenHelper;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang.StringUtils;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.*;
import org.springframework.web.socket.handler.AbstractWebSocketHandler;
import com.jb.chat.service.ChatService;
import javax.annotation.Resource;
import java.nio.ByteBuffer;
import java.util.Collections;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.ConcurrentHashMap;
import org.springframework.scheduling.annotation.Scheduled;
import java.io.IOException;
import java.util.Iterator;


@Component
@Slf4j
public class ChatHandler extends AbstractWebSocketHandler {
    // 添加常量定义
    private static final long HEARTBEAT_INTERVAL = 30 * 1000; // 30秒
    private static final long SESSION_TIMEOUT = 60 * 1000; // 60秒超时
    private static final long HEARTBEAT_SCHEDULE = 10 * 1000; // 每10秒检查一次心跳超时
    private static final String AUTH_MESSAGE_TYPE = "auth";
    private static final String PING_PAYLOAD = "server_heartbeat";
    //存储活跃会话
    private static final Map<String, ActSession> activeSessions = new ConcurrentHashMap<>();
    private static final Map<String, WebSocketSession> userSessions = new ConcurrentHashMap<>();

    @Resource
    private RedisUtils rds;

    @Resource
    private ChatService chatService;

    @Resource
    private JwtHelper jwtHelper;

    @Resource
    private TokenHelper tokenHelper;

    @Override
    public void afterConnectionEstablished(WebSocketSession session) throws Exception {
        // 连接建立时仅存入临时会话，不关联用户ID
        ActSession actSession = new ActSession(session, null, System.currentTimeMillis());
        activeSessions.put(session.getId(), actSession);
        log.info("新的WebSocket连接建立，sessionId: {}", session.getId());
    }

    @Override
    public void handleTextMessage(WebSocketSession session, TextMessage message) throws Exception {
        String sessionId = session.getId();
        ActSession actSession = activeSessions.get(sessionId);
        if (actSession == null) {
            session.close();
            return;
        }

        String payload = message.getPayload();
        Map<String, Object> messageMap = parseJsonToMap(payload);

        // 处理认证消息
        if (AUTH_MESSAGE_TYPE.equals(messageMap.get("type"))) {
            handleAuthentication(session, actSession, messageMap);
            return;
        }

        // 已认证用户的消息处理
        if (actSession.getUserId() != null) {
            actSession.setUpdateLastTime(System.currentTimeMillis());
            try {
                chatService.handleMessage(message);
            } catch (Exception e) {
                log.error("处理消息异常: {}", e.getMessage(), e);
                // 确保异常传播到事务管理器
                throw new RuntimeException("消息处理异常", e);
            }
        } else {
            // 未认证用户发送其他消息，关闭连接
            session.close();
        }
    }

    //验证token逻辑
    private void handleAuthentication(WebSocketSession session, ActSession actSession, Map<String, Object> messageMap) throws IOException {
        boolean flag = true;
        String token = (String) messageMap.get("token");
        if (Objects.isNull(token) || StringUtils.isBlank(token)) {
            session.close(CloseStatus.POLICY_VIOLATION.withReason("token is required"));
            flag = false;
            return;
        }
        // token格式是否正确
        if(flag) {
            String[] split = token.split("\\.");
            if(split.length != 3) {
                session.close(CloseStatus.POLICY_VIOLATION.withReason("token invalid"));
                flag = false;
            }
        }
        if (flag && rds.get(tokenHelper.getTK(token)) == null) {
            session.close(CloseStatus.POLICY_VIOLATION.withReason("token invalid"));
            flag = false;
        }

        if (flag) {
            flag = jwtHelper.checkToken(token);
            if(!flag) {
               session.close(CloseStatus.POLICY_VIOLATION.withReason("token invalid"));
            }
        }
        //从token中解析出用户标识
        String userId = jwtHelper.getClaims(token).getUserId().toString();

        // 关联用户ID与会话
        actSession.setUserId(userId);
        userSessions.put(userId, session);
        log.info("用户 {} 认证成功，sessionId: {}", userId, session.getId());

        // 发送认证成功响应
        session.sendMessage(new TextMessage("{\"type\":\"auth_success\",\"userId\":\"" + userId + "\"}"));
    }

    @Override
    public void afterConnectionClosed(WebSocketSession session, CloseStatus status) throws Exception {
        String sessionId = session.getId();
        ActSession actSession = activeSessions.remove(sessionId);
        if (actSession != null && actSession.getUserId() != null) {
            userSessions.remove(actSession.getUserId());
            log.info("用户 {} 断开WebSocket连接，sessionId: {}", actSession.getUserId(), sessionId);
        } else {
            log.info("未认证会话断开连接，sessionId: {}", sessionId);
        }
    }

    // 定时检查心跳超时
    @Scheduled(fixedRate = HEARTBEAT_SCHEDULE)
    public void checkHeartbeatTimeout() {
        long currentTime = System.currentTimeMillis();
        Iterator<Map.Entry<String, ActSession>> iterator = activeSessions.entrySet().iterator();

        while (iterator.hasNext()) {
            Map.Entry<String, ActSession> entry = iterator.next();
            ActSession actSession = entry.getValue();

            if (currentTime - actSession.getUpdateLastTime() > SESSION_TIMEOUT) {
                try {
                    WebSocketSession session = actSession.getSession();
                    if (session.isOpen()) {
                        session.close(CloseStatus.SESSION_NOT_RELIABLE.withReason("心跳超时"));
                    }
                } catch (IOException e) {
                    log.error("关闭超时会话失败", e);
                }
                iterator.remove();
                if (actSession.getUserId() != null) {
                    userSessions.remove(actSession.getUserId());
                    log.warn("用户 {} 心跳超时，已断开连接", actSession.getUserId());
                }
            }
        }
    }


    //定时发送向客户端Ping帧以保持连接活跃
    @Scheduled(fixedRate = HEARTBEAT_INTERVAL)
    public void sendServerPing() {
        if (userSessions.isEmpty()) {
            log.trace("当前没有活跃会话，无需发送Ping帧");
            return;
        }

        for (Map.Entry<String, WebSocketSession> entry : userSessions.entrySet()) {
            String userId = entry.getKey();
            WebSocketSession session = entry.getValue();

            // 只对打开状态的会话发送Ping
            if (session.isOpen()) {
                try {
                    PingMessage pingMessage = new PingMessage(ByteBuffer.wrap(PING_PAYLOAD.getBytes()));
                    session.sendMessage(pingMessage);
                    log.trace("向用户 {} 发送Ping帧，会话ID: {}", userId, session.getId());
                } catch (IOException e) {
                    log.error("向用户 {} 发送Ping帧失败", userId, e);
                    // 发送失败时尝试关闭会话并清理
                    try {
                        session.close(CloseStatus.SESSION_NOT_RELIABLE.withReason("Ping发送失败"));
                    } catch (IOException ex) {
                        log.error("关闭异常会话失败", ex);
                    }
                    userSessions.remove(userId);
                }
            } else {
                userSessions.remove(userId);
                log.warn("用户 {} 的会话已关闭，从活跃会话列表中移除", userId);
            }
        }
    }

    @Override
    protected void handlePongMessage(WebSocketSession session, PongMessage message) throws Exception {
        //处理心跳响应
        ActSession actSession = activeSessions.get(session.getId());
        if (actSession != null) {
            //更新最新活跃时间
            actSession.setUpdateLastTime(System.currentTimeMillis());
        }
    }

    /**
     * 获取所有在线用户的WebSocket连接
     * @return
     */
    public static Map<String,WebSocketSession> getUserConnections(){
        return Collections.unmodifiableMap(ChatHandler.userSessions);
    }

    /**
     * 将JSON字符串解析为Map
     * @param json JSON字符串
     * @return 解析后的Map
     */
    private Map<String, Object> parseJsonToMap(String json) {
        try {
            return new ObjectMapper().readValue(json, new TypeReference<Map<String, Object>>() {});
        } catch (Exception e) {
            log.error("解析JSON失败", e);
            return Collections.emptyMap();
        }
    }
}
