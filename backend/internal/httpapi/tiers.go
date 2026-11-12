package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"etheyl.local/backend/internal/store"
	"github.com/jackc/pgx/v5"
)

type Tier struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	PriceKopecks int64  `json:"priceKopecks"`
	Position     int32  `json:"position"`
	Used         bool   `json:"used"`
	Archived     bool   `json:"archived"`
}

func tierDTO(t store.SubscriptionTier) Tier {
	return Tier{strconv.FormatInt(t.ID, 10), t.Name, t.Description, t.PriceKopecks, t.Position, t.UsedAt.Valid, t.ArchivedAt.Valid}
}

type tierInput struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	PriceKopecks *int64 `json:"priceKopecks"`
}

func readTier(r *http.Request) (tierInput, error) {
	in, err := decode[tierInput](r)
	if err != nil {
		return in, err
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(in.Name) < 1 || utf8.RuneCountInString(in.Name) > 80 || utf8.RuneCountInString(in.Description) > 5000 || in.PriceKopecks == nil || *in.PriceKopecks < 0 || *in.PriceKopecks > 100000000 {
		return in, invalid()
	}
	return in, nil
}
func (s *Server) ownTiers(w http.ResponseWriter, r *http.Request) error {
	u, err := s.requireUser(r)
	if err != nil {
		return err
	}
	rows, err := s.q.ListTiers(r.Context(), u.ID)
	if err != nil {
		return err
	}
	tiers := make([]Tier, 0, len(rows))
	for _, row := range rows {
		tiers = append(tiers, tierDTO(row))
	}
	return respond(w, 200, map[string]any{"tiers": tiers})
}
func (s *Server) createTier(w http.ResponseWriter, r *http.Request) error {
	in, err := readTier(r)
	if err != nil {
		return err
	}
	tx, q, id, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	tiers, err := q.ListTiers(r.Context(), id)
	if err != nil {
		return err
	}
	active := 0
	for _, t := range tiers {
		if !t.ArchivedAt.Valid {
			active++
		}
	}
	if active >= 50 {
		return problem(422, "tier_limit", "Можно создать не более 50 активных уровней.")
	}
	row, err := q.CreateTier(r.Context(), store.CreateTierParams{AuthorID: id, Name: in.Name, Description: in.Description, PriceKopecks: *in.PriceKopecks})
	if err != nil {
		return err
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 201, map[string]any{"tier": tierDTO(row)})
}
func (s *Server) updateTier(w http.ResponseWriter, r *http.Request) error {
	in, err := readTier(r)
	if err != nil {
		return err
	}
	tierID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || tierID < 1 {
		return missing()
	}
	tx, q, id, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	row, err := q.UpdateTier(r.Context(), store.UpdateTierParams{ID: tierID, AuthorID: id, Name: in.Name, Description: in.Description, PriceKopecks: *in.PriceKopecks})
	if errors.Is(err, pgx.ErrNoRows) {
		return missing()
	}
	if err != nil {
		return err
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"tier": tierDTO(row)})
}
func (s *Server) archiveTier(w http.ResponseWriter, r *http.Request) error {
	return s.retireTier(w, r, false)
}
func (s *Server) deleteTier(w http.ResponseWriter, r *http.Request) error {
	return s.retireTier(w, r, true)
}
func (s *Server) retireTier(w http.ResponseWriter, r *http.Request, remove bool) error {
	tierID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || tierID < 1 {
		return missing()
	}
	tx, q, id, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	var n int64
	if remove {
		n, err = q.DeleteUnusedTier(r.Context(), store.DeleteUnusedTierParams{ID: tierID, AuthorID: id})
		if err != nil {
			return err
		}
	}
	if n == 0 {
		n, err = q.ArchiveTier(r.Context(), store.ArchiveTierParams{ID: tierID, AuthorID: id})
		if err != nil {
			return err
		}
	}
	if n == 0 {
		return missing()
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 200, map[string]string{"message": "Уровень убран с публичной страницы."})
}
func (s *Server) reorderTiers(w http.ResponseWriter, r *http.Request) error {
	in, err := decode[struct {
		IDs []string `json:"ids"`
	}](r)
	if err != nil {
		return err
	}
	tx, q, id, err := s.creatorTx(r)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	rows, err := q.ListTiers(r.Context(), id)
	if err != nil {
		return err
	}
	active := make([]store.SubscriptionTier, 0, len(rows))
	byID := make(map[string]store.SubscriptionTier)
	for _, row := range rows {
		if !row.ArchivedAt.Valid {
			active = append(active, row)
			byID[strconv.FormatInt(row.ID, 10)] = row
		}
	}
	if len(active) != len(in.IDs) {
		return invalid()
	}
	seen := make(map[string]bool)
	for i, key := range in.IDs {
		row, ok := byID[key]
		if !ok || seen[key] {
			return invalid()
		}
		seen[key] = true
		if row.UsedAt.Valid && row.Position != active[i].Position {
			return problem(409, "tier_order_locked", "Позицию используемого уровня менять нельзя.")
		}
		if row.Position != active[i].Position {
			if _, err := q.SetTierPosition(r.Context(), store.SetTierPositionParams{ID: row.ID, AuthorID: id, Position: active[i].Position}); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		return err
	}
	return s.ownTiers(w, r)
}
