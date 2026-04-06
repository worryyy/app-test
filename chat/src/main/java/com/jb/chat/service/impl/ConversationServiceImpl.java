package com.jb.chat.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.baomidou.mybatisplus.core.conditions.update.UpdateWrapper;
import com.jb.chat.dao.ConversationMbrDao;
import com.jb.chat.entity.Conversation;
import com.jb.chat.entity.ConversationMember;
import com.jb.chat.entity.Message;
import com.jb.chat.mapper.ConversationMapper;
import com.jb.chat.mapper.ConversationMbrMapper;
import com.jb.chat.service.ConversationService;
import com.jb.common.result.R;
import com.jb.common.result.RC;
import com.jb.common.result.Result;
import com.jb.common.utils.ThreadLocalUtil;
import com.jb.user.dao.UserDao;
import com.jb.user.service.UserService;
import com.jb.user.vo.UserVO;
import com.jb.userentity.entity.User;
import com.mongodb.client.result.DeleteResult;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.mongodb.core.MongoTemplate;
import org.springframework.data.mongodb.core.query.Criteria;
import org.springframework.data.mongodb.core.query.Query;
import org.springframework.stereotype.Service;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;
import org.springframework.util.StringUtils;
import org.springframework.transaction.annotation.Transactional;

import javax.annotation.Resource;
import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.Objects;

@Service("conversationService")
@Slf4j
public class ConversationServiceImpl implements ConversationService {

    @Resource
    private ConversationMapper conversationMapper;

    @Resource
    private ConversationMbrMapper conversationMbrMapper;

    @Resource
    private ConversationMbrDao conversationMbrDao;

    @Resource
    private UserDao userDao;

    @Resource
    private MongoTemplate mongoTemplate;
    /**
     * 获取用户的会话列表
     * @return 会话列表
     */
    @Override
    public Result<?> getConversations() {
        String userId = ThreadLocalUtil.getUserId().toString();
        List<String> conversationIds = getConversationIds(userId);
        if (CollectionUtils.isEmpty(conversationIds)) {
            return R.success(Collections.emptyList());
        }
        List<Conversation> conversations = conversationMapper.selectList(new QueryWrapper<Conversation>().in("id", conversationIds)
                .orderByDesc("updated_at"));
        log.info("用户 {} 的会话列表: {}", userId, conversations);
        return R.success(conversations);
    }


    /**
     * 获取会话未读数
     * @param conversationId
     * @return
     */
    @Override
    public Result<?> unreadCount(String conversationId) {
        //获取用户id
        String userId = ThreadLocalUtil.getUserId().toString();
        Assert.isTrue(StringUtils.hasText(conversationId),"会话ID不能为空");
        QueryWrapper<ConversationMember> queryWrapper = new QueryWrapper<ConversationMember>()
                .eq("conversation_id", conversationId)
                .eq("user_id",userId)
                .select("unread_count");
        List<ConversationMember> unreadCountList = conversationMbrMapper.selectList(queryWrapper);
        return R.success(unreadCountList);
    }

    /**
     * 用户进入会话时更新未读数
     * @param conversationId 会话ID
     * @param lastMessageId 最后一条消息ID
     * @return 更新结果
     */
    @Override
    public Result<?> enterConversation(String conversationId, String lastMessageId) {
        //获取用户id
        String userId = ThreadLocalUtil.getUserId().toString();
        // 检查会话ID是否为空
        Assert.isTrue(StringUtils.hasText(conversationId),"会话ID不能为空");
        //更新的前提是 未读数不为0 没有新消息也不用更新
        QueryWrapper<ConversationMember> queryWrapper = new QueryWrapper<ConversationMember>()
                .eq("conversation_id", conversationId)
                .eq("user_id", userId);
        ConversationMember member = conversationMbrDao.getOne(queryWrapper);
        if(member == null) {
            log.error("用户 {} 进入会话 {}，未找到会话成员记录", userId, conversationId);
            return R.fail(RC.ERROR_NOT_EXISTED);
        }
        if(member.getUnreadCount() == 0){
            log.info("用户 {} 进入会话 {}，未读数已为0，无需更新", userId, conversationId);
            return R.success();
        }
        // 更新会话成员的未读数为0
        Assert.isTrue(StringUtils.hasText(lastMessageId),"最后一条消息ID不能为空");
        UpdateWrapper<ConversationMember> updateWrapper = new UpdateWrapper<ConversationMember>()
                .eq("conversation_id", conversationId)
                .eq("user_id", userId)
                .set("unread_count", 0)
                .set("last_read_message_id", lastMessageId);
        boolean update = conversationMbrDao.update(updateWrapper);
        if(!update) {
            log.error("用户 {} 进入会话 {}，更新未读数失败", userId, conversationId);
            return R.fail().msg("更新未读数失败");
        }
        log.info("用户 {} 进入会话 {}，已将未读数重置为0,并更新最后一条消息ID为 {}", userId, conversationId,lastMessageId);
        return R.success();
    }

