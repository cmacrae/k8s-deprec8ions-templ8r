FROM golang:alpine as builder
RUN apk add --no-cache git ca-certificates && update-ca-certificates
ENV UID=10001
RUN adduser \
	--disabled-password \
	--gecos "" \
	--home "/nonexistent" \
	--no-create-home \
	--shell "/sbin/nologin" \
	--uid 10001 \
	kdt

WORKDIR /kdt
COPY main.go go.mod go.sum /kdt/
RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o kdt
RUN mkdir /kdt/swagger
RUN chown -R kdt:kdt /kdt

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /kdt /kdt
WORKDIR /kdt
USER kdt:kdt
ENTRYPOINT ["/kdt/kdt"]
