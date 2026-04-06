package com.jb.chat.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.jb.chat.entity.Conversation;
import com.jb.common.utils.MapperInterface;
import org.apache.ibatis.annotations.Mapper;

@Mapper
public interface ConversationMapper extends BaseMapper<Conversation>, MapperInterface {
}
