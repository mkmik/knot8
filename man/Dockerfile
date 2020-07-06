FROM debian/snapshot:stable-20200514@sha256:9f279047515d9326c449625f733b98506e11789c754d26664cf9602fdf768dbe as build

RUN apt-get update && apt-get install -y --no-install-recommends mandoc
RUN mkdir -p /app

COPY knot8.1 header.html footer.html /
RUN (cat /header.html; mandoc -Thtml -Ostyle=/mandoc.css -O fragment /knot8.1; cat /footer.html )>/app/index.html


FROM bitnami/nginx:1.18-debian-10@sha256:1a91023478eb0e97e4f0ff57a60d8b7c6d0b8dbdad798380c1bac8b43c63f0eb

COPY --from=build /app/index.html /app/
COPY mandoc.css Inconsolata.woff2 /app/
