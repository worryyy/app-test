FROM rabbitmq:3.12-management

LABEL maintainer="github.com/Milchstrassse"

RUN rabbitmq-plugins enable --offline rabbitmq_management

EXPOSE 5672 15672

CMD ["rabbitmq-server"]
