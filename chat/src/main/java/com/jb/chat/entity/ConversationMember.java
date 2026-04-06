package com.jb.chat.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.*;

import java.time.Instant;
import java.time.LocalDateTime;

@Data
@AllArgsConstructor
@NoArgsConstructor
@Builder
@TableName("conversation_members")
public class ConversationMember {
    private String conversationId;

    private String userId;

    @TableField(value = "last_read_message_id")
    private Long lastReadMessageId;

    @TableField(value = "unread_count")
    private Integer unreadCount;

    private LocalDateTime createdAt;
}
