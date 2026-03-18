# Zagforge — Storage & Snapshot Format [Phase 2]

## GCS Object Layout

Uses full UUIDs for org and repo, and full 40-character commit SHAs:

```
gs://zagforge-snapshots/
  {org_uuid}/
    {repo_uuid}/
      {full_commit_sha}/
        snapshot.json
```

Example:
```
gs://zagforge-snapshots/a1b2c3d4-e5f6-7890-abcd-ef1234567890/f9e8d7c6-b5a4-3210-fedc-ba0987654321/3fa912e1abc456def789012345678901abcdef01/snapshot.json
```

**IAM roles:**
- Worker container service account: `roles/storage.objectCreator` on the bucket (write-only)
- API service account: `roles/storage.objectViewer` on the bucket (read-only)

---

## Snapshot Format

```json
{
  "snapshot_version": 1,
  "zigzag_version": "0.11.0",
  "commit_sha": "3fa912e1abc456def789012345678901abcdef01",
  "branch": "main",
  "generated_at": "2026-03-14T12:00:00Z",
  "summary": {
    "source_files": 42,
    "total_lines": 8500,
    "total_size_bytes": 245000,
    "languages": [
      { "name": "go", "files": 30, "lines": 6200 },
      { "name": "sql", "files": 12, "lines": 2300 }
    ]
  },
  "files": [
    {
      "path": "cmd/api/main.go",
      "language": "go",
      "lines": 87,
      "content": "package main\n..."
    }
  ]
}
```

The `snapshot_version` field allows the API to handle format migrations as Zigzag evolves.
