public struct Fetcher {
    public uint32_t *pc;

    [[mutating]]
    public uint32_t fetch() {
        return *pc++;
    }
}

// TODO: rename to InterpreterState
public struct Registers {
    // TODO: place the first few regs in actual hw regs and the rest into
    // scratch.
    // TODO: make this a pointer when we get pointers to private/function
    public uint32_t scratch[32];

    public __subscript(int i) -> uint32_t
    {
        get {
            return scratch[i];
        }

        set {
            scratch[i] = newValue;
        }
    }
}
