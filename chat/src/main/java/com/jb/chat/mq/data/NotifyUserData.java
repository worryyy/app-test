package com.jb.chat.mq.data;

import com.jb.mq.base.BaseData;
import lombok.Data;
import lombok.EqualsAndHashCode;
import lombok.NoArgsConstructor;

@EqualsAndHashCode(callSuper = true)
@Data
@NoArgsConstructor
public class NotifyUserData extends BaseData<NotifyUserPayload> {
}


