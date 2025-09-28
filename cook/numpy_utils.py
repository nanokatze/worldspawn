import numpy as np


vec2 = np.dtype((np.float32, 2))
vec3 = np.dtype((np.float32, 3))
vec3 = np.dtype((np.float32, 3))


# TODO: move to bpy_utils?
def array_from_bpy_collection(collection, attr, dtype):
    array = np.empty(len(collection), dtype=dtype)
    array_flat = array.view()
    array_flat.shape = -1
    collection.foreach_get(attr, array_flat)
    return array


# TODO: rename ndarray_tofile
def write_ndarray(f, a):
    # TODO: endianness
    return f.write(a.tobytes())
