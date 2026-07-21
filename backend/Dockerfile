FROM golang:1.22-alpine AS build

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
COPY backend/go.mod backend/go.sum ./
RUN go mod download -x

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/study-plan-backend .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=build /out/study-plan-backend /app/study-plan-backend

ENV APP_ENV=production \
    PORT=8080 \
    DB_PATH=/data/study_plan.db

VOLUME ["/data"]
EXPOSE 8080
USER appuser
ENTRYPOINT ["/app/study-plan-backend"]
