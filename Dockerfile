# Use this instead if hrfee/jfa-go-build-docker doesn't support your architecture
# FROM --platform=$BUILDPLATFORM golang:latest AS support
FROM --platform=$BUILDPLATFORM docker.io/hrfee/jfa-go-build-docker:latest AS support
# FROM --platform=$BUILDPLATFORM jfa-go-bd AS support
ARG BUILT_BY
ENV JFA_GO_BUILT_BY=$BUILT_BY

COPY . /opt/build

# RUN curl -sfL https://goreleaser.com/static/run > /goreleaser && chmod +x /goreleaser
RUN cd /opt/build; INTERNAL=off ./scripts/version.sh /goreleaser build --snapshot --skip=validate --clean --id notray-e2ee
RUN mv /opt/build/dist/*_linux_arm_6 /opt/build/dist/placeholder_linux_arm
RUN sed -i 's#id="password_resets-watch_directory" placeholder="/config/jellyfin"#id="password_resets-watch_directory" value="/jf" disabled#g' /opt/build/build/data/html/setup.html

FROM golang:bookworm AS final
ARG TARGETARCH

COPY --from=support /opt/build/dist/*_linux_${TARGETARCH}* /opt/jfa-go
COPY --from=support /opt/build/build/data /opt/jfa-go/

RUN apt-get update -y && apt-get install libolm-dev -y

EXPOSE 8056
EXPOSE 8057

CMD [ "/opt/jfa-go/jfa-go", "-data", "/data" ]
