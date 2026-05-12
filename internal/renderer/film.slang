import gpu;

public struct Film {
    public ImageDescriptor color;

    public ImageDescriptor diffuseAlbedo;
    // public ImageDescriptor specularAlbedo;
    public ImageDescriptor normalAndRoughness;
    public ImageDescriptor depth;
    public ImageDescriptor motion;
    // public ImageDescriptor specularMotion;

    // TODO: AOVs. We can just have these follow the fixed members in the Film I
    // think. Or alternatively we should plop them behind a separate pointer. We
    // could also have an array of *ImageDescriptor and require that fixed
    // function outputs lie at particular indices.
}
