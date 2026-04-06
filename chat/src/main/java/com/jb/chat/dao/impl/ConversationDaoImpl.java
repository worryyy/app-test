package com.jb.chat.dao.impl;

import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import com.jb.chat.dao.ConversationDao;
import com.jb.chat.entity.Conversation;
import com.jb.chat.mapper.ConversationMapper;
import org.springframework.stereotype.Component;

@Component
public class ConversationDaoImpl extends ServiceImpl<ConversationMapper, Conversation> implements ConversationDao {
}
