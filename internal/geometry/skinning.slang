struct Uhh {
	uint32_t index;
	float weight;
}

struct Push {
	// TODO: shove these pairs into structs?

	// TODO: everything here should use our formatted buffer stuff (like texel
	// buffer but bda). Including jointIndices and jointWeights

	// TODO: what about the tangents?

	float3 *skinnedPositions;
	// float3 *skinnedNormals;

	float3 *restPositions;
	// float3 *restNormals;

	// TODO: we should interleave indices with weights probably. Or maybe not if
	// we want to use alternative compact formats. .... Or we could just use our
	// formatted buffer stuff.

	Uhh *jointWeights;
	uint32_t jointsPerVertex;

	uint32_t vertexCount;

	matrix<float, 4, 4> *pose; // we kinda ignore the last row so we might as well consider this Mat3x4
}

// TODO: jointIndices and jointWeights should probably be FormattedSlice or
// whatever.
float4x4 calcDeform(matrix<float, 4, 4> *pose, Uhh *jointWeights, uint32_t numJoints)
{
	var deform = (float4x4)0;
	var totalWeight = 0.0;
	for (uint32_t i = 0; i < numJoints; i++) {
		let weight = jointWeights[i].weight;
		deform += weight * pose[jointWeights[i].index];
		totalWeight += weight;
	}
	// TODO: remove this in favor of assuming that jointWeights sums up to 1 and
	// forcing the caller to maintain that.
	deform += max(1 - totalWeight, 0) *
		float4x4(1, 0, 0, 0,
				 0, 1, 0, 0,
				 0, 0, 1, 0,
				 0, 0, 0, 1);
	return deform;
}

[[vk::push_constant]] Push push;

[shader("compute"), numthreads(64, 1, 1)]
void skinMesh(uint32_t index : SV_DispatchThreadID) {
	if (index >= push.vertexCount)
		return;

	let deform = calcDeform(
		push.pose,
		push.jointWeights + index * push.jointsPerVertex, push.jointsPerVertex);

	push.skinnedPositions[index] = mul(deform, float4(push.restPositions[index], 1.0)).xyz;
}