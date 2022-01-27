FROM golang:1.16

WORKDIR /target

COPY go.* .
RUN go mod download

COPY . .

ENTRYPOINT ["go", "test", "-v", "-count=1", "-tags=sessionintegration", "./internal/session/integration_test"]
