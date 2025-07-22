class Instr:


    # TODO: specify type
    def __init__(self, op, args, aux=None):
        self.op = op
        self.args = args
        self.aux = aux
        # do not hash this!
        self.sources = [] # TODO: plop this stuff behind a DebugInfo object


    def add_source(self, src):
        self.sources.append(src)


class CSE:
    pass


class Builder:
    pass


# TODO: generate these automagically
def BSDFPrincipled(BaseColor, Metallic, Roughness, IOR, Alpha):
    return Instr('BSDFPrincipled', [
        BaseColor,
        Metallic,
        Roughness,
        IOR,
        Alpha,
    ])
