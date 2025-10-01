import numpy as np
import idprop
from mathutils import Vector, Quaternion


def array_from_prop_collection(collection, attr, dtype):
    array = np.empty(len(collection), dtype=dtype)
    array_flat = array.view()
    array_flat.shape = -1
    collection.foreach_get(attr, array_flat)
    return array


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
