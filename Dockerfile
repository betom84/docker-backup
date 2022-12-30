FROM golang:1.19 as build

WORKDIR /build

COPY . .

RUN go mod download && go mod verify
RUN CGO_ENABLED=0 go build -v -o docker-backup

## Deploy
FROM alpine:latest

WORKDIR /opt/docker-backup
ADD build/ .

COPY --from=build /build/docker-backup .

RUN chmod u+x ./docker-backup

#VOLUME /opt/marvin/config

ENTRYPOINT ["/opt/docker-backup/docker-backup"]
CMD ["daemon"]