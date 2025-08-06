import idprop
from mathutils import Vector, Quaternion


class ByteBuffer:


    def __init__(self):
        self.__buf = bytearray()


    def seek(self, offset, whence=1):
        assert offset == 0
        assert whence == 1
        return len(self.__buf)


    def write(self, b):
        self.__buf.extend(b)


    def bytes(self):
        return self.__buf


class UTF8Writer:


    def __init__(self, w):
        self.__w = w


    def write(self, s):
        return self.__w.write(s.encode('utf-8'))


# TODO: eventually switch back to vectors and rotation objects being serialized
# to/from a single string
# TODO: merge into fixupdict and remove in favor of the latter
def asdasd(o):
    match o:
        case Vector():
            return [str(e) for e in o]
        case Quaternion():
            return [str(o[i]) for i in [1, 2, 3, 0]]
    return None

# TODO: rename to dict_fixup?
def fixupdict(d):
    # TODO: make it automagically work on dict-like things and list-like things
    if isinstance(d, dict):
        for k, v in d.items():
            d[k] = fixupdict(v)
    elif isinstance(d, (list, tuple)):
        for k, v in enumerate(d):
            d[k] = fixupdict(v)
    elif isinstance(d, (int, float)):
        d = str(d)
    elif isinstance(d, idprop.types.IDPropertyArray):
        d = fixupdict(list(d))
    elif isinstance(d, idprop.types.IDPropertyGroup):
        d = fixupdict(dict(d))
    return d
