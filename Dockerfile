# GoReleaser Dockerfile - uses pre-built binary
FROM scratch

# Build arguments for labels
ARG BUILD_DATE
ARG VERSION
ARG VCS_REF
ARG VCS_URL="https://github.com/czechbol/request-raccoon"

# Add OCI-compliant labels to the final image
LABEL org.opencontainers.image.created=${BUILD_DATE}
LABEL org.opencontainers.image.authors="czechbol"
LABEL org.opencontainers.image.url=${VCS_URL}
LABEL org.opencontainers.image.documentation="${VCS_URL}#readme"
LABEL org.opencontainers.image.source=${VCS_URL}
LABEL org.opencontainers.image.version=${VERSION}
LABEL org.opencontainers.image.revision=${VCS_REF}
LABEL org.opencontainers.image.vendor="czechbol"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.title="Request Raccoon"
LABEL org.opencontainers.image.description="A fast HTTP logging server that catches every request! Perfect for webhook debugging and request monitoring."

# Add ca-certificates for any HTTPS requests
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the pre-built binary from GoReleaser
COPY request-raccoon /request-raccoon

# Run the application
ENTRYPOINT ["/request-raccoon"]