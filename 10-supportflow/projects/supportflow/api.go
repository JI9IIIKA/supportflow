package supportflow

import (
	"strings"
	p "supportflow/internal/platform"
)

func Category(text string) string {
	text = strings.ToLower(text)
	for _, word := range []string{"оплат", "платеж", "payment", "refund"} {
		if strings.Contains(text, word) {
			return "billing"
		}
	}
	for _, word := range []string{"ошиб", "error", "bug"} {
		if strings.Contains(text, word) {
			return "technical"
		}
	}
	return "general"
}
func Audit(r *p.Request, id, action string) error {
	return r.Exec(`INSERT INTO ticket_audit(tenant,ticket_id,actor,action) VALUES($1,$2,$3,$4)`, r.Tenant, id, r.User, action)
}
func Register(a *p.App) {
	a.Handle("POST", "/tickets", func(r *p.Request) (any, error) {
		var b struct {
			Subject string `json:"subject"`
			Message string `json:"message"`
		}
		if e := r.Decode(&b); e != nil {
			return nil, e
		}
		if strings.TrimSpace(b.Subject) == "" || strings.TrimSpace(b.Message) == "" {
			return nil, p.Bad("subject and message required")
		}
		id := p.ID()
		result, e := r.One(`INSERT INTO tickets(tenant,id,owner_id,subject,message) VALUES($1,$2,$3,$4,$5) RETURNING *`, r.Tenant, id, r.User, b.Subject, b.Message)
		if e != nil {
			return nil, e
		}
		return result, Audit(r, id, "created")
	})
	a.Handle("GET", "/tickets", func(r *p.Request) (any, error) {
		return r.All(`SELECT *,sla_at<now() AND status NOT IN ('sent','closed') AS overdue FROM tickets WHERE tenant=$1 ORDER BY created_at DESC LIMIT 100`, r.Tenant)
	})
	a.Handle("GET", "/tickets/{id}", func(r *p.Request) (any, error) {
		return r.One(`SELECT * FROM tickets WHERE tenant=$1 AND id=$2`, r.Tenant, r.Param("id"))
	})
	a.Handle("POST", "/tickets/{id}/draft", func(r *p.Request) (any, error) {
		var text, status string
		if e := r.Tx.QueryRow(r.Ctx, `SELECT message,status FROM tickets WHERE tenant=$1 AND id=$2 FOR UPDATE`, r.Tenant, r.Param("id")).Scan(&text, &status); e != nil {
			return nil, e
		}
		if status != "new" {
			return nil, p.Conflict("draft already prepared")
		}
		category := Category(text)
		draft, mode, e := p.AI(r.Ctx, "support", map[string]any{"message": text, "category": category})
		if e != nil {
			return nil, e
		}
		result, e := r.One(`UPDATE tickets SET status='drafted',category=$3,draft=$4,version=version+1 WHERE tenant=$1 AND id=$2 RETURNING *`, r.Tenant, r.Param("id"), category, draft)
		if e != nil {
			return nil, e
		}
		result["mode"] = mode
		return result, Audit(r, r.Param("id"), "drafted")
	})
	a.Handle("POST", "/tickets/{id}/approve", func(r *p.Request) (any, error) {
		if r.Role != "admin" {
			return nil, p.Fail(403, "human operator admin token required")
		}
		var b struct {
			Version int    `json:"expected_version"`
			Reply   string `json:"reply"`
		}
		if e := r.Decode(&b); e != nil {
			return nil, e
		}
		if strings.TrimSpace(b.Reply) == "" {
			return nil, p.Bad("reviewed reply required")
		}
		result, e := r.One(`UPDATE tickets SET status='approved',draft=$4,version=version+1 WHERE tenant=$1 AND id=$2 AND version=$3 AND status='drafted' RETURNING *`, r.Tenant, r.Param("id"), b.Version, b.Reply)
		if e != nil {
			return nil, e
		}
		return result, Audit(r, r.Param("id"), "approved")
	})
	// Sends to the local customer inbox. External delivery uses the outbox event.
	a.Handle("POST", "/tickets/{id}/send", func(r *p.Request) (any, error) {
		if r.Role != "admin" {
			return nil, p.Fail(403, "operator required")
		}
		var state, body string
		if e := r.Tx.QueryRow(r.Ctx, `SELECT status,COALESCE(draft,'') FROM tickets WHERE tenant=$1 AND id=$2 FOR UPDATE`, r.Tenant, r.Param("id")).Scan(&state, &body); e != nil {
			return nil, e
		}
		if state == "sent" {
			return map[string]bool{"duplicate": true}, nil
		}
		if state != "approved" {
			return nil, p.Conflict("operator approval required")
		}
		if e := r.Exec(`INSERT INTO support_replies(tenant,ticket_id,body) VALUES($1,$2,$3)`, r.Tenant, r.Param("id"), body); e != nil {
			return nil, e
		}
		if e := r.Exec(`UPDATE tickets SET status='sent',version=version+1 WHERE tenant=$1 AND id=$2`, r.Tenant, r.Param("id")); e != nil {
			return nil, e
		}
		if e := Audit(r, r.Param("id"), "sent"); e != nil {
			return nil, e
		}
		return nil, r.Event("ticket.reply", map[string]string{"ticket_id": r.Param("id"), "reply": body})
	})
	a.Handle("GET", "/tickets/{id}/reply", func(r *p.Request) (any, error) {
		return r.One(`SELECT * FROM support_replies WHERE tenant=$1 AND ticket_id=$2`, r.Tenant, r.Param("id"))
	})
	a.Handle("GET", "/tickets/{id}/audit", func(r *p.Request) (any, error) {
		return r.All(`SELECT actor,action,created_at FROM ticket_audit WHERE tenant=$1 AND ticket_id=$2 ORDER BY id`, r.Tenant, r.Param("id"))
	})
}
