# Build stage

# Build Go backend
FROM golang:1.25.1 AS go-builder
WORKDIR /app
COPY . .
RUN make build

# Build React frontend
FROM node:20 AS react-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm install
COPY web .
RUN npm run build

# Final image
FROM ubuntu:22.04
WORKDIR /app
COPY --from=go-builder /app/bin/gozarr /app/bin/gozarr
COPY --from=go-builder /app/internal /app/internal
COPY --from=go-builder /app/rejected_extras.json /app/rejected_extras.json
COPY --from=go-builder /app/go.mod /app/go.mod
COPY --from=go-builder /app/go.sum /app/go.sum
COPY --from=go-builder /app/scripts /app/scripts
COPY --from=go-builder /app/deployments /app/deployments
COPY --from=go-builder /app/Makefile /app/Makefile
COPY --from=react-builder /app/web/dist /app/web/dist
RUN apt-get update && apt-get install -y ffmpeg yt-dlp && rm -rf /var/lib/apt/lists/*
EXPOSE 8080
CMD ["/app/bin/gozarr"]
