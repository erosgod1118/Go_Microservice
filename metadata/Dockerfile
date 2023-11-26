FROM alpine:latest

RUN apk update && apk add libc6-compat

COPY main .
COPY configs/. .
EXPOSE 8081
CMD ["./main"]