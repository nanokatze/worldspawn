// TODO: replace this with just a float3x3/Mat3x3
public struct Frame {
    public float3 x, y, z;

    public static Frame fromNormal(float3 n) {
        float3 a = abs(n.x) > 0.5 ? float3(0, 0, 1) : float3(1, 0, 0);

        float3 b = normalize(cross(n, a));
        float3 t = cross(n, b);

        return {t, b, n};
    }

    public static Frame fromNT(float3 n, float3 t, float sign = 1.0) {
        if (abs(dot(n, t)) > 0.9) return fromNormal(n);

        float3 b = normalize(cross(n, t)) * sign;
        t = cross(b, n);

        return {t, b, n};
    }

    public float3 worldToLocal(float3 w) {
        return float3(dot(w, x), dot(w, y), dot(w, z));
    }

    public float3 localToWorld(float3 l) {
        return x * l.x + y * l.y + z * l.z;
    }
};
