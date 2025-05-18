FROM debian/snapshot:stable-20200514@sha256:9f279047515d9326c449625f733b98506e11789c754d26664cf9602fdf768dbe as build

RUN apt-get update && apt-get install -y --no-install-recommends mandoc
RUN mkdir -p /app

COPY knot8.1 header.html footer.html /
RUN (cat /header.html; mandoc -Thtml -Ostyle=/mandoc.css -O fragment /knot8.1; cat /footer.html )>/app/index.html


FROM bitnami/nginx:1.28.0-debian-12-r1@sha256:5293f4a37e6f69e0bba8c9e7ad27f1db71a4519b416c6e48b9702a57a7e37ded

COPY --from=build /app/index.html /app/
COPY mandoc.css Inconsolata.woff2 /app/
