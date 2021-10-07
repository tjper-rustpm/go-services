FROM golang:1.16

WORKDIR /target

COPY go.* .
RUN go mod download

COPY . .

ENTRYPOINT ["go", "test", "-v", "-count=1", "-timeout=60m", "-tags=integration", "./cmd/cronman/graph/integration_test"]
