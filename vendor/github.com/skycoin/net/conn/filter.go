package conn

type roundTripCount uint32
type rate uint64

type maxBandwidthFilterSample struct {
	sample rate
	time   roundTripCount
}

type maxBandwidthFilter struct {
	zeroValue    rate
	windowLength roundTripCount
	estimates    [3]maxBandwidthFilterSample
}

func newMaxBandwidthFilter(windowLength roundTripCount, zeroValue rate, zeroTime roundTripCount) *maxBandwidthFilter {
	return &maxBandwidthFilter{
		windowLength: windowLength,
		zeroValue:    zeroValue,
		estimates: [3]maxBandwidthFilterSample{
			{zeroValue, zeroTime},
			{zeroValue, zeroTime},
			{zeroValue, zeroTime},
		},
	}
}

func (f *maxBandwidthFilter) Update(newSample rate, r roundTripCount) {
	if f.estimates[0].sample == f.zeroValue ||
		newSample > f.estimates[0].sample ||
		r-f.estimates[2].time > f.windowLength {
		f.Reset(newSample, r)
		return
	}

	if newSample > f.estimates[1].sample {
		f.estimates[1] = maxBandwidthFilterSample{newSample, r}
		f.estimates[2] = f.estimates[1]
	} else if newSample > f.estimates[2].sample {
		f.estimates[2] = maxBandwidthFilterSample{newSample, r}
	}

	if r-f.estimates[0].time > f.windowLength {
		f.estimates[0] = f.estimates[1]
		f.estimates[1] = f.estimates[2]
		f.estimates[2] = maxBandwidthFilterSample{newSample, r}

		if r-f.estimates[0].time > f.windowLength {
			f.estimates[0] = f.estimates[1]
			f.estimates[1] = f.estimates[2]
		}
		return
	}

	if f.estimates[1].sample == f.estimates[0].sample &&
		r-f.estimates[1].time > f.windowLength>>2 {
		f.estimates[1] = maxBandwidthFilterSample{newSample, r}
		f.estimates[2] = f.estimates[1]
	}

	if f.estimates[2].sample == f.estimates[1].sample &&
		r-f.estimates[2].time > f.windowLength>>1 {
		f.estimates[2] = maxBandwidthFilterSample{newSample, r}
	}
}

func (f *maxBandwidthFilter) Reset(newSample rate, r roundTripCount) {
	f.estimates = [3]maxBandwidthFilterSample{
		{newSample, r},
		{newSample, r},
		{newSample, r},
	}
}

func (f *maxBandwidthFilter) GetBest() rate {
	return f.estimates[0].sample
}

func (f *maxBandwidthFilter) GetSecondBest() rate {
	return f.estimates[1].sample
}

func (f *maxBandwidthFilter) GetThirdBest() rate {
	return f.estimates[2].sample
}
