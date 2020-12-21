FROM debian

RUN apt-get update \
    && apt-get install -y \
        perl \
        libtimedate-perl \
        libemail-sender-perl \
    && apt-get clean -y \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY LICENSE rotating-rsync-backup.pl README.md sample.conf ./

ENTRYPOINT ["perl", "/app/rotating-rsync-backup.pl"]
