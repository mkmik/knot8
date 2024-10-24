FROM debian/snapshot:stable-20200514@sha256:9f279047515d9326c449625f733b98506e11789c754d26664cf9602fdf768dbe as build

RUN apt-get update && apt-get install -y --no-install-recommends mandoc
RUN mkdir -p /app

COPY knot8.1 header.html footer.html /
RUN (cat /header.html; mandoc -Thtml -Ostyle=/mandoc.css -O fragment /knot8.1; cat /footer.html )>/app/index.html


FROM bitnami/nginx:1.27.2-debian-12-r2@sha256:43933dcafcf90f34d249c84c11e3d0a77a056a183c4f91bdcacb8ad029706e2c

COPY --from=build /app/index.html /app/
COPY mandoc.css Inconsolata.woff2 /app/
