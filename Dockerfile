# Use this instead if hrfee/jfa-go-build-docker doesn't support your architecture
# FROM --platform=$BUILDPLATFORM golang:latest AS support
FROM --platform=$BUILDPLATFORM docker.io/hrfee/jfa-go-build-docker:latest AS support

COPY . /opt/build

# Uncomment this if hrfee/jfa-go-build-docker doesn't support your architecture
# RUN apt-get update -y \
#     && apt-get install build-essential python3-pip -y \
#     && (curl -sL https://deb.nodesource.com/setup_current.x | bash -) \
#     && apt-get install nodejs
RUN (cd /opt/build; npm i; make precompile INTERNAL=off GOESBUILD=off) \
    && sed -i 's#id="password_resets-watch_directory" placeholder="/config/jellyfin"#id="password_resets-watch_directory" value="/jf" disabled#g' /opt/build/build/data/html/setup.html

ARG TARGETARCH
ENV GOARCH=$TARGETARCH
ARG BUILT_BY
ENV BUILTBY=$BUILT_BY

RUN apt-get update -y && apt-get install libolm-dev -y

RUN (cd /opt/build; make compile INTERNAL=off UPDATER=docker)

FROM golang:bookworm

RUN apt-get update -y && apt-get install libolm-dev -y
COPY --from=support /opt/build/build /opt/jfa-go

EXPOSE 8056
EXPOSE 8057

CMD [ "/opt/jfa-go/jfa-go", "-data", "/data" ]
