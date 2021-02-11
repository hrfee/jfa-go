FROM golang:latest AS build

COPY . /opt/build

RUN apt update -y \
    && apt install build-essential python3-pip curl software-properties-common sed upx -y \
    && (curl -sL https://deb.nodesource.com/setup_14.x | bash -) \
    && apt install nodejs \
    && (cd /opt/build; make all GOESBUILD=on; make compress) \
    && sed -i 's#id="password_resets-watch_directory" placeholder="/config/jellyfin"#id="password_resets-watch_directory" value="/jf" disabled#g' /opt/build/build/data/html/setup.html

FROM golang:latest

COPY --from=build /opt/build/build /opt/jfa-go

EXPOSE 8056

CMD [ "/opt/jfa-go/jfa-go", "-data", "/data" ]


