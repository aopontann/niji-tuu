# にじ通
- にじさんじの動画公開時などに通知をするWEBアプリ

### 使用技術
- [Go](https://go.dev/)
- [FCM](https://firebase.google.com/docs/cloud-messaging?hl=ja)
- [PostgreSQL](https://www.postgresql.org/)
- [Bun](https://bun.uptrace.dev/)
- [Astro](https://astro.build/)
- [Bulma](https://bulma.io/)

### デプロイ
バッチ
```
ko build ./cmd/batch
gcloud functions deploy batch-check \
--gen2 \
--runtime=go121 \
--region=asia-northeast1 \
--source=. \
--entry-point=check \
--trigger-http \
--allow-unauthenticated

gcloud functions deploy batch-song \
--gen2 \
--runtime=go121 \
--region=asia-northeast1 \
--source=. \
--entry-point=song \
--trigger-http \
--allow-unauthenticated

gcloud functions deploy batch-topic \
--gen2 \
--runtime=go121 \
--region=asia-northeast1 \
--source=. \
--entry-point=topic \
--trigger-http \
--allow-unauthenticated

gcloud functions deploy batch-demo \
--gen2 \
--runtime=go121 \
--region=asia-northeast1 \
--source=. \
--entry-point=demo \
--trigger-http \
--allow-unauthenticated
```
WEBページ
```
ko build ./frontend
npx wrangler pages deploy dist --branch main
```

### メモ
```
export GOOGLE_APPLICATION_CREDENTIALS="token.json"
export KO_DOCKER_REPO=asia-northeast1-docker.pkg.dev/niji-tuu/buildpacks-dev
gcloud auth configure-docker asia-northeast1-docker.pkg.dev
gcloud config set project
gcloud auth application-default login
```

### DBコンテナ
```
podman run --name niji-tuu-postgres -e POSTGRES_PASSWORD=example -p 5432:5432 -d docker.io/library/postgres:16.3
podman start niji-tuu-postgres
podman stop niji-tuu-postgres
```
