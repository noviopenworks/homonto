# End-to-end smoke image: builds the real homonto binary and runs a disposable
# `apply` against a throwaway $HOME inside the container, so the host system is
# never touched. Built and run by scripts/docker-test.sh (and CI).
FROM golang:1.23

WORKDIR /src

# Cache module downloads separately from source.
COPY go.mod go.sum ./
RUN go mod download

# Build the binary from the full source tree.
COPY . .
RUN go build -o /usr/local/bin/homonto .

ENTRYPOINT ["sh", "/src/test/docker/smoke.sh"]
