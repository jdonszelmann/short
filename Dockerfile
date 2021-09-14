FROM golang as build

WORKDIR /build

# Force modules
ENV GO111MODULE=on

# Cache dependencies
COPY go.* ./
RUN go mod download

# Build project
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o short ./services/short

# Run stage
FROM debian

COPY --from=build /build/short /short
COPY --from=build /build/static /static

ENTRYPOINT ["/short"]