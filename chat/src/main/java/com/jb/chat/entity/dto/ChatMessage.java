package com.jb.chat.entity.dto;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.*;
import org.springframework.data.annotation.Transient;
import org.springframework.data.mongodb.core.index.CompoundIndex;
import org.springframework.data.mongodb.core.index.CompoundIndexes;
import org.springframework.data.mongodb.core.index.Indexed;
import org.springframework.data.mongodb.core.mapping.Document;
import org.springframework.data.mongodb.core.mapping.Field;

import javax.validation.constraints.NotNull;
import javax.validation.constraints.Null;
import javax.validation.constraints.Size;
import java.time.Instant;
import java.time.LocalDateTime;
import java.util.Map;

@Data
@AllArgsConstructor
@NoArgsConstructor
@Builder
@Document(collection = "campus_messages")
@CompoundIndexes({
        // 创建复合索引 (conversation_id + message_id)
        @CompoundIndex(
                name = "conversation_message_idx",
                def = "{'conversation_id': 1, 'message_id': -1}",
                unique = true
        )
})
public class ChatMessage  {

    @Field("message_id")
    private Long messageId;

    @Indexed  // todo 分区键创建索引待定
    @Field("conversation_id")
    private String conversationId;

    @NotNull
    @Field("sender_id")
    private String senderId;

    @NotNull
    @Field("receiver_id")
    private String receiverId;

    @Size(max = 512)
    @Field("content")
    private String content;

    @Field("message_type")
    //todo 消息类型待定拓展
    private Integer messageType;

    @Null
    private LocalDateTime sentAt = LocalDateTime.now();  // 消息发送时间

    @Field("metadata")
    private Map<String, Object> metadata;  // 元数据存储

    @Transient
    private String handleType; // 冗余的json转换字段  避免报错

    }
