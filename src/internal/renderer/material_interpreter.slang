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

    ConditionalSelect32,

    Load, // TODO: rename
    LoadAttribute, // TODO: rename
    // TODO: make normal be loaded either with LoadAttribute or with a dedicated
    // thing.
};

/*
public enum TwoSrc {
    AddF32,
    SubF32,
    MulF32,
    DivF32,
    MinF32,
    MaxF32,

    EqualF32,
    NotEqualF32,
    LessOrEqualF32,
};
*/
