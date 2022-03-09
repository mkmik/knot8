FROM debian/snapshot:stable-20200514@sha256:9f279047515d9326c449625f733b98506e11789c754d26664cf9602fdf768dbe as build

RUN apt-get update && apt-get install -y --no-install-recommends mandoc
RUN mkdir -p /app

COPY knot8.1 header.html footer.html /
RUN (cat /header.html; mandoc -Thtml -Ostyle=/mandoc.css -O fragment /knot8.1; cat /footer.html )>/app/index.html


FROM bitnami/nginx:1.21-debian-10@sha256:4bd4935a3f8fe7e42f3a2126d7bfe4ff3a8a76aeae959e1c5688f7d92a7a4676

COPY --from=build /app/index.html /app/
COPY mandoc.css Inconsolata.woff2 /app/
