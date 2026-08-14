package quiz

import (
	"math"
	"testing"
	"time"
)

var fsrsNow = time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

func TestScheduleFSRSFirstReviewStabilityOrder(t *testing.T) {
	prevStability := 0.0
	prevDifficulty := 11.0
	for _, grade := range []struct {
		rating string
	}{
		{RatingAgain}, {RatingHard}, {RatingGood}, {RatingEasy},
	} {
		st := ScheduleFSRS(nil, grade.rating, fsrsNow)
		if st.Stability <= prevStability {
			t.Errorf("%s: stability %v should exceed previous %v", grade.rating, st.Stability, prevStability)
		}
		if st.Difficulty >= prevDifficulty {
			t.Errorf("%s: difficulty %v should drop below previous %v", grade.rating, st.Difficulty, prevDifficulty)
		}
		prevStability = st.Stability
		prevDifficulty = st.Difficulty
	}
	if st := ScheduleFSRS(nil, RatingAgain, fsrsNow); st.LearningStep != LearnStepLearning {
		t.Fatalf("first again learning_step = %d, want learning phase", st.LearningStep)
	}
}

func TestScheduleFSRSFirstIntervals(t *testing.T) {
	again := ScheduleFSRS(nil, RatingAgain, fsrsNow)
	if got := time.Duration(again.IntervalSeconds) * time.Second; got != 10*time.Minute {
		t.Fatalf("again interval = %v, want 10m", got)
	}
	good := ScheduleFSRS(nil, RatingGood, fsrsNow)
	if got := time.Duration(good.IntervalSeconds) * time.Second; got != 24*time.Hour {
		t.Fatalf("good interval = %v, want capped first-day 24h", got)
	}
	hard := ScheduleFSRS(nil, RatingHard, fsrsNow)
	if got := time.Duration(hard.IntervalSeconds) * time.Second; got >= 24*time.Hour {
		t.Fatalf("hard interval %v should sit below good's 24h", got)
	}
	easy := ScheduleFSRS(nil, RatingEasy, fsrsNow)
	if got := time.Duration(easy.IntervalSeconds) * time.Second; got <= 24*time.Hour {
		t.Fatalf("easy interval %v should sit above good's 24h", got)
	}
}

func TestScheduleFSRSStabilityGrowsOnSuccess(t *testing.T) {
	prev := ScheduleFSRS(nil, RatingGood, fsrsNow.Add(-24*time.Hour))
	next := ScheduleFSRS(&prev, RatingGood, fsrsNow)
	if next.Stability <= prev.Stability {
		t.Fatalf("stability should grow on Good: %v -> %v", prev.Stability, next.Stability)
	}
	if next.LearningStep != LearnStepReview {
		t.Fatalf("expected review phase, got %d", next.LearningStep)
	}
	if !next.DueAt.After(fsrsNow) {
		t.Fatalf("due_at %v should be in the future", next.DueAt)
	}
}

func TestScheduleFSRSStabilityDecaysOnLapse(t *testing.T) {
	prev := ScheduleFSRS(nil, RatingGood, fsrsNow.Add(-24*time.Hour))
	// Give the card a second success so stability is meaningfully above the
	// first-day baseline before we fail it.
	prev = ScheduleFSRS(&prev, RatingGood, fsrsNow.Add(-23*time.Hour))
	lapsed := ScheduleFSRS(&prev, RatingAgain, fsrsNow)
	if lapsed.Stability >= prev.Stability {
		t.Fatalf("stability should fall on Again: %v -> %v", prev.Stability, lapsed.Stability)
	}
	if lapsed.LearningStep != LearnStepLearning {
		t.Fatalf("lapse should drop to learning step, got %d", lapsed.LearningStep)
	}
	if got := time.Duration(lapsed.IntervalSeconds) * time.Second; got != 10*time.Minute {
		t.Fatalf("lapse interval = %v, want 10m requeue", got)
	}
}

func TestScheduleFSRSHardBelowGoodBelowEasyInterval(t *testing.T) {
	// With the corrected FSRS-4.5 hard/easy multipliers (w15=0.6014 for
	// Hard, w16=1.8729 for Easy), a second-day review after an initial Good
	// strictly orders the intervals.
	prev := ScheduleFSRS(nil, RatingGood, fsrsNow.Add(-24*time.Hour))
	hard := ScheduleFSRS(&prev, RatingHard, fsrsNow)
	good := ScheduleFSRS(&prev, RatingGood, fsrsNow)
	if hard.Stability >= good.Stability {
		t.Fatalf("hard stability %v should be below good %v", hard.Stability, good.Stability)
	}
	if time.Duration(hard.IntervalSeconds)*time.Second >= time.Duration(good.IntervalSeconds)*time.Second {
		t.Fatalf("hard interval %v should be below good %v", hard.IntervalSeconds, good.IntervalSeconds)
	}
	easy := ScheduleFSRS(&prev, RatingEasy, fsrsNow)
	if easy.Stability <= good.Stability {
		t.Fatalf("easy stability %v should exceed good %v", easy.Stability, good.Stability)
	}
	if time.Duration(easy.IntervalSeconds)*time.Second <= time.Duration(good.IntervalSeconds)*time.Second {
		t.Fatalf("easy interval %v should exceed good %v", easy.IntervalSeconds, good.IntervalSeconds)
	}
}

func TestConvertSM2ToFSRS(t *testing.T) {
	prev := ReviewState{
		Repetitions:     4,
		IntervalSeconds: int64(7 * 24 * time.Hour / time.Second),
		EaseFactor:      2.5,
		DueAt:           fsrsNow.Add(3 * 24 * time.Hour),
	}
	out := ConvertSM2ToFSRS(prev)
	wantStability := (7 * 24 * time.Hour).Hours() / 24 / 2.5
	if math.Abs(out.Stability-wantStability) > 1e-9 {
		t.Fatalf("stability = %v, want %v", out.Stability, wantStability)
	}
	wantD := 8 - 4*(2.5-minEaseFactor)/(maxEaseFactor-minEaseFactor)
	if math.Abs(out.Difficulty-wantD) > 1e-9 {
		t.Fatalf("difficulty = %v, want the EF=2.5-band midpoint %v", out.Difficulty, wantD)
	}
	if out.LearningStep != LearnStepReview {
		t.Fatalf("converted rows graduate, learning_step = %d", out.LearningStep)
	}
	if !out.DueAt.Equal(prev.DueAt) {
		t.Fatalf("due_at must survive conversion: %v -> %v", prev.DueAt, out.DueAt)
	}
	// Extreme ease factors clamp to the FSRS difficulty range.
	hard := ConvertSM2ToFSRS(ReviewState{IntervalSeconds: 86400, EaseFactor: minEaseFactor})
	if hard.Difficulty != 8 {
		t.Fatalf("min EF should map to D=8, got %v", hard.Difficulty)
	}
	easy := ConvertSM2ToFSRS(ReviewState{IntervalSeconds: 86400, EaseFactor: maxEaseFactor})
	if easy.Difficulty != 4 {
		t.Fatalf("max EF should map to D=4, got %v", easy.Difficulty)
	}
}

func TestRetrievabilityCurveShape(t *testing.T) {
	if got := fsrsR(0, 10); math.Abs(got-1) > 1e-9 {
		t.Fatalf("R(0) = %v, want 1", got)
	}
	r3, r7 := fsrsR(3, 10), fsrsR(7, 10)
	if !(r3 < 1 && r7 < r3 && r7 > 0) {
		t.Fatalf("curve should decay: R(3)=%v R(7)=%v", r3, r7)
	}
}
