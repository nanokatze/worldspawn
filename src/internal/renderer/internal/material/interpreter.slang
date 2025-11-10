import "../interpreter";
import "../../spectrum";
import shading_frame;
import bsdf;
import bsdf_diffuse;
import bsdf_microfacet;
import edf;
import edf_uniform;

public enum BSDFNum {
    _,
    Diffuse,
    MicrofacetGGXTest,
};

public enum EDFNum {
    _,
    Uniform,
};

public struct interpreterABI {
    // TODO: outline this into surface, we can have two surfaces: frontface and
    // backface.

    public uint8_t bsdfs[4];
    public uint8_t bsdfCount;
    public uint8_t bsdfsOff;

    public uint8_t edfs[1];
    public uint8_t edfCount;
    public uint8_t edfsOff; // TODO: move it before the array or something?

    // TODO: kill
    public uint32_t outputsReg;
}

public struct interpreterProgram {
    public interpreterABI abi;
    public uint32_t *code;
}

// TODO: kill kill kill
public struct MaterialParams {
    public interpreterProgram program;

    public void *triangles;
    public uint32_t num_triangles;
    public float3 *posBuffer;
    public float3 *normals;
    public float2 *uvs;
    public float baseColor[3];
    public float emission[3];
}

// TODO: move interpolators into their own file!!!

// TODO: review whether we want to do something more clever (numerically) in the
// interpolators, e.g. do we need to tinker with rounding modes or whatever?

float
interpolate1(float3x1 attrs, float2 barys)
{
    return mul(transpose(attrs), float3(1.0 - barys.x - barys.y, barys));
}

float2
interpolate2(float3x2 attrs, float2 barys)
{
    return mul(transpose(attrs), float3(1.0 - barys.x - barys.y, barys));
}

float3
interpolate3(float3x3 attrs, float2 barys)
{
    return mul(transpose(attrs), float3(1.0 - barys.x - barys.y, barys));
}

public enum Op {
    Stop,

    Copy32,

    Const32,

    FAddE8M23,
    FSubE8M23,
    FMulE8M23,
    FDivE8M23,

    FMinE8M23,
    FMaxE8M23,

    FFloorE8M23,

    FEqual32,
    FLessOrEqual32,

    CondSelect32,

    LoadParam,
    LoadAttr,
    LoadNormal,
};

public void mateval(
    __ref InterpreterState regs,
    uint32_t *code,
    MaterialParams *materialParams,
    float3x4 ObjectToWorld,
    uint3 triIdxs,
    float2 barys_yz)
{
    Fetcher fetcher = {code};

    // TODO: factor decoding out
evalLoop:
    for (;;) {
        let instrw = fetcher.fetch();

        let op = (Op)(instrw & 0xff);
        let dst = (instrw >> 8) & 0xff;
        let src0 = (instrw >> 16) & 0xff;
        let src1 = (instrw >> 24) & 0xff;

        switch (op) {
        default:
            // abort();
        case Op::Stop:
            break evalLoop;

        case Op::Copy32: {
            regs[dst] = regs[src0];
            break;
        }

        case Op::Const32: {
            uint32_t val = fetcher.fetch();
            regs[dst] = val;
            break;
        }

        case Op::FAddE8M23: {
            regs[dst] = reinterpret<uint32_t>(
                reinterpret<float>(regs[src0]) +
                reinterpret<float>(regs[src1]));
            break;
        }
        case Op::FSubE8M23: {
            regs[dst] = reinterpret<uint32_t>(
                reinterpret<float>(regs[src0]) -
                reinterpret<float>(regs[src1]));
            break;
        }
        case Op::FMinE8M23: {
            regs[dst] = reinterpret<uint32_t>(min(
                reinterpret<float>(regs[src0]),
                reinterpret<float>(regs[src1])));
            break;
        }

        case Op::FFloorE8M23: {
            regs[dst] = reinterpret<uint32_t>(floor(reinterpret<float>(regs[src0])));
            break;
        }

        case Op::FLessOrEqual32: {
            regs[dst] = reinterpret<float>(regs[src0]) <= reinterpret<float>(regs[src1]);
            break;
        }

        case Op::CondSelect32: {
            uint32_t predicate = fetcher.fetch();
            regs[dst] = regs[predicate] ? regs[src0] : regs[src1];
            break;
        }

        case Op::LoadParam: {
            uint32_t off = fetcher.fetch();
            regs[dst] = *(uint32_t*)((uint8_t*)materialParams + off);
            break;
        }
        case Op::LoadAttr: {
            uint32_t off = fetcher.fetch();
            float2 *attrbuf = *(float2**)((uint8_t*)materialParams + off);
            float2 attr0 = attrbuf[triIdxs[0]];
            float2 attr1 = attrbuf[triIdxs[1]];
            float2 attr2 = attrbuf[triIdxs[2]];
            // TODO: make a dedicated interpolation function which will take care of the
            // edge cases
            float2 attr = interpolate2(float3x2(attr0, attr1, attr2), barys_yz);
            regs[dst+0] = reinterpret<uint32_t>(attr.x);
            regs[dst+1] = reinterpret<uint32_t>(attr.y);
            // TODO: we'll probs be loading the entire vec3 or vec4 at all times
            // I guess?
            break;
        }
        // TODO: remove this in favor of LoadAttr + DisturbNormal or smth
        case Op::LoadNormal: {
            let normal = mul((float3x3)ObjectToWorld,
                interpolate3(
                    float3x3(
                        materialParams->normals[triIdxs[0]],
                        materialParams->normals[triIdxs[1]],
                        materialParams->normals[triIdxs[2]]),
                    barys_yz));
            regs[dst+0] = reinterpret<uint32_t>(normal.x);
            regs[dst+1] = reinterpret<uint32_t>(normal.y);
            regs[dst+2] = reinterpret<uint32_t>(normal.z);
            break;
        }
        }
    }
}


