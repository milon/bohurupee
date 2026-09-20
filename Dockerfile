# Release image. GoReleaser places a prebuilt static binary at
# $TARGETPLATFORM/bohurupee. Do not compile in this file.
#
# Mount config at /bohurupee.yaml (auto-discovered) or pass --config.
# Loopback bind values in YAML are upgraded to 0.0.0.0 for port publish.
FROM scratch
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/bohurupee /bohurupee
ENV BOHURUPEE_IN_DOCKER=1
EXPOSE 4190
ENTRYPOINT ["/bohurupee"]
