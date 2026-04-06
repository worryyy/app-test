package com.jb.chat.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.jb.chat.dao.ConversationMbrDao;
import com.jb.chat.entity.ConversationMember;
import com.jb.chat.entity.Message;
import com.jb.chat.mapper.ConversationMbrMapper;
import com.jb.chat.service.MessageService;
import com.jb.common.VO.CusPage;
import com.jb.common.result.R;
import com.jb.common.result.Result;
import com.jb.common.utils.ThreadLocalUtil;
import lombok.extern.slf4j.Slf4j;
import org.bson.types.ObjectId;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.domain.Sort;
import org.springframework.data.mongodb.core.MongoTemplate;
import org.springframework.data.mongodb.core.query.Criteria;
import org.springframework.data.mongodb.core.query.Query;
import org.springframework.stereotype.Service;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;
import org.springframework.util.StringUtils;

import javax.annotation.Resource;
import java.util.Collections;
import java.util.List;
import java.util.Objects;

@Service("messageService")
@Slf4j
public class MessageServiceImpl implements MessageService {

    @Resource
    private MongoTemplate mongoTemplate;

    @Resource
    private ConversationMbrMapper conversationMbrMapper;

    @Resource
    private ConversationMbrDao conversationMbrDao;


    /**
     * 拉取离线消息
     *
     * @param lastMessageId 上次拉取的最后一条消息ID
     * @return 离线消息列表
     */
    @Override
    public Result<?> getMessages(Long lastMessageId) {
        // 获取当前用户ID
        String userId = ThreadLocalUtil.getUserId().toString();

        List<String> conversationIds = getConversationIds(userId);

        if (CollectionUtils.isEmpty(conversationIds)) {
            log.info("用户 {} 没有参与任何会话，返回空消息列表", userId);
            return R.success(Collections.emptyList());
        }

        Criteria criteria = Criteria.where("conversation_id").in(conversationIds);
        // 当lastMessageId为空时拉取所有消息
        if (Objects.nonNull(lastMessageId)) {
            criteria.and("message_id").gt(lastMessageId);
        }
        // 添加按消息时间戳排序
        List<Message> messages = mongoTemplate.find(new Query(criteria).with(Sort.by(Sort.Direction.ASC, "message_id")), Message.class);
        log.info("用户 {} 拉取离线消息，最后一条消息ID: {}, 拉取到 {} 条消息", userId,lastMessageId, messages.size());
        return R.success(messages);

    }

    /**
     * 查询历史消息
     *
     * @param conversationId
     * @param oldestMessageId
     * @return
     */
    @Override
    public Result<?> getHistoryMessages(Integer page, Integer size, String conversationId,Long oldestMessageId) {
        // 获取当前用户ID
        String userId = ThreadLocalUtil.getUserId().toString();
        // 验证会话ID不能为空
        Assert.isTrue(StringUtils.hasText(conversationId), "会话ID不能为空");

        // 检查用户是否为会话成员
        ConversationMember isMember = conversationMbrMapper.selectOne(new QueryWrapper<ConversationMember>()
                .eq("conversation_id", conversationId)
                .eq("user_id", userId));
        if (Objects.isNull(isMember)) {
            log.warn("用户 {} 尝试访问未授权会话 {}", userId, conversationId);
            return R.fail().msg("无权访问该会话历史");
        }

        // 构建游标分页查询
        Criteria criteria = Criteria.where("conversation_id").is(conversationId);
        if (Objects.nonNull(oldestMessageId)) {
            // 游标条件：仅当有上次ID时添加
            criteria.and("message_id").lt(oldestMessageId);
        }

        Query query = new Query(criteria)
                //升序排序
                .with(Sort.by(Sort.Direction.ASC, "message_id"))
                .limit(50);

        List<Message> messages = mongoTemplate.find(query, Message.class);
        CusPage<Message> res = new CusPage<>(messages, page, (long) messages.size(), size);
        log.info("用户 {} 查询会话 {} 历史消息，游标ID: {}, 返回 {} 条消息", userId, conversationId, oldestMessageId, messages.size());
        return R.data(res);
    }

    @Override
    public Result<?> getUnread() {
        // 获取当前用户ID
        String userId = ThreadLocalUtil.getUserId().toString();
        return R.success(conversationMbrDao.HasUnreadOrNot(userId));
    }

    private List<String> getConversationIds(String userId){
        //在会话成员表中查询用户参与的会话ID
        return conversationMbrMapper.selectList(
                new QueryWrapper<ConversationMember>().eq("user_id",userId))
                .stream()
                .filter(Objects::nonNull)
                .map(ConversationMember::getConversationId)
                .toList();
    }
}
