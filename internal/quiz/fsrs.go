// FSRS-4.5 (Free Spaced Repetition Scheduler) port, pure Go.
//
// Weight vector is the 21-parameter FSRS-4.5 default set published by the
// open-spaced-repetition community (Jarrett Ye et al.), in the canonical
// ordering used by the reference python/go implementations. Ratings map to
// grades 1..4 = Again/Hard/Good/Easy, matching the existing SM-2 state
// machine so DTOs and the four-button UI stay unchanged.
package quiz

import (
	"math"
	"time"
)

// Scheduler ids persisted per user in user_settings ("sm2" is the default).
const (
	SchedulerSM2  = "sm2"
	SchedulerFSRS = "fsrs"
)

// Learning-step states persisted in the INTEGER learning_step column.
// 0 = learning/relearning (an Again left the card on the 10-minute requeue),
// 1 = graduated review phase.
const (
	LearnStepLearning = 0
	LearnStepReview   = 1
)

// Again always requeues a card for 10 minutes; the first Good interval is
// capped at this SM-2-compatible one-day step.
const (
	againRequeueDelay = 10 * time.Minute
	firstGoodMaxStep  = 24 * time.Hour
)

// fsrsDefaultW is the reference 21-weight FSRS-4.5 set from py-fsrs
// scheduler.py (open-spaced-repetition/py-fsrs). Ratings map to grades
// 1..4 = Again/Hard/Good/Easy, matching the existing SM-2 state machine so
// DTOs and the four-button UI stay unchanged.
//
//	w[0..3]   initial stability after a first Again/Hard/Good/Easy
//	w[4]      initial-difficulty baseline (D for grade 1 is w[4]-e^0+1)
//	w[5]      initial-difficulty exponent per grade past 1
//	w[6]      linear-damping scale on difficulty movement per grade
//	w[7]      mean-reversion blend: weight of the initial-difficulty target
//	w[8]      ln-amplitude of post-recall stability growth
//	w[9]      diminishing recall stability growth as S rises
//	w[10]     recall growth drive as retrievability drops
//	w[11]     post-lapse stability amplitude
//	w[12]     lapse decay with difficulty
//	w[13]     lapse retention of previous stability
//	w[14]     lapse growth drive as retrievability drops
//	w[15]     Hard multiplier on recall stability growth
//	w[16]     Easy multiplier on recall stability growth
//	w[17]     short-term (same-day) stability coefficient
//	w[18]     short-term grade offset
//	w[19]     short-term stability decay in S
//	w[20]     forgetting-curve decay (drives the retrievability factor)
var fsrsDefaultW = [21]float64{
	0.212, 1.2931, 2.3065, 8.2956,
	6.4133, 0.8334, 3.0194, 0.001,
	1.8722, 0.1666, 0.796,
	1.4835, 0.0614, 0.2629, 1.6483,
	0.6014, 1.8729, 0.5425, 0.0912, 0.0658,
	0.1542,
}

// desired retention for the interval conversion; our old fixed curve.
const fsrsDesiredRetention = 0.9

var (
	fsrsDecay  = -fsrsDefaultW[20]
	fsrsFactor = math.Pow(fsrsDesiredRetention, 1/fsrsDecay) - 1
)

func fsrsClampD(d float64) float64 { return math.Max(1, math.Min(10, d)) }
func fsrsClampS(s float64) float64 { return math.Max(0.001, math.Min(36500, s)) }

// elapsedDays returns whole UTC calendar days between two instants. The
// FSRS-4.5 math is day-granular; the 10-minute requeue bypasses it.
func elapsedDays(last, now time.Time) float64 {
	if now.Before(last) {
		return 0
	}
	return now.UTC().Truncate(24*time.Hour).Sub(last.UTC().Truncate(24*time.Hour)).Hours() / 24
}

// fsrsR estimates retrievability after elapsed days using the FSRS-4.5
// power forgetting curve: R(t) = (1 + FACTOR * t / S)^DECAY.
func fsrsR(elapsed, stability float64) float64 {
	if stability <= 0 {
		return 0
	}
	const factor = 19.0 / 81.0
	const decay = -0.5
	return math.Pow(1+factor*elapsed/stability, decay)
}

