package com.jb.chat.entity;

import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import com.jb.mq.base.BaseData;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import nonapi.io.github.classgraph.json.Id;
import org.springframework.data.mongodb.core.mapping.FieldType;
import org.springframework.data.mongodb.core.mapping.MongoId;

import javax.validation.constraints.NotBlank;
import javax.validation.constraints.Null;
import java.io.Serializable;
import java.sql.Timestamp;
import java.time.Instant;
import java.time.LocalDateTime;

@Data
@AllArgsConstructor
@NoArgsConstructor
@Builder
@TableName("conversations")
public class Conversation implements Serializable {

    @TableId
    @Null
    private String id;

    @NotBlank
    private Integer type;  // 1:单聊 2:群聊

    private String lastMessageContent;

    private String  lastMessageSenderId;

    private LocalDateTime lastMessageSentAt;

    private LocalDateTime createdAt;

    private LocalDateTime updatedAt;
}