    /**
     * 查询与目标用户的共同会话是否存在
     * @param targetUserId
     * @return
     */
    @Override
    public Result<?> getCommonConversation(String targetUserId) {
        // 获取当前用户ID
        String userId = ThreadLocalUtil.getUserId().toString();
        Assert.isTrue(StringUtils.hasText(targetUserId),"目标用户ID不能为空");
        // 查询与目标用户的会话
        List<String> targetUserIds = getConversationIds(targetUserId);
        List<String> userIds = getConversationIds(userId);
        //求两个list的交集
        List<String> list = targetUserIds.stream()
                .filter(userIds::contains)
                .toList();
        if(list.isEmpty()){
            log.info("用户 {} 和目标用户 {} 没有共同的会话", userId, targetUserId);
            return R.data(Collections.emptyList());
        }
        // 返回共同的会话ID
        log.info("用户 {} 和目标用户 {} 的共同会话ID: {}", userId, targetUserId, list);
        return R.data(list);

    }

    /**
     * 会话列表用户信息渲染
     * @param conversationId
     * @return
     */
    @Override
    public Result<?> getUserProfile(String conversationId) {
        Assert.isTrue(StringUtils.hasText(conversationId),"会话ID不能为空");
        // 获取当前用户ID
        String userId = ThreadLocalUtil.getUserId().toString();
        QueryWrapper<ConversationMember> memberQueryWrapper = new QueryWrapper<ConversationMember>()
                .eq("conversation_id", conversationId)
                .ne("user_id", userId);
        ConversationMember one = conversationMbrDao.getOne(memberQueryWrapper);
        if(Objects.isNull(one)) {
            log.error("会话 {} 中未找到聊天对象", conversationId);
            return R.fail(RC.ERROR_NOT_EXISTED).msg("会话中未找到聊天对象");
        }
        User user = userDao.getById(one.getUserId());
        if(Objects.isNull(user)) {
            log.error("会话 {} 中的用户信息未找到", conversationId);
            return R.fail(RC.ERROR_NOT_EXISTED).msg("会话中用户信息未找到");
        }
        UserVO userVO = UserVO.builder()
                .nickname(user.getNickname())
                .avatar(user.getAvatar())
                .userId(String.valueOf(user.getId()))
                .build();
        return R.data(userVO);

    }

    @Override
    @Transactional(rollbackFor = Exception.class)
    public Result<?> deleteConversation(String conversationId) {
        // 参数验证
        Assert.isTrue(StringUtils.hasText(conversationId), "会话ID不能为空");
        
        // 获取当前用户ID
        String userId = ThreadLocalUtil.getUserId().toString();
        
        log.info("用户 {} 开始删除会话 {}", userId, conversationId);
        
        try {
            // 验证会话是否存在
            Conversation conversation = conversationMapper.selectById(conversationId);
            if (conversation == null) {
                log.warn("用户 {} 尝试删除不存在的会话 {}", userId, conversationId);
                return R.fail(RC.ERROR_NOT_EXISTED).msg("会话不存在");
            }
            
            // 验证用户是否有权限删除该会话（必须是会话成员）
            QueryWrapper<ConversationMember> memberQuery = new QueryWrapper<ConversationMember>()
                    .eq("conversation_id", conversationId)
                    .eq("user_id", userId);
            ConversationMember member = conversationMbrDao.getOne(memberQuery);
            if (member == null) {
                log.warn("用户 {} 尝试删除无权限的会话 {}", userId, conversationId);
                return R.fail().msg("无权限删除该会话");
            }
            
            // 删除会话成员记录
            int memberDeleteCount = conversationMbrMapper.delete(memberQuery);
            if (memberDeleteCount == 0) {
                log.warn("用户 {} 删除会话成员记录失败，会话ID: {}", userId, conversationId);
                return R.fail().msg("删除会话成员失败");
            }
            
            // 删除MongoDB中的消息记录
            DeleteResult messageDeleteResult = mongoTemplate.remove(
                    new Query(Criteria.where("conversation_id").is(conversationId)), 
                    Message.class
            );
            log.info("用户 {} 删除会话 {} 的消息记录，删除数量: {}", 
                    userId, conversationId, messageDeleteResult.getDeletedCount());
            
            // 检查是否还有其他成员，如果没有则删除整个会话
            QueryWrapper<ConversationMember> remainingMembersQuery = new QueryWrapper<ConversationMember>()
                    .eq("conversation_id", conversationId);
            long remainingMemberCount = conversationMbrMapper.selectCount(remainingMembersQuery);
            
            if (remainingMemberCount == 0) {
                // 没有其他成员了，删除整个会话
                int conversationDeleteCount = conversationMapper.deleteById(conversationId);
                if (conversationDeleteCount == 0) {
                    log.warn("删除会话记录失败，会话ID: {}", conversationId);
                    return R.fail().msg("删除会话记录失败");
                }
                log.info("用户 {} 删除会话 {} 成功，会话已无其他成员，已完全删除", userId, conversationId);
            } else {
                log.info("用户 {} 退出会话 {} 成功，会话还有 {} 个其他成员", userId, conversationId, remainingMemberCount);
            }
            
            return R.success().msg("删除会话成功");
            
        } catch (Exception e) {
            log.error("用户 {} 删除会话 {} 时发生异常: {}", userId, conversationId, e.getMessage(), e);
            throw new RuntimeException("删除会话失败: " + e.getMessage(), e);
        }
    }


    /**
     * 获取用户参与的会话ID列表
     * @param userId
     * @return
     */
    private List<String> getConversationIds(String userId) {
        //在会话成员表中查询用户参与的会话ID
        return conversationMbrMapper.selectList(new QueryWrapper<ConversationMember>()
                        .eq("user_id", userId))
                        .stream()
                        .filter(Objects::nonNull)
                        .map(ConversationMember::getConversationId)
                        .toList();
    }
}
