package com.jb.chat.mapper;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.jb.chat.entity.ConversationMember;
import com.jb.common.utils.MapperInterface;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

@Mapper
public interface ConversationMbrMapper extends BaseMapper<ConversationMember>, MapperInterface {

    @Select("select coalesce(sum(unread_count),0) from conversation_members where user_id = #{userId}")
    Integer getUnreadCount(@Param(value = "userId") String userId);
}
