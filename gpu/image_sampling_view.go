package gpu

// TODO: delete
type SamplingView struct{ handle uint32 }

// TODO: delet
type SamplingViewWithSampler struct{ handle uint32 }

func (samplingView SamplingView) WithSampler(sampler Sampler) SamplingViewWithSampler {
	// TODO: perform various checks here
	return SamplingViewWithSampler{handle: samplingView.handle | sampler.handle}
}
