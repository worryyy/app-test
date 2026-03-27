FROM redis:7.2

LABEL maintainer="github.com/Milchstrassse"

WORKDIR /data/redis

COPY redis.conf /etc/redis/redis.conf

CMD ["redis-server", "/etc/redis/redis.conf"]
EXPOSE 6379
