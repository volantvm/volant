---
title: "Extensibility and Conventions"
author: "VolantVM"
date: "2025-11-01"
---


## Images

- Manifest schema: docs/6_reference/schemas/image-manifest-v1.json
- Authoring guides: docs/4_image-development/*
- Install flow: POST /api/v1/images with manifest; stored in DB; exposed via registry

## Config Overrides

- vmconfig.Config supports per-VM overrides for boot media, kernel cmdline, API host/port, resources, and cloud-init.
- CLI flags map to override fields; flags take precedence when provided.

## Networking

- Declarative in manifest; overridable in config; easy to add more modes by extending helpers and launcher args.

## Storage/DB

- New entities → add migrations in internal/server/db/sqlite/migrations and repository interfaces in db package.