// TODO: rename
struct RegisterReader {
    InterpreterState regs;
    int i;

    // TODO: rename to readUint32?
    [[mutating]]
    uint32_t read() {
        return regs[i++];
    }
};

Spectrum readSpectrum(__ref RegisterReader rd) {
    return Spectrum(reinterpret<float>(rd.read()),
                    reinterpret<float>(rd.read()),
                    reinterpret<float>(rd.read()));
}

Frame readShadingFrameN(__ref RegisterReader rd) {
    let normal = float3(reinterpret<float>(rd.read()),
                        reinterpret<float>(rd.read()),
                        reinterpret<float>(rd.read()));
    return Frame::fromNormal(normal);
}

BSDFAdapter<DiffuseBSDF> readDiffuseBSDF(__ref RegisterReader tints, __ref RegisterReader bsdfparams) {
    return {
        readSpectrum(tints),
        readShadingFrameN(bsdfparams),
    };
}

BSDFAdapter<MicrofacetBSDF<GGX, TestFresnel>> readMicrofacetGGXTestBSDF(__ref RegisterReader tints, __ref RegisterReader bsdfparams) {
    return {
        readSpectrum(tints),
        readShadingFrameN(bsdfparams),
    };
}

float spectrumAsWeight(Spectrum s) {
    return abs((s.x + s.y + s.z) / 3);
}

// BUG: composition of BSDFs is very broken, we should review and fix it.
//
// See https://github.com/mmp/pbrt-v4/blob/f140d7cba5dc7b941f9346d6b7d1476a05c28c37/src/pbrt/util/sampling.h#L79
public struct EvaluatedInterpretedMaterial : BSDF {
    // TODO: we'd like to make this a non-pointer but slang generates broken
    // code when we do
    public interpreterABI *abi;

    public InterpreterState regs;

    // TODO: would be really nice to implement an iterator over bsdfs so we get
    // dynamic bsdf at each iteration and don't need to open code the chungus
    // switch every time.

    public BSDFFlags Flags() {
        if (abi.bsdfCount == 0)
            return (BSDFFlags)0;

        RegisterReader tints = {regs, abi.outputsReg + abi.bsdfsOff};
        RegisterReader bsdfparams = {regs, abi.outputsReg + abi.bsdfsOff + abi.bsdfCount * 3};

        uint32_t flags = 0;
        for (uint32_t i = 0; i < abi.bsdfCount; i++) {
            switch ((BSDFNum)abi.bsdfs[i]) {
            case BSDFNum::Diffuse: {
                let bsdf = readDiffuseBSDF(tints, bsdfparams);
                flags |= (uint32_t)bsdf.Flags();
                break;
            }

            case BSDFNum::MicrofacetGGXTest: {
                let bsdf = readMicrofacetGGXTestBSDF(tints, bsdfparams);
                flags |= (uint32_t)bsdf.Flags();
                break;
            }

            default:
                // abort();
                break;
            }
        }

        // TODO: should we also include EDFs here?

        return (BSDFFlags)flags;
    }

