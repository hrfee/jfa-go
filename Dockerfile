FROM --platform=$BUILDPLATFORM golang:latest AS support

COPY . /var/build

RUN apt-get update -y \
    && apt-get install build-essential python3-pip curl software-properties-common sed -y \
    && (curl -sL https://deb.nodesource.com/setup_current.x | bash -) \
    && apt-get install nodejs \
    && (cd /var/build; make configuration npm email typescript variants-html bundle-css inline-css swagger copy INTERNAL=off GOESBUILD=on) \
    && sed -i 's#id="password_resets-watch_directory" placeholder="/config/jellyfin"#id="password_resets-watch_directory" value="/jf" disabled#g' /var/build/build/data/html/setup.html


FROM --platform=$BUILDPLATFORM golang:latest AS build
ARG TARGETARCH
ENV GOARCH=$TARGETARCH

COPY --from=support /var/build /var/build

RUN (cd /var/build; make compile INTERNAL=off UPDATER=docker)

FROM golang:latest

COPY --from=build /var/build/build /usr/bin/jfa-go

EXPOSE 8056
EXPOSE 8057

VOLUME /config

ENTRYPOINT ["/usr/bin/jfa-go/jfa-go", \
            "-data", "/config"]