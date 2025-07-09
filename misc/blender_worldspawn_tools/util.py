import numpy as np

from mathutils import Vector, Quaternion

# TODO: type annotations?

np_vec2 = np.dtype((np.float32, 2))
np_vec3 = np.dtype((np.float32, 3))


def np_reshape_in_place(a, newshape):
    reshaped_array = a.view()
    reshaped_array.shape = newshape
    return reshaped_array


def np_array_from_bpy_collection(collection, attr, dtype):
    a = np.empty(len(collection), dtype=dtype)
    collection.foreach_get(attr, np_reshape_in_place(a, -1))
    return a


# TODO: merge into fixupdict and remove in favor of the latter
def asdasd(o):
    match o:
        case Vector():
            return [str(e) for e in o]
        case Quaternion():
            return [str(o[i]) for i in [1, 2, 3, 0]]
    return None


def dict_skip_nulls(stuff):
    return dict((k, v) for (k, v) in stuff if v is not None)


# TODO: rename to dict_fixup?
def fixupdict(d):
    if isinstance(d, dict):
        for k, v in d.items():
            d[k] = fixupdict(v)
    elif isinstance(d, (list, tuple)):
        for k, v in enumerate(d):
            d[k] = fixupdict(v)
    elif isinstance(d, (int, float)):
        d = str(d)
    return d
