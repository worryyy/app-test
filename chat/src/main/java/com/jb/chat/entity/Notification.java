package com.jb.chat.entity;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import org.springframework.data.mongodb.core.index.CompoundIndexes;
import org.springframework.data.mongodb.core.index.CompoundIndex;
import org.springframework.data.mongodb.core.index.Indexed;
import org.springframework.data.mongodb.core.mapping.Document;
import org.springframework.data.mongodb.core.mapping.Field;
import org.springframework.data.mongodb.core.mapping.MongoId;
import org.springframework.data.mongodb.core.mapping.FieldType;
import java.util.Date;

@Data
@NoArgsConstructor
@AllArgsConstructor
@Builder
@Document(collection = "campus_notifications")
@CompoundIndexes({
    @CompoundIndex(
        name = "receive_id_type_idx",
        def = "{'receiver_id': 1, 'type': 1}"
    )
})
public class Notification {

    @MongoId(value = FieldType.OBJECT_ID)
    private String id;

    @Indexed
    @Field("receiver_id")
    private String receiverId;

    @Field("sender_id")
    private String senderId;

    @Field("type")
    private String type;

    @Field("content")
    private String content;

    @Field("topic_id")
    private String topicId;

    @Field("comment_id")
    private String commentId;

    @Field("created_time")
    private Date createdTime;

    @Field("is_read")
    private Boolean isRead = false;

}


