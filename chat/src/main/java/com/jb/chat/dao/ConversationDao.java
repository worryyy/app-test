package com.jb.chat.dao;

import com.baomidou.mybatisplus.extension.service.IService;
import com.jb.chat.entity.Conversation;
import org.apache.ibatis.annotations.Mapper;

@Mapper
public interface ConversationDao extends IService<Conversation> {
}
