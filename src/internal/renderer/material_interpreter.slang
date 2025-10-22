import internal.interpreter;
import scene;

// TODO: rename to mvm.slang?

// TODO: move interpolators into their own file!!!

// TODO: review whether we want to do something more clever (numerically) in the
// interpolators, e.g. do we need to tinker with rounding modes or whatever?

// TODO: move interpolators to a different file pls...

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

/*
public enum TwoSrc {
    FAddE8M23,
    FSubE8M23,
    FMulE8M23,
    FDivE8M23,
    FMinE8M23,
    FMaxE8M23,

    FEqual32,
    FNotEqual32,
    FLessOrEqual32,
};
*/

public void mateval(
    __ref Registers regs,
    uint32_t *program,
    MaterialParams *materialParams,
    float3x4 ObjectToWorld,
    uint3 triIdxs,
    float2 barys_yz)
{
    Fetcher fetcher;
    fetcher.pc = program;

    // TODO: factor decoding out
evalLoop:
    for (;;) {
        let instrw = fetcher.fetch();

        let op = (Op)(instrw & 0xff);
        let dst = (instrw >> 8) & 0xff;
        let src0 = (instrw >> 16) & 0xff;
        let src1 = (instrw >> 24) & 0xff;

        // TODO: reorganize instr encoding to be hierarchical, with the first
        // step being basically "instruction template" and ops following. So
        // e.g. we'd have a enum Opc and we'd have the following Opcs: Stop,
        // DstImm32, DstOneSrc, DstTwoSrc, PredTwoSrc, ...
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
        // TODO: remove this in favor of LoadAttr + DisturbNormal or smth?
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
