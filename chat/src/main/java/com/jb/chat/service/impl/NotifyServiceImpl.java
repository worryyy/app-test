package com.jb.chat.service.impl;

import com.jb.chat.entity.Notification;
import com.jb.chat.service.NotifyService;
import com.jb.common.VO.CusPage;
import com.jb.common.result.R;
import com.jb.common.result.Result;
import com.jb.common.utils.ThreadLocalUtil;
import com.mongodb.client.result.UpdateResult;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang.StringUtils;
import org.springframework.data.domain.PageRequest;
import org.springframework.data.domain.Sort;
import org.springframework.data.mongodb.core.MongoTemplate;
import org.springframework.data.mongodb.core.query.Criteria;
import org.springframework.data.mongodb.core.query.Query;
import org.springframework.data.mongodb.core.query.Update;
import org.springframework.stereotype.Service;
import org.springframework.util.Assert;

import javax.annotation.Resource;
import java.util.Collections;
import java.util.List;


@Service
@Slf4j
public class NotifyServiceImpl implements NotifyService {

    @Resource
    private MongoTemplate mongoTemplate;


    @Override
    public Result<?> getNotifications(Integer page, Integer size, String type) {
        Assert.isTrue(StringUtils.isNotBlank(type), "type 不能为空");
        String userId = ThreadLocalUtil.getUserId().toString();
        PageRequest request = PageRequest.of(page - 1, size, Sort.by(Sort.Direction.DESC, "_id"));
        Query query = new Query(Criteria.where("receiver_id").is(userId).and("type").is(type));
        List<Notification> notifications = mongoTemplate.find(query.with(request), Notification.class);
        if (notifications.isEmpty()) {
            return R.data(Collections.emptyList());
        }
        log.info("获取 {} 通知: page={}, size={}, notifications={}", type, page, size, notifications);
        
        // 使用 findAndModify 原子性地查找并更新最新通知
        Query updateQuery = buildQuery(userId, type);
        Notification updatedNotification = mongoTemplate.findAndModify(
                updateQuery, 
                new Update().set("is_read", true), 
                Notification.class
        );
        
        if (updatedNotification != null) {
            log.info("原子性更新最新通知已读状态: notificationId={}", updatedNotification.getId());
        }
        
        CusPage<Notification> cusPage = new CusPage<>(notifications, page, (long) notifications.size(), size);
        return R.success(cusPage);
    }

    @Override
    public Result<?> getNotification(String type) {
        Assert.isTrue(StringUtils.isNotBlank(type), "type 不能为空");
        String userId = ThreadLocalUtil.getUserId().toString();
        Query query = buildQuery(userId, type);
        Notification one = mongoTemplate.findOne(query,Notification.class);
        return R.data(one);
    }

    @Override
    public Result<?> haveUnreadNotifications(String type) {
        Assert.isTrue(StringUtils.isNotBlank(type), "type 不能为空");
        String userId = ThreadLocalUtil.getUserId().toString();
        Query query = buildQuery(userId, type);
        // 查询最新的一条通知
        Notification notification = mongoTemplate.findOne(query, Notification.class);
        return R.success(notification != null && !notification.getIsRead());
    }

    private Query buildQuery(String userId, String type) {
        return new Query(Criteria.where("receiver_id").is(userId).and("type").is(type))
                .with(Sort.by(Sort.Direction.DESC, "created_time")).limit(1);
    }
}
