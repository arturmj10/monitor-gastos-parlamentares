FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/coleta ./cmd/coleta

FROM gcr.io/distroless/static-debian12
COPY --from=build /app/coleta /coleta
EXPOSE 8080
ENTRYPOINT ["/coleta"]