    public BSDFEval Eval(float3 wo, float3 wi) {
        if (abi.bsdfCount == 0)
            return {};

        RegisterReader tints = {regs, abi.outputsReg + abi.bsdfsOff};
        RegisterReader bsdfparams = {regs, abi.outputsReg + abi.bsdfsOff + abi.bsdfCount * 3};

        BSDFEval result = {};
        for (uint32_t i = 0; i < abi.bsdfCount; i++) {
            float pmf = 1.0 / abi.bsdfCount;

            switch ((BSDFNum)abi.bsdfs[i]) {
            case BSDFNum::Diffuse: {
                let bsdf = readDiffuseBSDF(tints, bsdfparams);
                var e = bsdf.Eval(wo, wi);
                e.pdf *= pmf;
                result = result.Add(e);
                break;
            }
            case BSDFNum::MicrofacetGGXTest: {
                let bsdf = readMicrofacetGGXTestBSDF(tints, bsdfparams);
                var e = bsdf.Eval(wo, wi);
                e.pdf *= pmf;
                result = result.Add(e);
                break;
            }
            default:
                // abort();
                break;
            }
        }

        return result;
    }

    public BSDFSample Sample(float3 wo, float uc, float2 u) {
        if (abi.bsdfCount == 0)
            return {};

#if 0
        float total = 0.0;
        for (uint32_t i = 0; i < abi.bsdfCount; i++) {
            let weight = spectrumAsWeight(spectrumFromRegs(regs, abi.outputsReg + abi.bsdfsOff + i * 3));
            total += weight;
        }

        uint32_t bsdfReg = abi.outputsReg + abi.bsdfsOff + abi.bsdfCount * 3;

        float accumulated = 0.0;
        float pdf = 0;
        uint32_t bsdfIndex;
        for (bsdfIndex = 0;; bsdfIndex++) {
            let weight = spectrumAsWeight(spectrumFromRegs(regs, abi.outputsReg + abi.bsdfsOff + bsdfIndex * 3));

            pdf = weight / total;

            if (bsdfIndex >= abi.bsdfCount)
                break;
            if (accumulated + pdf > uc)
                break;

            switch ((BSDFNum)abi.bsdfs[bsdfIndex]) {
            case BSDFNum::Diffuse: {
                bsdfReg += 3 /* normal */;
                break;
            }
            default:
                // abort();
                break;
            }

            accumulated += pdf;
        }
#endif

        // TODO: be a lil smarter about assigning probabilities
        let pmf = 1.0 / abi.bsdfCount;
        let bsdfIndex = (int)fract(uc * abi.bsdfCount);

        // TODO: adjust uc here so that it's in 0..1 again and useful for
        // downstream bsdf sample

        RegisterReader tints = {regs, abi.outputsReg + abi.bsdfsOff};
        RegisterReader bsdfparams = {regs, abi.outputsReg + abi.bsdfsOff + abi.bsdfCount * 3};

        for (int i = 0; i < bsdfIndex; i++) {
            switch ((BSDFNum)abi.bsdfs[i]) {
            case BSDFNum::Diffuse: {
                readDiffuseBSDF(tints, bsdfparams);
                break;
            }

            case BSDFNum::MicrofacetGGXTest: {
                readMicrofacetGGXTestBSDF(tints, bsdfparams);
                break;
            }

            default:
                // abort();
                break;
            }
        }

        switch ((BSDFNum)abi.bsdfs[bsdfIndex]) {
        case BSDFNum::Diffuse: {
            let bsdf = readDiffuseBSDF(tints, bsdfparams);
            var sample = bsdf.Sample(wo, uc, u);
            sample.pdf *= pmf;
            return sample;
        }
        case BSDFNum::MicrofacetGGXTest: {
            let bsdf = readMicrofacetGGXTestBSDF(tints, bsdfparams);
            var sample = bsdf.Sample(wo, uc, u);
            sample.pdf *= pmf;
            return sample;
        }
        default:
            // abort();
            return {};
        }
    }

    // TODO: fold into Eval? Blender doesn't, so I guess we shouldn't either.
    // TODO: rename to EmissionEval? And rename BSDF eval and sample to
    // EvalScattering and SampleScattering? Or EDFEval, BSDFEval, BSDFSample?
    public Spectrum EDFEval(float3 wi) {
        if (abi.edfCount == 0)
            return (Spectrum)0;

        RegisterReader tints = {regs, abi.outputsReg + abi.edfsOff};

        // TODO: edfparams

        Spectrum result = (Spectrum)0;
        for (uint32_t i = 0; i < abi.edfCount; i++) {
            switch ((EDFNum)abi.edfs[i]) {
            case EDFNum::Uniform: {
                EDFAdapter<UniformEDF> edf = {
                    readSpectrum(tints),
                    Frame::fromNormal(float3(0.0, 0.0, 1.0)), // should be geometric normal
                };
                result += edf.Eval(wi);
                break;
            }

            default:
                // abort();
                break;
            }
        }

        return result;
    }
}
