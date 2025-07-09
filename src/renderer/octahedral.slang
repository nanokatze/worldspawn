[require(spirv_1_0)] module;

// Octahedral Spherical Direction Packing Helpers
// Note: Full 32 bit floats used in these packing functions so that the full precision of 16 bit snorms
// (which is greater than that of a 16 bit float across a good portion of the 0-1 range) is utilized properly.
// https://knarkowicz.wordpress.com/2014/04/16/octahedron-normal-vector-encoding/
// http://jcgt.org/published/0003/02/01/paper.pdf

float2 octWrap(float2 v) {
	float x = copysign(1.0f - abs(v.y), v.x);
	float y = copysign(1.0f - abs(v.x), v.y);

	return float2(x, y);
}

// Converts a normalized spherical vector to a signed octahedral encoding (2 channels of [-1, 1])
public float2 sphereDirectionToSignedOctahedral(float3 v) {
	float2 p = v.xy * (1.0f / (abs(v.x) + abs(v.y) + abs(v.z)));
	if (v.z >= 0.0f)
		p = octWrap(p);
	return p;
}

public float3 signedOctahedralToSphereDirection(float2 p) {
	// Note: Slightly optimized decoding method https://twitter.com/Stubbesaurus/status/937994790553227264
	float3 v = float3(p.x, p.y, 1.0f - abs(p.x) - abs(p.y));
	float t = max(-v.z, 0.0f);

	v.x += (v.x >= 0.0f) ? -t : t;
	v.y += (v.y >= 0.0f) ? -t : t;

	return normalize(v);
}
