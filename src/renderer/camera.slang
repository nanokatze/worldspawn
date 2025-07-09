// TODO: make this a C header so we can share with host?

module;

import gpu;

public struct Camera {
	public uint32_t frameNumber; // TODO: rename

	public SamplingView blueNoise;

	// TODO: move this around in struct to somewhere else
	public SamplingViewWithSampler sky;

	public matrix<float, 4, 4> proj;
	public matrix<float, 4, 4> projInverse;
	public matrix<float, 4, 4> view;
	public matrix<float, 4, 4> viewInverse;

	// Precomputed stuff goes at the end

	public matrix<float, 4, 4> viewProj; // precomputed extras should always go at the end
};
