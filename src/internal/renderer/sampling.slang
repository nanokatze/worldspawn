static const float M_PI = 3.141592653589793;

public float3 SampleCosineHemisphere(float2 u) {
  let phi = u.x * 2.0f * M_PI;
  let sinTheta = sqrt(u.y);
  let cosTheta = sqrt(1.0f - u.y);

  float cosPhi;
  float sinPhi;
  sincos(phi, sinPhi, cosPhi);

  return float3(cosPhi * sinTheta, sinPhi * sinTheta, cosTheta);
}

public float CosineHemispherePDF(float cosTheta) {
    return cosTheta / M_PI;
}