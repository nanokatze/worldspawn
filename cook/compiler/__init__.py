class Value:


    def __init__(self, id, type, op, imm, *args):
        # TODO: make all of the fields private and expose getters
        self.id = id
        self.type = type
        self.op = op
        self.imm = imm
        self.args = args


class Sea:


    def __init__(self):
        # TODO: make all of the fields private
        self.values = []
        self.idctr = 0
        self.instructions = [] # TODO: kill this
        self.names = dict() # TODO: also probs kill this


    # TODO: make args vararg
    def value(self, type, op, imm, *args):
        # hash cons things
        v = Value(self.idctr, type, op, imm, *args)
        self.idctr += 1
        self.values.append(v)
        self.instructions.append(v)
        self.names[v.id] = "v{}".format(v.id)
        return v


    def set_name(self, v, name):
        self.names[v.id] = name
