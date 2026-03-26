-- name: CreateAuditLog :exec
INSERT INTO audit_logs (org_id, actor_id, action, target_type, target_id, ip_address, user_agent, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE org_id = $1 AND created_at < $2
ORDER BY created_at DESC
LIMIT $3;

-- name: ListAuditLogsByAction :many
SELECT * FROM audit_logs
WHERE org_id = $1 AND action = $2 AND created_at < $3
ORDER BY created_at DESC
LIMIT $4;