// nextStability computes post-recall stability for grades 2..4:
//
//	S' = S * (1 + e^w8 * (11-D) * S^-w9 * (e^(w10*(1-R))-1) * w15/w16 modifiers)
func nextStability(prev ReviewState, grade int, elapsed float64) float64 {
	w := fsrsDefaultW
	r := fsrsR(elapsed, prev.Stability)
	bonus := 1.0
	switch grade {
	case 2:
		bonus = w[15]
	case 4:
		bonus = w[16]
	}
	factor := math.Exp(w[8]) * (11 - prev.Difficulty) * math.Pow(prev.Stability, -w[9]) *
		(math.Exp(w[10]*(1-r)) - 1) * bonus
	return fsrsClampS(prev.Stability * (1 + factor))
}

// nextStabilityLapse computes stability after an Again:
//
//	S' = min(w11 * D^-w12 * ((S+1)^w13 - 1) * e^(w14*(1-R)), S / e^(w17*w18))
func nextStabilityLapse(prev ReviewState, elapsed float64) float64 {
	w := fsrsDefaultW
	r := fsrsR(elapsed, prev.Stability)
	long := w[11] * math.Pow(prev.Difficulty, -w[12]) *
		(math.Pow(prev.Stability+1, w[13]) - 1) * math.Exp(w[14]*(1-r))
	short := prev.Stability / math.Exp(w[17]*w[18])
	return fsrsClampS(math.Min(long, short))
}

// nextShortTermStability handles same-day reviews:
//
//	S' = S * e^(w17 * (grade - 3 + w18)) * S^-w19
//
// clamped at >= S for successful grades so an in-day drill never regresses.
func nextShortTermStability(prev ReviewState, grade int) float64 {
	w := fsrsDefaultW
	gain := math.Exp(w[17]*(float64(grade)-3+w[18])) * math.Pow(prev.Stability, -w[19])
	if grade > 1 && gain < 1 {
		gain = 1
	}
	return fsrsClampS(prev.Stability * gain)
}

// NextDifficulty applies linear damping then mean reversion:
//
//	delta = -w6*(grade-3)
//	damped = D + (10-D)*delta/9
//	D' = clamp( w7 * initialDifficultyRaw(Easy) + (1-w7) * damped )
func NextDifficulty(d float64, grade int) float64 {
	w := fsrsDefaultW
	delta := -w[6] * float64(grade-3)
	damped := d + (10-d)*delta/9
	target := initialDifficultyRaw(4)
	return fsrsClampD(w[7]*target + (1-w[7])*damped)
}

// intervalFromStability converts stability into a wall-clock interval at
// the configured desired retention: the number of days until retrievability
// falls to fsrsDesiredRetention. Full-day granularity (py-fsrs behaviour).
func intervalFromStability(stability float64, _ int) time.Duration {
	if stability <= 0 {
		return 24 * time.Hour
	}
	days := stability / fsrsFactor * (math.Pow(fsrsDesiredRetention, 1/fsrsDecay) - 1)
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}
	return time.Duration(math.Round(days*24)) * time.Hour
}

// initialDifficultyRaw returns the unclamped first-review difficulty for a
// grade; NextDifficulty mean-reverts toward the grade-4 (Easy) value.
func initialDifficultyRaw(grade int) float64 {
	return fsrsDefaultW[4] - math.Exp(fsrsDefaultW[5]*float64(grade-1)) + 1
}

// initDifficulty returns the clamped first-review difficulty for a grade.
func initDifficulty(grade int) float64 { return fsrsClampD(initialDifficultyRaw(grade)) }

