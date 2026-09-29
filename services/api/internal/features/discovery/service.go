package discovery

import (
	"context"
	"strconv"
	"strings"

	"example.com/api/internal/platform/urlsign"
)

type Service struct {
	repo   *Repository
	signer *urlsign.Signer
	ranker Ranker
}

func NewService(repo *Repository, signer *urlsign.Signer, ranker Ranker) *Service {
	if ranker == nil {
		ranker = DefaultRanker{}
	}
	return &Service{repo: repo, signer: signer, ranker: ranker}
}

func parseLimit(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > maxLimit {
		return 0, ErrBadRequest
	}
	return n, nil
}

func (s *Service) Discover(ctx context.Context, userID, rawLimit string) ([]PublicProfile, error) {
	limit, err := parseLimit(rawLimit)
	if err != nil {
		return nil, err
	}
	v, err := s.repo.LoadViewer(ctx, userID)
	if err != nil {
		return nil, err
	}
	fetch := limit * overfetch
	if fetch > maxFetch {
		fetch = maxFetch
	}
	cands, err := s.repo.Candidates(ctx, v, s.ranker.OrderBy(), fetch)
	if err != nil {
		return nil, err
	}
	s.ranker.Rank(cands)
	if len(cands) > limit {
		cands = cands[:limit]
	}
	out := make([]PublicProfile, 0, len(cands))
	if len(cands) == 0 {
		return out, nil
	}
	ids := make([]string, len(cands))
	for i, c := range cands {
		ids[i] = c.UserID
	}
	interests, err := s.repo.Interests(ctx, ids)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.Photos(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, c := range cands {
		p := PublicProfile{
			UserID: c.UserID, FirstName: c.FirstName, Gender: c.Gender, Bio: c.Bio, City: c.City,
			Interests: interests[c.UserID], Photos: []Photo{},
		}
		if p.Interests == nil {
			p.Interests = []string{}
		}
		if c.ShowAge {
			age := c.Age
			p.Age = &age
		}
		if c.ShowDistance {
			d := BucketDistance(c.DistanceKm)
			p.DistanceKm = &d
		}
		for _, ph := range photos[c.UserID] {
			p.Photos = append(p.Photos, Photo{ID: ph.ID, URL: s.signer.PhotoURL(ph.ID), Position: ph.Position})
		}
		out = append(out, p)
	}
	return out, nil
}
