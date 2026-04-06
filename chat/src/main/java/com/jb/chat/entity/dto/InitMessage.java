package com.jb.chat.entity.dto;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import javax.validation.constraints.Size;
import java.time.Instant;
import java.time.LocalDateTime;

@Data
@AllArgsConstructor
@NoArgsConstructor
@Builder
@TableName("conversations")
public class InitMessage {
    //创建会话所需字段
    @TableId(type = IdType.ASSIGN_ID)
    private String id;

    @Size(max = 512)
    @TableField(value = "last_message_content")
    private String content;

    @TableField(value = "last_message_sender_id")
    private String senderId;

    @TableField(exist = false)
    private String receiverId;

    @TableField(value = "last_message_sent_at")
    private LocalDateTime sentAt = LocalDateTime.now();

    private LocalDateTime createdAt = LocalDateTime.now();

    private LocalDateTime updatedAt = LocalDateTime.now();

    @TableField(exist = false)
    private String handleType;
}
