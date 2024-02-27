FROM golang:1.22 as builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -installsuffix 'static' .

RUN echo 'nobody:x:65534:65534:nobody:/:' > /etc/passwd && \
    echo 'nobody:x:65534:' > /etc/group


FROM scratch

COPY --from=builder /app/alertmanager2gelf /app/alertmanager2gelf
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Perform any further action as an unprivileged user.
USER nobody:nobody

ENV LISTEN_ON="localhost:5001"
ENV GRAYLOG_ADDR="localhost:12201"
ENV HOST_ID="alert2gelf"

ENTRYPOINT ["/app/alertmanager2gelf"]