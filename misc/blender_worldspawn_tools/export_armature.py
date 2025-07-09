import dataclasses
import json
import math

import bpy
import bmesh

from mathutils import Vector, Quaternion

from .util import asdasd, fixupdict


def save(operator, context, object, outfile):
    armature = object.data

    rest_pose = {}

    for bone in armature.bones:
        # TODO: we should export the matrix as is, as poses might involve
        # shearing, which can't be decomposed into translation, rotation and
        # scale
        t, r, s = bone.matrix_local.inverted_safe().decompose()

        rest_pose[bone.name] = {
            'Translation': t,
            'Rotation':    r,
            'Scale':       s,
        }

    with open(outfile, 'wb') as f:
        f.write(json.dumps(fixupdict(rest_pose), default=asdasd, indent='\t').encode('utf-8'))
