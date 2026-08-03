import numpy as np


vec2 = np.dtype((np.float32, 2))
vec3 = np.dtype((np.float32, 3))
vec4 = np.dtype((np.float32, 4))


def tofile(f, a):
    # TODO: endianness
    return f.write(a.tobytes())
