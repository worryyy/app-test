package com.jb.chat.service;

import com.jb.common.result.Result;

public interface NotifyService {

    Result<?> haveUnreadNotifications(String type);

    Result<?> getNotifications(Integer page, Integer size, String type);

    Result<?> getNotification(String type);
}
