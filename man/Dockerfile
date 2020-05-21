FROM bitnami/nginx:1.18-debian-10@sha256:1a91023478eb0e97e4f0ff57a60d8b7c6d0b8dbdad798380c1bac8b43c63f0eb

USER 0
RUN install_packages mandoc

COPY knot8.1 /
COPY mandoc.css Inconsolata.woff2 /app/

RUN mandoc -Thtml -Ostyle=/mandoc.css /knot8.1 >/app/index.html

USER 1001

