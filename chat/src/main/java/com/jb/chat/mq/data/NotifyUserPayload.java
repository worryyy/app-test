package com.jb.chat.mq.data;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.io.Serializable;
import java.util.Date;

@Data
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class NotifyUserPayload implements Serializable {

    /**
     * 接收者用户ID
     */
    private String receiverId;

    /**
     * 触发者用户ID
     */
    private String senderId;

     // 通知类型：NotifyType.COMMENT_ADD、NotifyType.COMMENT_REPLY、NotifyType.TOPIC_LIKE、NotifyType.TOPIC_COLLECTION
    private String notifyType;

    /**
     * 关联的帖子ID
     */
    private String topicId;

    /**
     * 关联的评论ID（可选）
     */
    private String commentId;

    /**
     * 展示的提示内容
     */
    private String content;

    /**
     * 创建时间
     */
    private Date createdTime;
}



