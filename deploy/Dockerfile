FROM golang:1.27.2-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/erp ./cmd/erp && \
    CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/erpctl ./cmd/erpctl

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/erp /out/erpctl /usr/local/bin/
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/erp"]
CMD ["serve"]
