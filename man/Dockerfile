FROM debian/snapshot:stable-20200514@sha256:9f279047515d9326c449625f733b98506e11789c754d26664cf9602fdf768dbe as build

RUN apt-get update && apt-get install -y --no-install-recommends mandoc
RUN mkdir -p /app

COPY knot8.1 header.html footer.html /
RUN (cat /header.html; mandoc -Thtml -Ostyle=/mandoc.css -O fragment /knot8.1; cat /footer.html )>/app/index.html


FROM bitnami/nginx:1.29.1-debian-12-r0@sha256:b2e803958eda5723aae1e36ed0e418f6e0c79e7ce890820eba9cad85a6381286

COPY --from=build /app/index.html /app/
COPY mandoc.css Inconsolata.woff2 /app/
