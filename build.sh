export GIT_COMMIT=$(git rev-parse --short HEAD) && \
    go build -trimpath -ldflags="-s -w -X main.Version=${GIT_COMMIT} -X main.APIRoot=https://fnradio.jaren.wtf" .