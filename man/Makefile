all: knot8.1

IMAGE = gcr.io/mkm-cloud/knot8-io

oci-build: Dockerfile knot8.1 mandoc.css Inconsolata.woff2
	docker build -t $(IMAGE) .

oci-push: oci-build
	docker push $(IMAGE)

oci-deploy: oci-push
	docker pull $(IMAGE)
	docker inspect --format='{{index .RepoDigests 0}}' $(IMAGE)
	gcloud run deploy --platform managed --region us-central1 knot8 --image $(IMAGE)
