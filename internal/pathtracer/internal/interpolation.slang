// TODO: rename to make it clear that we're interpolating on a triangle

float
interpolate(float v0, float v1, float v2, float2 barys)
{
    // TODO: verify that this is ok numerically
    return v0 * (1.0 - barys.x - barys.y) + v1 * barys.x + v2 * barys.y;
}

float2
interpolate2(float2 verts[3], float2 barys)
{
    return float2(
        interpolate(verts[0].x, verts[1].x, verts[2].x, barys),
        interpolate(verts[0].y, verts[1].y, verts[2].y, barys));
}

float3
interpolate3(float3 verts[3], float2 barys)
{
    return float3(
        interpolate(verts[0].x, verts[1].x, verts[2].x, barys),
        interpolate(verts[0].y, verts[1].y, verts[2].y, barys),
        interpolate(verts[0].z, verts[1].z, verts[2].z, barys));
}

float4
interpolate4(float4 verts[3], float2 barys)
{
    return float4(
        interpolate(verts[0].x, verts[1].x, verts[2].x, barys),
        interpolate(verts[0].y, verts[1].y, verts[2].y, barys),
        interpolate(verts[0].z, verts[1].z, verts[2].z, barys),
        interpolate(verts[0].w, verts[1].w, verts[2].w, barys));
}
