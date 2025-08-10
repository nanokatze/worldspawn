module;

import gpu;

// TODO: get rid of this shader and split it into gather/scatter memcpy and
// compute transforms + scatter?

float4 rot3nlerp(float4 a, float4 b, float t) {
    float u = 1 - t;
    float v = dot(a, b) >= 0 ? t : -t;

    return normalize(a * u + b * v);
}

struct TRS3 {
    float3 translation;
    float4 rotation;
    float3 scale;

    TRS3 NLerp(TRS3 b, float t) {
        return {
            lerp(translation, b.translation, t),
            rot3nlerp(rotation, b.rotation, t),
            lerp(scale, b.scale, t),
        };
    }

    float4x4 Mat4x4() {
        float3 t = translation;
        float4 r = rotation;
        float3 s = scale;

        float4x4 A = {};
        A[0][0] = s[0] * (r[3]*r[3] + r[0]*r[0] - r[1]*r[1] - r[2]*r[2]);
        A[0][1] = s[1] * (r[0]*r[1]*2 - r[3]*r[2]*2);
        A[0][2] = s[2] * (r[3]*r[1]*2 + r[0]*r[2]*2);

        A[1][0] = s[0] * (r[3]*r[2]*2 + r[0]*r[1]*2);
        A[1][1] = s[1] * (r[3]*r[3] - r[0]*r[0] + r[1]*r[1] - r[2]*r[2]);
        A[1][2] = s[2] * (r[1]*r[2]*2 - r[3]*r[0]*2);

        A[2][0] = s[0] * (r[0]*r[2]*2 - r[3]*r[1]*2);
        A[2][1] = s[1] * (r[3]*r[0]*2 + r[1]*r[2]*2);
        A[2][2] = s[2] * (r[3]*r[3] - r[0]*r[0] - r[1]*r[1] + r[2]*r[2]);

        for (int i = 0; i < 3; i++) {
            A[i][3] = t[i];
        }

        A[3][3] = 1;

        return A;
    }
};

/*
struct Update {
    uint32_t instanceIndex;
};
*/

struct AccelInstance {
    float transform[3][4];
    uint32_t instanceIDAndMask;
    uint32_t sbtOffsetAndFlags;
    uint8_t *accelerationStructure;
};

struct Scene {
    uint32_t *parent;
    TRS3 *transformT0;
    TRS3 *transformT1;

    uint8_t *mask;
    uint8_t **blas;

    float4x4 Transform(uint32_t i, float t) {
        float4x4 A = float4x4(
            1, 0, 0, 0,
            0, 1, 0, 0,
            0, 0, 1, 0,
            0, 0, 0, 0,
        );
        for (; i != 0; i = parent[i]) {
            float4x4 B = transformT0[i].NLerp(transformT1[i], t).Mat4x4();
            A = mul(B, A);
        }
        return A;
    }
};

struct Push {
    uint32_t n;
    Scene *scene;
    uint32_t *dirty;
    float time;
    AccelInstance *instances;
};

uint32_t pack24_8(uint32_t x, uint32_t y) {
    return x | (y << 24);
}

[[vk::push_constant]] Push push;

[shader("compute")]
void scene(uint32_t index: SV_DispatchThreadID) {
    if (index >= push.n)
        return;

    uint32_t instanceIndex = push.dirty[index];

    float4x4 transform = push.scene->Transform(instanceIndex, push.time);

    // TODO: outline into a func
    float transform2[3][4];
    for (int i = 0; i < 3; i++) {
        for (int j = 0; j < 4; j++) {
            transform2[i][j] = transform[i][j];
        }
    }

    push.instances[instanceIndex] = {
        transform2,
        pack24_8(instanceIndex, push.scene->mask[instanceIndex]),
        pack24_8(0, 0),
        push.scene->blas[instanceIndex],
    };
}
