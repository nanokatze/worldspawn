module;

// TODO: unify all mesh deformations so we do one dispatch with a list of
// transformations to apply in order.

struct Push {
	// TODO: shove these pairs into structs

	float3 *skinnedPositions;
	float3 *skinnedNormals;

	float3 *restPositions;
	float3 *restNormals;

	// TODO: we should interleave indices with weights probably. Or maybe not if
	// we want to use alternative compact formats.

	uint32_t *groupIndices;
	float *groupWeights;
	uint32_t groupElementsPerVertex;

	uint32_t vertexCount;

	matrix<float, 4, 4> *pose; // we kinda ignore the last row so we might as well consider this Mat3x4
};

[[vk::push_constant]] Push push;

// TODO: can we specify WG size from host?
// TODO: use 128?
// TODO: make a RT pipe (a callable per type of transformation) variant of skinMesh?
[shader("compute"), numthreads(64, 1, 1)]
void skinMesh(uint32_t index : SV_DispatchThreadID) {
	if (index >= push.vertexCount)
		return;

	uint32_t *groupIndices = push.groupIndices + index * push.groupElementsPerVertex;
	float *groupWeights = push.groupWeights + index * push.groupElementsPerVertex;

	float3 pos = push.restPositions[index];
	float3 normal = push.restNormals[index];

	// TODO: explicitly pass no. of non-zero elements or whatever, or at least a
	// special value for when there's no influence so we should just do
	//
	//   skinned = I * pos
	//
	// instead of comparing totalWeight after
	float3 skinnedPos = (float3)0;
	float3 skinnedNormal = normal;
	float totalWeight = 0.0;
	for (uint32_t i = 0; i < push.groupElementsPerVertex; i++) {
		float4x4 transform = push.pose[groupIndices[i]];
		float weight = groupWeights[i];
		skinnedPos += mul(transform, float4(pos, 1.0)).xyz * weight;
		totalWeight += weight;
	}
	if (totalWeight < 1e-3) {
		skinnedPos = pos;
		skinnedNormal = normal;
	}

	push.skinnedPositions[index] = skinnedPos;
	push.skinnedNormals[index] = skinnedNormal;
}