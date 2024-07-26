# Use this instead if hrfee/jfa-go-build-docker doesn't support your architecture
# FROM --platform=$BUILDPLATFORM golang:latest AS support
FROM --platform=$BUILDPLATFORM hrfee/jfa-go-build-docker AS support

COPY . /opt/build

# Uncomment this if hrfee/jfa-go-build-docker doesn't support your architecture
# RUN apt-get update -y \
#     && apt-get install build-essential python3-pip -y \
#     && (curl -sL https://deb.nodesource.com/setup_current.x | bash -) \
#     && apt-get install nodejs
RUN (cd /opt/build; make configuration npm email typescript variants-html bundle-css inline-css swagger copy INTERNAL=off GOESBUILD=on) \
    && sed -i 's#id="password_resets-watch_directory" placeholder="/config/jellyfin"#id="password_resets-watch_directory" value="/jf" disabled#g' /opt/build/build/data/html/setup.html

FROM --platform=$BUILDPLATFORM golang:latest AS build
ARG TARGETARCH
ENV GOARCH=$TARGETARCH
ARG BUILT_BY
ENV BUILTBY=$BUILT_BY

COPY --from=support /opt/build /opt/build

RUN (cd /opt/build; make compile INTERNAL=off UPDATER=docker)

FROM golang:latest

COPY --from=build /opt/build/build /opt/jfa-go

EXPOSE 8056
EXPOSE 8057

CMD [ "/opt/jfa-go/jfa-go", "-data", "/data" ]


