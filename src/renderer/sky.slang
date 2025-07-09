module;

import camera;
import gpu;

struct Push {
	Camera *camera;
};

[[vk::push_constant]] Push push;

struct VSOut {
	float4 pos : SV_Position;
	float2 uv;
};

static float2 bigTriangle[] = {float2(-1, -1), float2(-1, 3), float2(3, -1)};

[shader("vertex")]
VSOut skyVert(uint32_t rawVertexID : SV_VertexID) {
	return {float4(bigTriangle[rawVertexID], 0, 1), bigTriangle[rawVertexID]};
}

struct FSOut {
	float4 color0 : SV_Target0;
};

[shader("fragment")]
FSOut skyFrag(float2 uv) {
	float3 rayDir = normalize(mul(push.camera->viewInverse, float4(mul(push.camera->projInverse, float4(uv.x, uv.y, 1, 1)).xyz, 0)).xyz);

	// TODO: make a matrix to convert from our convention to KTX2 convention
	return {
		push.camera->sky.SampleCube(float3(-rayDir.y, rayDir.z, rayDir.x)),
	};
}
