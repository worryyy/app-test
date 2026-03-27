FROM mysql:8.0

LABEL maintainer="github.com/Milchstrassse"

WORKDIR /data/mysql

ENV MYSQL_ROOT_PASSWORD='Xy8@Kp3!Mv9#Jq'

COPY my.cnf /etc/mysql/my.cnf

RUN chmod 644 /etc/mysql/my.cnf

EXPOSE 3306

CMD ["mysqld"]
