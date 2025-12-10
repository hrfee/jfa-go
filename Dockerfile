FROM --platform=$BUILDPLATFORM docker.io/hrfee/jfa-go-build-docker:latest AS support
ARG BUILT_BY
ENV JFA_GO_BUILT_BY=$BUILT_BY

COPY . /opt/build

RUN cd /opt/build; INTERNAL=off UPDATER=docker ./scripts/version.sh goreleaser build --snapshot --skip=validate --clean --id notray-e2ee
RUN mv /opt/build/dist/*_linux_arm_6 /opt/build/dist/placeholder_linux_arm
RUN sed -i 's#id="password_resets-watch_directory" placeholder="/config/jellyfin"#id="password_resets-watch_directory" value="/jf" disabled#g' /opt/build/build/data/html/setup.html

FROM gcr.io/distroless/base:latest AS final
ARG TARGETARCH

COPY --from=support /opt/build/dist/*_linux_${TARGETARCH}* /jfa-go
COPY --from=support /opt/build/build/data /jfa-go/data

EXPOSE 8056
EXPOSE 8057

CMD [ "/jfa-go/jfa-go", "-data", "/data" ]
