package stats

// Aggregates are the profile-page statistics derived from stored results.
type Aggregates struct {
	HighestWPM     float64
	AvgWPM         float64
	AvgWPMLast10   float64
	HighestRaw     float64
	HighestAcc     float64
	AvgAcc         float64
	AvgConsistency float64
	PBs            map[int]Result // best result per duration, by WPM
}

// Aggregate computes profile statistics over all results.
func Aggregate(results []Result) Aggregates {
	a := Aggregates{PBs: make(map[int]Result)}
	if len(results) == 0 {
		return a
	}
	for _, r := range results {
		a.AvgWPM += r.WPM
		a.AvgAcc += r.Accuracy
		a.AvgConsistency += r.Consistency
		a.HighestWPM = max(a.HighestWPM, r.WPM)
		a.HighestRaw = max(a.HighestRaw, r.Raw)
		a.HighestAcc = max(a.HighestAcc, r.Accuracy)
		if pb, ok := a.PBs[r.DurationSec]; !ok || r.WPM > pb.WPM {
			a.PBs[r.DurationSec] = r
		}
	}
	n := float64(len(results))
	a.AvgWPM /= n
	a.AvgAcc /= n
	a.AvgConsistency /= n

	last := results
	if len(last) > 10 {
		last = last[len(last)-10:]
	}
	for _, r := range last {
		a.AvgWPMLast10 += r.WPM
	}
	a.AvgWPMLast10 /= float64(len(last))
	return a
}