// ScheduleFSRS is the FSRS equivalent of ScheduleReview. All timestamps UTC.
// The returned state fills IntervalSeconds/EaseFactor so SM-2 DTOs keep
// working: IntervalSeconds still describes the next interval; EaseFactor
// stays 2.5 as an inert placeholder for existing JSON consumers.
func ScheduleFSRS(prev *ReviewState, rating string, now time.Time) ReviewState {
	grade := 3
	switch rating {
	case RatingAgain:
		grade = 1
	case RatingHard:
		grade = 2
	case RatingEasy:
		grade = 4
	}
	now = now.UTC()

	if prev == nil || prev.Stability <= 0 {
		// First review. Again requeues in 10 minutes; Good is capped at the
		// SM-2-compatible first-day step so new cards never jump to 20 days.
		stability := fsrsClampS(fsrsDefaultW[grade-1])
		var interval time.Duration
		if grade == 1 {
			interval = againRequeueDelay
		} else {
			interval = intervalFromStability(stability, grade)
			// Keep first-session order: Again < Hard < Good <= Easy. Hard is
			// squeezed under Good's cap so a first Hard never buries a card
			// longer than Good would. intervalFromStability is day-floored,
			// so Hard floors to 1 day first, then this squeeze applies.
			if grade == 3 && interval > firstGoodMaxStep {
				interval = firstGoodMaxStep
			}
			if grade == 2 && interval >= 24*time.Hour {
				interval = time.Hour
			}
		}
		interval = clampInterval(interval)
		learningStep := LearnStepReview
		if grade == 1 {
			// An initial Again is a learning requeue, just like a lapse.
			learningStep = LearnStepLearning
		}
		return ReviewState{
			Repetitions:     1,
			IntervalSeconds: int64(interval / time.Second),
			EaseFactor:      2.5,
			Difficulty:      initDifficulty(grade),
			Stability:       stability,
			LearningStep:    learningStep,
			DueAt:           now.Add(interval),
			LastReviewedAt:  now,
			LastRating:      rating,
		}
	}

	state := *prev
	elapsed := elapsedDays(state.LastReviewedAt, now)
	state.Difficulty = NextDifficulty(state.Difficulty, grade)
	state.Repetitions++
	state.LastReviewedAt = now
	state.LastRating = rating
	state.LearningStep = LearnStepReview
	if grade == 1 {
		if elapsed < 1 {
			state.Stability = nextShortTermStability(state, grade)
		} else {
			state.Stability = nextStabilityLapse(state, elapsed)
		}
		state.LearningStep = LearnStepLearning
		interval := clampInterval(againRequeueDelay)
		state.IntervalSeconds = int64(interval / time.Second)
		state.DueAt = now.Add(interval)
		return state
	}
	if elapsed < 1 {
		state.Stability = nextShortTermStability(state, grade)
	} else {
		state.Stability = nextStability(state, grade, elapsed)
	}
	interval := clampInterval(intervalFromStability(state.Stability, grade))
	state.IntervalSeconds = int64(interval / time.Second)
	state.DueAt = now.Add(interval)
	return state
}

// ConvertSM2ToFSRS estimates FSRS difficulty/stability from an SM-2 row so a
// user can flip schedulers without losing history. Stability is seeded from
// the interval and difficulty from the ease factor; Because SM-2's final
// interval already contains one ease-factor multiplier, dividing the raw
// interval by EF once undoes that last bump and lands on the felt stability.
// Difficulty maps EF inversely across [minEaseFactor, maxEaseFactor] onto
// the FSRS band [4, 8]; EF 2.5 lands at 5.88, a mild difficulty just above
// FSRS's Good-first baseline. The original DueAt is kept so nothing
// reschedules on the flip.
func ConvertSM2ToFSRS(prev ReviewState) ReviewState {
	out := prev
	ef := out.EaseFactor
	if ef <= 0 {
		ef = 2.5
	}
	if out.Stability <= 0 {
		days := float64(out.IntervalSeconds) / 86400
		if days < 0.1 {
			days = 0.1
		}
		out.Stability = fsrsClampS(days / ef)
	}
	if out.Difficulty <= 0 {
		ratio := math.Max(0, math.Min(1, (ef-minEaseFactor)/(maxEaseFactor-minEaseFactor)))
		out.Difficulty = fsrsClampD(8 - ratio*4)
	}
	out.LearningStep = LearnStepReview
	return out
}
