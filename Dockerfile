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
# Install latest ffmpeg and yt-dlp with curl_cffi for impersonation support
RUN apt-get update && apt-get install -y ca-certificates python3 python3-pip wget xz-utils \
	&& rm -rf /var/lib/apt/lists/* \
	&& wget -O - https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz | tar -xJ -C /usr/local/bin --strip-components=1 --wildcards '*/ffmpeg' '*/ffprobe' \
	&& chmod +x /usr/local/bin/ffmpeg /usr/local/bin/ffprobe \
	&& pip3 install --no-cache-dir yt-dlp curl_cffi
EXPOSE 8080
CMD ["/app/bin/gozarr"]
