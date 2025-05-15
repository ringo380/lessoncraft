FROM golang:1.18

COPY . /go/src/lessoncraft

WORKDIR /go/src/lessoncraft

RUN ssh-keygen -N "" -t rsa -f /etc/ssh/ssh_host_rsa_key >/dev/null

RUN CGO_ENABLED=0 go build -a -installsuffix nocgo -o /go/bin/lessoncraft .


FROM alpine

RUN apk --update add ca-certificates
RUN mkdir -p /app/pwd

COPY --from=0 /go/bin/lessoncraft /app/lessoncraft
COPY --from=0 /etc/ssh/ssh_host_rsa_key /etc/ssh/ssh_host_rsa_key

WORKDIR /app
CMD ["./lessoncraft"]

EXPOSE 3000
