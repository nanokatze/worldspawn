float sqr(float x) {
    return x * x;
}

// Move this into mis.slang
public float PowerHeuristic(int nf, float fPdf, int ng, float gPdf) {
    float f = nf * fPdf;
    float g = ng * gPdf;
    if (sqr(f) == 1.0 / 0.0)
        return 1;
    return sqr(f) / (sqr(f) + sqr(g));
}
