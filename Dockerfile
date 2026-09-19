# Release image. GoReleaser places a prebuilt static binary at
# $TARGETPLATFORM/bohurupee. Do not compile in this file.
FROM scratch
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/bohurupee /bohurupee
ENV BOHURUPEE_IN_DOCKER=1
EXPOSE 4190
ENTRYPOINT ["/bohurupee"]
