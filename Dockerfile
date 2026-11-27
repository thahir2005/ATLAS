FROM golang:1.27.4-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY services/region-api ./services/region-api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /atlas-region-api ./services/region-api

FROM scratch
COPY --from=builder /atlas-region-api /atlas-region-api
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/atlas-region-api"]
CMD ["--listen=:8080", "--region=Container"]
