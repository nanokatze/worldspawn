float sqr(float x) {
    return x * x;
}

public float PowerHeuristic(int nf, float fPdf, int ng, float gPdf) {
    float f = nf * fPdf;
    float g = ng * gPdf;
    if (isinf(sqr(f)))
        return 1;
    return sqr(f) / (sqr(f) + sqr(g));
}
