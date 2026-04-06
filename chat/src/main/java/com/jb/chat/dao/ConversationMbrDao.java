package com.jb.chat.dao;

import com.baomidou.mybatisplus.extension.service.IService;
import com.jb.chat.entity.ConversationMember;
import org.apache.ibatis.annotations.Mapper;

@Mapper
public interface ConversationMbrDao extends IService<ConversationMember> {

    boolean HasUnreadOrNot(String userId);
}
