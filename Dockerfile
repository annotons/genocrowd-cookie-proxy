FROM alpine:3.4
EXPOSE 5000

RUN apk update && \
	apk add curl

RUN curl -L https://github.com/annotons/genocrowd-cookie-proxy/releases/download/v0.9.9/genocrowd-cookie-proxy_linux_amd64 > /usr/bin/genocrowd-cookie-proxy && \
	chmod +x /usr/bin/genocrowd-cookie-proxy

ENTRYPOINT ["/usr/bin/genocrowd-cookie-proxy"]
