module;

import camera;
import gpu;
import lds;

#define M_PI_F (3.1415926535897932f)
#define M_PI_2_F (1.5707963267948966f)
#define M_PI_4_F (0.7853981633974830f)

// TODO: would be nice if we could mark these pointers read-only
struct Push {
	Camera *camera;

	float3 *vertexPositions;
	float3 *vertexNormals;
	// TODO: replace with a pointer to an array of pointers to attributes
	//
	// This is only used by FS now.
	float2 *vertexUVs;

	uint16_t *indexBuffer; // this can be either uint16_t or uint32_t

	matrix<float, 3, 4> transform;

	// Used by FS
	uint8_t *material;
};

[[vk::push_constant]] Push push;

struct VSOut {
	float4 pos : SV_Position;
	float3 pos2; // TODO: should be float hit time
	float3 normal;
	float3 barycentrics; // can be float2
	uint32_t primitiveID;
};

[shader("vertex")]
VSOut vertMain(uint32_t rawVertexID : SV_VertexID) {
	uint32_t vertexID = push.indexBuffer[rawVertexID];

	// Keep this computation uniform
	float4x4 mvp = mul(push.camera->viewProj, float4x4(push.transform, float4(0, 0, 0, 1)));

	float4 pos = float4(push.vertexPositions[vertexID], 1.0f);

	VSOut vsOut = {};
	vsOut.pos = mul(mvp, pos);
	vsOut.pos2 = mul(push.transform, pos).xyz;
	// TODO: this doesn't make sense for non-uniform scaling
	vsOut.normal = normalize(mul(float3x3(push.transform), push.vertexNormals[vertexID]).xyz);
	vsOut.barycentrics[rawVertexID % 3] = 1.0;
	vsOut.primitiveID = rawVertexID / 3;
	return vsOut;
}

struct MaterialShaderletArgs {
	SamplingViewWithSampler BaseColorTexture; // so... we can't choose between UNORM and SRGB sampling at runtime, otherwise mips are invalid.
	float metallic;
	float roughness;
	float ior;
};

struct FSOut {
	float4 color : SV_Target0;
	// float3 motion : SV_Target1; how should we do TAA mask? with sentinel values or another attachment?
};

float distance2(float3 a, float3 b) {
	float3 ab = a - b;
	return dot(ab, ab);
}

#define assert(x)

#define Bounces 1

// TODO: use a enum
#define DimensionX 0
#define DimensionY 1
#define Dimensions 2

float2 pointOnDisk(float2 uv) {
	float theta = 2 * M_PI_F * uv.x;
	float r = sqrt(uv.y);

	return float2(cos(theta), sin(theta)) * r;
}

// Alternative implementation used by Blender's Cycles. Produces worse results for us.
float2 pointOnDiskCycles(float2 uv) {
	float a = 2.0f * uv.x - 1.0f;
	float b = 2.0f * uv.y - 1.0f;

	if (a == 0.0f && b == 0.0f) {
		return float2(0.0);
	}

	float phi, r;
	if (a * a > b * b) {
		r = a;
		phi = M_PI_4_F * (b / a);
	} else {
		r = b;
		phi = M_PI_2_F - M_PI_4_F * (a / b);
	}
	return float2(cos(phi), sin(phi)) * r;
}

[shader("fragment")]
FSOut fragMain(VSOut input) { // TODO: I'm not sure I like the idea of passing in just VSOut as is.
	MaterialShaderletArgs *material = (MaterialShaderletArgs*)push.material;

	// TODO: helper for this garbage
	float2 uv0 = push.vertexUVs[push.indexBuffer[3 * input.primitiveID + 0]];
	float2 uv1 = push.vertexUVs[push.indexBuffer[3 * input.primitiveID + 1]];
	float2 uv2 = push.vertexUVs[push.indexBuffer[3 * input.primitiveID + 2]];
	float2 uv = mul(transpose(float3x2(uv0, uv1, uv2)), input.barycentrics);

	float3 w_o = normalize(push.camera->viewPos - input.pos2);
	float3 n = normalize(input.normal);

	SamplingView blueNoiseTexture = push.camera->blueNoise;
	float4 blueNoiseValue = blueNoiseTexture.At2D(int2(input.pos.xy) & (blueNoiseTexture.Extent2D() - 1));

	float3 color = float3(0.0);

	Roberts lds = Roberts(Dimensions * Bounces);

	uint32_t ldsPointIndex = push.camera->frameNumber % 127;

	float2 lightUV = fract(blueNoiseValue.xy + float2(lds.Get(DimensionX, ldsPointIndex), lds.Get(DimensionY, ldsPointIndex)));

	// do IBL stuff here

	float4 baseColor = material->BaseColorTexture.Sample2D(uv);

	/* <--- light stuff ---> */

	float3 reflected = reflect(-w_o, n);
	float3 reflectionColor = push.camera->sky.SampleCube(float3(-reflected.y, reflected.z, reflected.x)).xyz;
	color += lerp(baseColor.xyz * max(_Lambert({w_o, n, normalize(float3(0.5, 0.5, 1))}), 0.1), reflectionColor, 0.05);

	/* <--- light stuff ---> */

	return {
		float4(color, 1.0),
	};
}

// TODO: rename to something nicer
// TODO: use more descriptive names instead of w_i, w_o, n. E.g. surface_normal, etc
struct Context {
	float3 w_o; // ...
	float3 n;   // rename to surface_normal
	float3 w_i;
};

typedef float2 vec2;
typedef float3 vec3;
typedef float4 vec4;

float _Lambert(Context ctx) {
	return dot(ctx.n, ctx.w_i) * (1.0 / M_PI_F);
}
