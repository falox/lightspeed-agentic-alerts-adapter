FROM golang:1.26 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o adapter ./cmd/adapter/

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /build/adapter /adapter
USER 65532:65532
ENTRYPOINT ["/adapter"]
