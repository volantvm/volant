---
title: "API Reference"
author: "VolantVM"
date: "2025-11-01"
---


Generate OpenAPI JSON with:

```bash
make openapi-export
```

This builds `bin/openapi-export` and writes `docs/6_reference/api/openapi.json` with the server URL set to `https://docs.volantvm.com`.

References:
- internal/server/httpapi
- cmd/openapi-export (spec builder)
