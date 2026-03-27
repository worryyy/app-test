FROM nginx:1.24.0-alpine

LABEL maintainer="github.com/Milchstrassse"

RUN apk add --no-cache shadow && \
        groupadd -r nginx-user && \
        useradd -r -g nginx-user -s /sbin/nologin nginx-user


RUN rm -rf /etc/nginx/conf.d/default.conf

COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80 443

HEALTHCHECK --interval=60s --timeout=3s --retries=3 \
  CMD curl -f http://localhost/health-check || exit 1

CMD ["nginx", "-g", "daemon off;"]
