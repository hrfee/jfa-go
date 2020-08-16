FROM golang:alpine

RUN apk update

RUN apk add --update make py3-pip curl sed npm build-base python3-dev

ADD . / /opt/build/

RUN (cd /opt/build; make headless)

RUN mv /opt/build/build /opt/jfa-go

RUN rm -rf /opt/build

RUN sed -i 's#id="pwrJfPath" placeholder="Folder"#id="pwrJfPath" value="/jf" disabled#g' /opt/jfa-go/data/templates/setup.html

RUN apk del py3-pip python3-dev build-base python3 nodejs npm

RUN (rm -rf /go; rm -rf /usr/local/go)

EXPOSE 8056

CMD [ "/opt/jfa-go/jfa-go", "-data", "/data" ]


