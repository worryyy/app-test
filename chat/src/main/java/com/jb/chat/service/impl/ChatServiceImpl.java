package com.jb.chat.service.impl;

import com.alibaba.fastjson.JSONException;
import com.baomidou.mybatisplus.core.conditions.update.UpdateWrapper;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import com.jb.chat.dao.ConversationDao;
import com.jb.chat.dao.ConversationMbrDao;
import com.jb.chat.dao.InitMessageDao;
import com.jb.chat.entity.*;
import com.jb.chat.entity.dto.ChatMessage;
import com.jb.chat.entity.dto.InitMessage;
import com.jb.chat.handler.ChatHandler;
import com.jb.common.utils.SnowflakeIdGenerator;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang.StringUtils;
import org.springframework.data.mongodb.core.MongoTemplate;
import org.springframework.stereotype.Service;
import com.jb.chat.service.ChatService;
import org.springframework.transaction.support.TransactionSynchronizationManager;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;

import javax.annotation.Resource;
import java.io.IOException;
import java.time.LocalDateTime;
import java.util.Map;


@Slf4j
@Service("chatService")


public class ChatServiceImpl implements ChatService {


    private static final String HANDLE_TYPE = "handleType";
    private static final String HANDLE_TYPE_INIT = "INIT";
    //添加jason模块依赖允许LocalDate类字段序列化为时间戳
    private final ObjectMapper objectMapper = new ObjectMapper()
                        .registerModule(new JavaTimeModule()).disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS);

    @Resource
    private InitMessageDao initMessageDao;

    @Resource
    private MongoTemplate mongoTemplate;

    @Resource
    private ConversationMbrDao conversationMbrDao;

    @Resource
    private ConversationDao conversationDao;


    @Override
    public void handleMessage(TextMessage message) throws JSONException,RuntimeException {
        try {
            String payload = message.getPayload();
            if (payload.isBlank()) {
                log.warn("接收到错误格式的消息:{}",payload);
                return;
            }

            Map<String, String> map = parseJsonToMap(payload);
            if (HANDLE_TYPE_INIT.equals(map.get(HANDLE_TYPE))) {
                handleInitMessage(payload);
            } else {
                handleChatMessage(payload);
            }
        } catch (JsonProcessingException e) {
            log.error("无法正常解析消息: {}", e.getMessage(), e);
            throw new RuntimeException("消息解析失败", e);
        } catch (Exception e) {
            //todo 触发补偿机制
            log.error("处理消息出错: {}", e.getMessage(), e);
        }
    }



    private void updateConversationAndMember(ChatMessage chatMessage) {
        if (chatMessage == null || StringUtils.isBlank(chatMessage.getConversationId())) {
            log.warn("错误的ChatMessage对象: {}", chatMessage);
            throw new IllegalArgumentException("ChatMessage对象不能为空或缺少必要字段");
        }

        //更新会话成员的最后阅读消息ID
        UpdateWrapper<ConversationMember> updateLastMessageId = new UpdateWrapper<ConversationMember>()
                .eq("conversation_id", chatMessage.getConversationId())
                .eq("user_id", chatMessage.getSenderId())
                .set("last_read_message_id", chatMessage.getMessageId());
        conversationMbrDao.update(updateLastMessageId);

        //更新消息接收者的未读消息数
        UpdateWrapper<ConversationMember> updateUnreadWrapper = new UpdateWrapper<ConversationMember>()
                .eq("conversation_id", chatMessage.getConversationId())
                .eq("user_id", chatMessage.getReceiverId())
                .setSql("unread_count = unread_count + 1");
        boolean update = conversationMbrDao.update(updateUnreadWrapper);
        if(!update) {
            log.error("更新会话成员未读数失败，conversationId: {}, receiverId: {}", chatMessage.getConversationId(), chatMessage.getReceiverId());
            throw new RuntimeException("更新会话成员未读数失败");
        }
        log.info("更新会话成员未读数成功，conversationId: {}, receiverId: {}", chatMessage.getConversationId(), chatMessage.getReceiverId());

        UpdateWrapper<Conversation> conversationUpdateWrapper = new UpdateWrapper<Conversation>()
                .eq("id", chatMessage.getConversationId())
                .set("last_message_content", chatMessage.getContent())
                .set("last_message_sender_id", chatMessage.getSenderId())
                .set("last_message_sent_at", chatMessage.getSentAt())
                .set("updated_at", LocalDateTime.now());
        update = conversationDao.update(conversationUpdateWrapper);
        if(!update) {
            log.error("更新会话信息失败，conversationId: {}", chatMessage.getConversationId());
            throw new RuntimeException("更新会话信息失败");
        } else {
            log.info("更新会话信息成功，conversationId: {}", chatMessage.getConversationId());
        }
    }


    private void createConversationMember(InitMessage initMessage, Long messageId) throws IllegalArgumentException {
        if (initMessage == null  || initMessage.getSenderId().isBlank() || initMessage.getReceiverId().isBlank()) {
            log.warn("错误的InitMessage对象: {}", initMessage);
            throw new IllegalArgumentException("InitMessage对象不能为空或缺少必要字段");
        }

        // 创建发送者会话成员
        ConversationMember conversationMemberA = ConversationMember.builder()
                .conversationId(initMessage.getId())
                .userId(initMessage.getSenderId())
                .lastReadMessageId(messageId)
                .unreadCount(0)  // 发送者自己的消息未读数应为0
                .createdAt(LocalDateTime.now())
                .build();
        conversationMbrDao.save(conversationMemberA);

        // 创建接收者会话成员
        ConversationMember conversationMemberB = ConversationMember.builder()
                .conversationId(initMessage.getId())
                .userId(initMessage.getReceiverId())
                .lastReadMessageId(null)
                .unreadCount(1)  // 接收者有一条未读消息
                .createdAt(LocalDateTime.now())
                .build();
        log.info("创建会话成员: {} {}",conversationMemberA,conversationMemberB);
        conversationMbrDao.save(conversationMemberB);


    }

    private void forwardMessage(ChatMessage chatMessage) throws RuntimeException{
        //接收用户是否在线
        Map<String, WebSocketSession> userConnections = ChatHandler.getUserConnections();
        WebSocketSession receiverSession = userConnections.get(chatMessage.getReceiverId());
        if (receiverSession != null && receiverSession.isOpen()) {
            try {
                receiverSession.sendMessage(new TextMessage(
                        objectMapper.writeValueAsString(chatMessage)));
            } catch (IOException e) {
                log.error("向用户 {} 发送消息失败: {}", chatMessage.getReceiverId(), e.getMessage(), e);
                throw new RuntimeException("消息发送失败",e);
            }
        }
    }



    // JSON解析工具
    private Map<String, String> parseJsonToMap(String json) throws JsonProcessingException {
        return objectMapper.readValue(json, new TypeReference<>() {});
    }

    // 处理初始化消息
    public void handleInitMessage(String payload) throws JsonProcessingException,RuntimeException {
        InitMessage initMessage = objectMapper.readValue(payload, InitMessage.class);
        // 创建会话
        initMessageDao.save(initMessage);
        log.info("已保存会话: {}", initMessage.getId());

        ChatMessage chatMessage = objectMapper.readValue(payload, ChatMessage.class);
        chatMessage.setConversationId(initMessage.getId());
        // 保存消息至数据库
        chatMessage.setMessageId(SnowflakeIdGenerator.getInstance().nextId());
        mongoTemplate.insert(chatMessage);
        log.info("已保存消息: {}", chatMessage.getMessageId());

        // 转发消息
        forwardMessage(chatMessage);

        // 注册会话成员
        createConversationMember(initMessage, chatMessage.getMessageId());
    }

    // 处理普通聊天消息
    public void handleChatMessage(String payload) throws JsonProcessingException,RuntimeException {
        ChatMessage chatMessage = objectMapper.readValue(payload, ChatMessage.class);
        chatMessage.setMessageId(SnowflakeIdGenerator.getInstance().nextId());
        mongoTemplate.insert(chatMessage);
        forwardMessage(chatMessage);
        //更新会话和成员信息
        updateConversationAndMember(chatMessage);
    }
}