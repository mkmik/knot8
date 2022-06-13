FROM debian/snapshot:stable-20200514@sha256:9f279047515d9326c449625f733b98506e11789c754d26664cf9602fdf768dbe as build

RUN apt-get update && apt-get install -y --no-install-recommends mandoc
RUN mkdir -p /app

COPY knot8.1 header.html footer.html /
RUN (cat /header.html; mandoc -Thtml -Ostyle=/mandoc.css -O fragment /knot8.1; cat /footer.html )>/app/index.html


FROM bitnami/nginx:1.22-debian-10@sha256:0766f88508f8d76d871f4d5dbc73693cd90fa36ef16e914d7320871406798132

COPY --from=build /app/index.html /app/
COPY mandoc.css Inconsolata.woff2 /app/
