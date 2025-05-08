FROM debian/snapshot:stable-20200514@sha256:9f279047515d9326c449625f733b98506e11789c754d26664cf9602fdf768dbe as build

RUN apt-get update && apt-get install -y --no-install-recommends mandoc
RUN mkdir -p /app

COPY knot8.1 header.html footer.html /
RUN (cat /header.html; mandoc -Thtml -Ostyle=/mandoc.css -O fragment /knot8.1; cat /footer.html )>/app/index.html


FROM bitnami/nginx:1.28.0-debian-12-r0@sha256:eaf9066e86f6caced6b42c18beda52e00cb6ec8bb06be8dfa28f74c4d6533006

COPY --from=build /app/index.html /app/
COPY mandoc.css Inconsolata.woff2 /app/
