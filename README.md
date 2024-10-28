<p align="center">
 <img align="center" src="https://raw.githubusercontent.com/breeew/brew/main/assets/logo.png" height="96" />
 <h1 align="center">
  Brew
 </h1>
</p>

![Discord](https://img.shields.io/discord/1293497229096521768)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/breeew/brew/ghcr.yml)

**Brew** is a lightweight and user-friendly Retrieval-Augmented Generation (RAG) system designed to help you build your own second brain. With its fast and easy-to-use interface, Brew empowers users to efficiently manage and retrieve information.

## Community

Join our community on Discord to connect with other users, share ideas, and get support: [Discord Community](https://discord.gg/YGrbmbCVRF).

## Install

### Databases

- Install DB: [pgvector](https://github.com/pgvector/pgvector)，don't forget `CREATE EXTENSION vector;`
- Create database like 'brew'
- Execute create table sqls via `/internal/store/sqlstore/*.sql`

### Service

- Clone & go build cmd/main.go
- Copy default config(cmd/service/etc/service-default.toml) to your config path
  `brew-api service -c {your config path}`

### Web

- [brew web-app](https://github.com/breeew/web-app)
