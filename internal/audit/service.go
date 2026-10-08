package audit

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/logger"
)

type AuditLog struct {
	ID        string                 `json:"id"`
	ActorID   string                 `json:"actor_id"`
	ActorType string                 `json:"actor_type"`
	Action    string                 `json:"action"`
	EntityType string                `json:"entity_type"`
	EntityID   string                `json:"entity_id"`
	Metadata  map[string]interface{} `json:"metadata"`
	IPAddress string                 `json:"ip_address"`
	CreatedAt string                 `json:"created_at"`
}

type Service struct {
	db *database.DB
	log *logger.Logger
}

func NewService(db *database.DB, log *logger.Logger) *Service {
	return &Service{db: db, log: log}
}

func (s *Service) Log(ctx context.Context, actorID, actorType, action, entityType, entityID string, metadata map[string]interface{}, ipAddress string) error {
	auditLog := &AuditLog{
		ID:         uuid.New().String(),
		ActorID:    actorID,
		ActorType:  actorType,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Metadata:   metadata,
		IPAddress:  ipAddress,
	}

	metadataJSON, _ := json.Marshal(metadata)

	query := `
		INSERT INTO audit_logs (id, actor_id, actor_type, action, entity_type, entity_id, metadata, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`
	_, err := s.db.Pool.Exec(ctx, query,
		auditLog.ID, auditLog.ActorID, auditLog.ActorType, auditLog.Action,
		auditLog.EntityType, auditLog.EntityID, metadataJSON, auditLog.IPAddress)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to write audit log")
		return err
	}

	return nil
}

func (s *Service) GetLogs(ctx context.Context, filter Filter, page, pageSize int) ([]*AuditLog, int, error) {
	offset := (page - 1) * pageSize

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.ActorID != "" {
		whereClause += " AND actor_id = $" + string(rune(argIndex+'0'))
		args = append(args, filter.ActorID)
		argIndex++
	}
	if filter.EntityType != "" {
		whereClause += " AND entity_type = $" + string(rune(argIndex+'0'))
		args = append(args, filter.EntityType)
		argIndex++
	}
	if filter.EntityID != "" {
		whereClause += " AND entity_id = $" + string(rune(argIndex+'0'))
		args = append(args, filter.EntityID)
		argIndex++
	}
	if filter.Action != "" {
		whereClause += " AND action = $" + string(rune(argIndex+'0'))
		args = append(args, filter.Action)
		argIndex++
	}

	countQuery := "SELECT COUNT(*) FROM audit_logs " + whereClause
	var total int
	if err := s.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, actor_id, actor_type, action, entity_type, entity_id, metadata, ip_address, created_at
		FROM audit_logs ` + whereClause + ` ORDER BY created_at DESC LIMIT $` + string(rune(argIndex+'0')) + ` OFFSET $` + string(rune(argIndex+1+'0'))
	args = append(args, pageSize, offset)

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		var metadata []byte
		if err := rows.Scan(&log.ID, &log.ActorID, &log.ActorType, &log.Action, &log.EntityType, &log.EntityID, &metadata, &log.IPAddress, &log.CreatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(metadata, &log.Metadata)
		logs = append(logs, log)
	}

	return logs, total, nil
}

type Filter struct {
	ActorID    string
	EntityType string
	EntityID   string
	Action     string
}