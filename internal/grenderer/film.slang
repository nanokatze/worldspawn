import gpu;

public struct Film {
    public ImageDescriptors color;

    public ImageDescriptors diffuseAlbedo;
    // public ImageDescriptors specularAlbedo;
    public ImageDescriptors normalAndRoughness;
    public ImageDescriptors depth;
    public ImageDescriptors motion;
    // public ImageDescriptors specularMotion;

    // TODO: AOVs. We can just have these follow the fixed members in the Film I
    // think. Or alternatively we should plop them behind a separate pointer. We
    // could also have an array of *ImageDescriptors and require that fixed
    // function outputs lie at particular indices.
}
