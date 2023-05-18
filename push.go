package stringid

import (
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

const (
	// pushChars are the lexiographically correct base 64 characters used for
	// push-style IDs.
	pushChars = "-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"
)

// PushGenerator is a push-style ID generator that satisifies the Generator
// interface.
type PushGenerator struct {
	seed int64

	// mu is the mutex lock.
	mu sync.Mutex

	// r is the random source.
	r *rand.Rand

	// stamp is the timestamp of the last ID creation, used to prevent
	// collisions if called multiple times during the same millisecond.
	stamp int64

	// stamp is comprised of y bytes of entropy converted to y characters.
	// this is appended to the generated id to prevent collisions.
	// the numeric value is incremented in the event of a collision.
	last []int

	// ret is data retention that allow re-use correlation id again after some
	// specific duration for example if the value is time.Hour then the same
	// correlation id will be used again after one hour
	ret *time.Duration

	_timeLength, _lastLength int
}

// NewPushGenerator creates a new push ID generator.
func NewPushGenerator(r *rand.Rand, ret *time.Duration) *PushGenerator {
	// ensure rand source
	var seed int64
	if r == nil {
		seed = time.Now().UnixNano()
		r = rand.New(rand.NewSource(seed))
	}

	var timeLength, lastLength int
	if ret != nil {
		// source of the math https://stackoverflow.com/a/58095701/10961466
		// 6 are coming from pushChars = 64 = 2^6
		timeLength = int(math.Ceil(math.Log(float64(ret.Milliseconds()))/math.Log(float64(2)))) / 6
	} else {
		timeLength = 8
	}
	lastLength = 6

	// create generator and random entropy
	pg := &PushGenerator{
		r:           r,
		seed:        seed,
		ret:         ret,
		last:        make([]int, lastLength),
		_timeLength: timeLength,
		_lastLength: lastLength,
	}
	for i := 0; i < lastLength; i++ {
		pg.last[i] = r.Intn(64)
	}

	return pg
}

// String satisfies the fmt.Stringer interface.
func (pg *PushGenerator) String() string {
	return strconv.FormatInt(pg.seed, 10)
}

// Generate generates a unique, 20-character push-style ID.
func (pg *PushGenerator) Generate() string {
	var i int

	id := make([]byte, pg._timeLength+pg._lastLength)

	// grab last characters
	pg.mu.Lock()
	now := time.Now().UTC().UnixNano() / 1e6
	if pg.ret != nil {
		now %= pg.ret.Milliseconds()
	}
	if pg.stamp == now {
		for i = 0; i < pg._lastLength; i++ {
			pg.last[i]++
			if pg.last[i] < 64 {
				break
			}
			pg.last[i] = 0
		}
	}
	pg.stamp = now

	// set last characters
	for i = 0; i < pg._lastLength; i++ {
		id[len(id)-1-i] = pushChars[pg.last[i]%64]
	}
	pg.mu.Unlock()

	// set id to first 8 characters
	for i = pg._timeLength; i >= 0; i-- {
		id[i] = pushChars[int(now%64)]
		now /= 64
	}

	return string(id)
}
