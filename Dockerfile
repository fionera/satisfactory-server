FROM golang:1.17-alpine3.13 as build_go
WORKDIR /go/src/app
COPY mod_helper .

RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -v -o mod_helper .

FROM steamcmd/steamcmd:ubuntu-18

RUN set -x \
    && dpkg --add-architecture i386 \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y cron gettext-base sudo wine-stable \
    && mkdir -p /config \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -ms /bin/bash satisfactory

COPY Game.ini Engine.ini Scalability.ini /home/satisfactory/
COPY backup.sh init.sh steamscript.txt /
COPY --from=build_go /go/src/app/mod_helper /

RUN chmod +x "/backup.sh" "/init.sh"

VOLUME /config
WORKDIR /config

ENV GAMECONFIGDIR="/home/satisfactory/.wine/drive_c/users/satisfactory/Local Settings/Application Data/FactoryGame/Saved" \
    MAXBACKUPS=10 \
    STEAMAPPID="526870" \
    STEAMBETA="false"

EXPOSE 7777/udp

ENTRYPOINT ["bash", "-c", "/init.sh"]