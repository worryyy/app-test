package com.jb.chat.mq.producer;

import com.jb.common.config.MQConf;
import com.jb.mq.base.BaseProducer;
import com.jb.chat.mq.data.NotifyUserData;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

@Component
@Slf4j
public class NotifyUserProducer extends BaseProducer<NotifyUserData> {

    public void produce(NotifyUserData data) {
        init(MQConf.EXCHANGE, MQConf.NOTIFY_USER_KEY);
        log.info("notify content = {}", data.toString());
        sendMessage(data);
    }
}



