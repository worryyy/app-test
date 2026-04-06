package com.jb.chat.dao.impl;

import com.baomidou.mybatisplus.extension.service.impl.ServiceImpl;
import com.jb.chat.dao.InitMessageDao;
import com.jb.chat.mapper.InitMessageMapper;
import com.jb.chat.entity.dto.InitMessage;
import org.springframework.stereotype.Component;

@Component("initMessageDao")
public class InitMessageDaoImpl extends ServiceImpl<InitMessageMapper, InitMessage> implements InitMessageDao {

}