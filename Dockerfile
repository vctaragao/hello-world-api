FROM golang:1.26.3 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd

FROM gcr.io/distroless/static-debian12

COPY --from=build /api /api

EXPOSE 8097

USER nonroot:nonroot
ENTRYPOINT ["/api"]
