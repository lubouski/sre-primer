FROM cgr.dev/chainguard/go AS build

WORKDIR /app
COPY go.mod go.sum main.go main_test.go ./
RUN go mod download
RUN go test -v
RUN CGO_ENABLED=0 go build -o /app/server

FROM cgr.dev/chainguard/wolfi-base
WORKDIR /app
COPY --from=build /app/server /app/server
EXPOSE 8080
CMD ["/app/server"]
