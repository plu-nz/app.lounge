package stremio

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// libraryCollection is the datastore collection holding the account's
// library items (stremio-core's LIBRARY_COLLECTION_NAME).
const libraryCollection = "libraryItem"

// LibraryItemState is the watch state of a library item. Durations and
// offsets are in milliseconds, as in stremio-core.
type LibraryItemState struct {
	LastWatched string `json:"lastWatched"`
	TimeOffset  int64  `json:"timeOffset"`
	Duration    int64  `json:"duration"`
	TimesWatch  int64  `json:"timesWatched"`
	Flagged     int64  `json:"flaggedWatched"`
	VideoID     string `json:"video_id"`
}

// LibraryItem is an account library item. The server's fields that
// Lounge does not use are kept in raw and sent back unchanged, so a push
// never drops data written by other Stremio clients.
type LibraryItem struct {
	ID      string           `json:"_id"`
	Name    string           `json:"name"`
	Type    string           `json:"type"`
	Poster  string           `json:"poster"`
	Removed bool             `json:"removed"`
	Temp    bool             `json:"temp"`
	CTime   string           `json:"_ctime"`
	MTime   string           `json:"_mtime"`
	State   LibraryItemState `json:"state"`

	raw map[string]json.RawMessage
}

// MTimeMillis returns the modification time in Unix milliseconds, or 0.
func (i LibraryItem) MTimeMillis() int64 {
	t, err := time.Parse(time.RFC3339, i.MTime)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

// FormatTime formats Unix milliseconds the way the Stremio API does.
func FormatTime(ms int64) string {
	return time.UnixMilli(ms).UTC().Format("2006-01-02T15:04:05.000Z")
}

// UnmarshalJSON decodes the known fields and keeps the rest.
func (i *LibraryItem) UnmarshalJSON(data []byte) error {
	type plain LibraryItem
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*i = LibraryItem(p)
	i.raw = raw
	return nil
}

func putFields(m map[string]json.RawMessage, fields map[string]any) error {
	for k, v := range fields {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		m[k] = b
	}
	return nil
}

// MarshalJSON writes the known fields over the preserved server fields.
func (i LibraryItem) MarshalJSON() ([]byte, error) {
	out := map[string]json.RawMessage{}
	for k, v := range i.raw {
		out[k] = v
	}
	state := map[string]json.RawMessage{}
	if b, ok := out["state"]; ok {
		_ = json.Unmarshal(b, &state)
	}
	// Fields the API requires that a brand-new item would lack.
	for k, v := range map[string]any{"timeWatched": 0, "overallTimeWatched": 0, "noNotif": false} {
		if _, ok := state[k]; !ok {
			if err := putFields(state, map[string]any{k: v}); err != nil {
				return nil, err
			}
		}
	}
	if _, ok := out["posterShape"]; !ok {
		_ = putFields(out, map[string]any{"posterShape": "poster"})
	}
	err := putFields(state, map[string]any{
		"lastWatched": i.State.LastWatched, "timeOffset": i.State.TimeOffset,
		"duration": i.State.Duration, "timesWatched": i.State.TimesWatch,
		"flaggedWatched": i.State.Flagged, "video_id": i.State.VideoID,
	})
	if err != nil {
		return nil, err
	}
	err = putFields(out, map[string]any{
		"_id": i.ID, "name": i.Name, "type": i.Type, "poster": i.Poster,
		"removed": i.Removed, "temp": i.Temp, "_ctime": i.CTime, "_mtime": i.MTime,
		"state": state,
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(out)
}

// LibraryItems returns every library item of the account, including
// removed ones, which are the history Stremio keeps.
func (c *Client) LibraryItems(ctx context.Context, authKey string) ([]LibraryItem, error) {
	var res []LibraryItem
	err := c.api(ctx, "datastoreGet", map[string]any{
		"authKey": authKey, "collection": libraryCollection, "ids": []string{}, "all": true,
	}, &res)
	return res, err
}

// PutLibraryItems writes library items to the account.
func (c *Client) PutLibraryItems(ctx context.Context, authKey string, items []LibraryItem) error {
	if len(items) == 0 {
		return nil
	}
	var res struct {
		Success bool `json:"success"`
	}
	err := c.api(ctx, "datastorePut", map[string]any{
		"authKey": authKey, "collection": libraryCollection, "changes": items,
	}, &res)
	if err == nil && !res.Success {
		return errors.New("the Stremio API did not confirm the library update")
	}
	return err
}
