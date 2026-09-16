package communityclient

import (
	"context"
	"net/http"
	"strconv"
)

// ThreadState is one (thread, user) row, and unread and subscription both ride
// on it. The row is sparse, the way Discourse's topic_users is: it exists only
// once that pair has interacted, so a thread the reader never opened carries no
// row and reports no state at all -- which is not the same as reporting
// everything unread, and is what keeps this from costing a row per thread per
// user.
type ThreadState struct {
	ThreadID           int64 `json:"thread_id"`
	UserID             int   `json:"user_id"`
	LastReadPostNumber int   `json:"last_read_post_number"`
	HighestPostNumber  int   `json:"highest_post_number"`
	UnreadCount        int   `json:"unread_count"`
	NotificationLevel  int   `json:"notification_level"`
}

// MarkRead reports how far a user has read. The mark is monotonic upstream and
// clamped to the thread's highest post, so a stale receipt from a slow tab
// cannot un-read what another tab already read.
func (c *Client) MarkRead(ctx context.Context, threadID int64, userID, postNumber int) (*ThreadState, error) {
	var env envelope[ThreadState]
	payload := map[string]any{"user_id": userID, "last_read_post_number": postNumber}
	lane := "/threads/" + strconv.FormatInt(threadID, 10) + "/read"
	if err := c.do(ctx, http.MethodPost, lane, payload, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) SetNotification(ctx context.Context, threadID int64, userID, level int) (*ThreadState, error) {
	var env envelope[ThreadState]
	payload := map[string]any{"user_id": userID, "level": level}
	lane := "/threads/" + strconv.FormatInt(threadID, 10) + "/notification"
	if err := c.do(ctx, http.MethodPost, lane, payload, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// ThreadStates batches one user's state over a list of threads (≤100 upstream),
// so a screen showing many threads spends one round trip on its badges.
// Threads the user never touched are absent from the answer.
func (c *Client) ThreadStates(ctx context.Context, userID int, threadIDs []int64) ([]ThreadState, error) {
	if len(threadIDs) == 0 {
		return nil, nil
	}
	var env envelope[struct {
		States []ThreadState `json:"states"`
	}]
	payload := map[string]any{"user_id": userID, "thread_ids": threadIDs}
	if err := c.do(ctx, http.MethodPost, "/threads/states", payload, &env); err != nil {
		return nil, err
	}
	return env.Data.States, nil
}
