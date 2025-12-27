class Value:


    def __init__(self, id, op, type, imm, *args):
        # TODO: make all of the fields private and expose getters
        self.id = id
        self.op = op
        self.type = type
        self.imm = imm
        self.args = args


class Sea:


    def __init__(self):
        # TODO: make all of the fields private
        self.values = []
        self.idctr = 0
        self.instructions = [] # TODO: kill this
        self.names = dict() # TODO: also probs kill this


    def value(self, op, type, imm, *args):
        # hash cons things
        v = Value(self.idctr, op, type, imm, *args)
        self.idctr += 1
        self.values.append(v)
        self.instructions.append(v)
        self.names[v.id] = "v{}".format(v.id)
        return v


# TODO: eventually remove. Python part should only exist to build and feed stuff
# to the compiler.
class Builder:


    def __init__(self, sea):
        self.sea = sea
        self.rewrite_rules = []


    def value(self, op, type, imm, *args):
        # TODO: apply rewrite rules
        return self.sea.value(op, type, imm, *args)
