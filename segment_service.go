package main

import (
	"context"
	"time"

	"github.com/plu-nz/app.lounge/internal/segments"
)

// SegmentService finds the intro, recap, and credits of the playing video.
type SegmentService struct {
	provider segments.Provider
}

func newSegmentService() *SegmentService {
	return &SegmentService{provider: &segments.Cached{Provider: segments.NewIntroDB()}}
}

// Resolve returns the skippable segments. Chapter titles win; the segment
// provider fills the kinds the chapters lack, and only for series episodes
// (imdbID set). Provider failures are ignored.
func (s *SegmentService) Resolve(chapters []segments.Chapter, duration float64, imdbID string, season, episode int) []segments.Segment {
	fromChapters := segments.FromChapters(chapters, duration)
	var remote []segments.Segment
	if imdbID != "" && season > 0 && episode > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		remote, _ = s.provider.Segments(ctx, imdbID, season, episode)
	}
	return segments.Merge(fromChapters, remote)
}
