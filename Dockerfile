FROM ubuntu:22.04
ENV DEBIAN_FRONTEND noninteractive
LABEL org.opencontainers.image.description "Logicmonitor Bootstrap Collector"

# NTP is needed for some collector operations
RUN apt update && apt-get update \
  && apt install software-properties-common -y \
  && add-apt-repository ppa:deadsnakes/ppa \
  && apt-get install --no-install-recommends -y \
  tcl \
  inetutils-traceroute \
  file \
  iputils-ping \
  ntp \
  perl \
  procps \
  xxd \
  curl \
  wget \
  && apt-get -y clean \
  && rm -rf /var/lib/apt/lists/*

RUN curl -s https://api.github.com/repos/vkumbhar94/lm-bootstrap-collector/releases/latest \
| grep "browser_download_url.*`(uname -s)`.*86_64" \
| cut -d : -f 2,3 \
| tr -d \" \
| wget -qi - \
&& ls | grep -e "lm-bootstrap.*.tar.gz" | xargs tar -zxf && cp lm-bootstrap-collector /usr/local/bin/ \
&& cp lm-bootstrap-collector /usr/local/bin/lmbc
RUN mkdir /usr/local/logicmonitor

COPY ./entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
