package com.jb.chat.mq.consumer;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.jb.common.config.MQConf;
import com.jb.mq.base.BaseConsumer;
import com.jb.chat.mq.data.NotifyUserData;
import com.jb.chat.mq.data.NotifyUserPayload;
import com.jb.chat.entity.Notification;
import com.jb.chat.handler.ChatHandler;
import com.rabbitmq.client.Channel;
import lombok.extern.slf4j.Slf4j;
import org.springframework.amqp.core.Message;
import org.springframework.amqp.rabbit.annotation.Queue;
import org.springframework.amqp.rabbit.annotation.Exchange;
import org.springframework.amqp.rabbit.annotation.QueueBinding;
import org.springframework.amqp.rabbit.annotation.RabbitHandler;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.data.mongodb.core.MongoTemplate;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;

import javax.annotation.Resource;
import java.util.Map;

@Component
@Slf4j
public class NotificationConsumer extends BaseConsumer<NotifyUserData> {

    private static final String RDS_KEY = "campus:NOTIFY:";

    @Resource
    private MongoTemplate mongoTemplate;

    private final ObjectMapper objectMapper = new ObjectMapper();

    @RabbitListener(
            bindings = @QueueBinding(
                    value = @Queue(value = MQConf.NOTIFY_USER_QUEUE),
                    exchange = @Exchange(value = MQConf.EXCHANGE)
            )
    )
    @RabbitHandler
    public void consume(NotifyUserData data, Channel channel, Message message) throws Exception {
        handleMessage(RDS_KEY, data, message, channel, d -> {
            log.info(NotificationConsumer.class.getName());
            NotifyUserPayload p = d.getData();
            Notification notification = Notification.builder()
                    .receiverId(p.getReceiverId())
                    .senderId(p.getSenderId())
                    .type(p.getNotifyType())
                    .content(p.getContent())
                    .topicId(p.getTopicId())
                    .commentId(p.getCommentId())
                    .createdTime(p.getCreatedTime())
                    .build();

            mongoTemplate.insert(notification);

            Map<String, WebSocketSession> connections = ChatHandler.getUserConnections();
            WebSocketSession session = connections.get(p.getReceiverId());
            if (session != null && session.isOpen()) {
                try {
                    session.sendMessage(new TextMessage(objectMapper.writeValueAsString(notification)));
                } catch (Exception e) {
                    log.error("实时推送通知失败, user={}, error={}", p.getReceiverId(), e.getMessage(), e);
                }
            }
            return true;
        });
    }
}


