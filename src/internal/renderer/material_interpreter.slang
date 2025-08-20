// TODO: rename to MatOp
public enum Op {
    Stop,

    Movk,

    AddF32,
    SubF32,
    MulF32,
    DivF32,
    MinF32,
    MaxF32,

    FracF32,

    EqualF32,
    NotEqualF32,
    LessOrEqualF32,

    ConditionalSelect,

    Load, // TODO: rename
    LoadAttribute, // TODO: rename

    BSDFOrenNayarDiffuse,
};
