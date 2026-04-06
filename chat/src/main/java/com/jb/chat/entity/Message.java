package com.jb.chat.entity;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import org.springframework.data.mongodb.core.mapping.Document;
import org.springframework.data.mongodb.core.mapping.Field;
import org.springframework.data.mongodb.core.mapping.FieldType;
import org.springframework.data.mongodb.core.mapping.MongoId;

import javax.validation.constraints.Size;
import java.time.Instant;
import java.time.LocalDateTime;
import java.util.Date;

@Data
@AllArgsConstructor
@NoArgsConstructor
@Builder
@Document(collection = "campus_messages")
public class Message {

    @Field("message_id")
    private Long messageId; // 消息ID

    @Field("conversation_id")
    private String conversationId; // 会话ID

    @Field("receiver_id")
    private String receiverId; // 接收者ID

    @Field("sender_id")
    private String senderId; //发送者ID

    @JsonProperty("content")
    private String content; // 消息内容

    //todo 拓展待定
    //private String type; // 消息类型（文本、图片、视频等）

    //private String status; // 消息状态（已发送、已接收、已读）

    private Date sentAt; // 消息发送时间

}
