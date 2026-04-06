package com.jb.chat.dao.impl;

import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import com.jb.chat.dao.ConversationMbrDao;
import com.jb.chat.entity.ConversationMember;
import com.jb.chat.mapper.ConversationMbrMapper;
import org.springframework.stereotype.Component;
import org.springframework.util.Assert;
import org.springframework.util.StringUtils;

@Component("conversationMbrDao")
public class ConversationMbrDaoImpl extends ServiceImpl<ConversationMbrMapper, ConversationMember> implements ConversationMbrDao {

    /**
     * 检查用户是否有未读消息
     * @param userId
     * @return
     */
    @Override
    public boolean HasUnreadOrNot(String userId) {
        Assert.isTrue(StringUtils.hasText(userId), "用户id不能为空");
        Integer unreadCount = baseMapper.getUnreadCount(userId);
        return unreadCount != null && unreadCount != 0;
    }
}
