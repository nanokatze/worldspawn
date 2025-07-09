import json
import math

import bpy
from mathutils import Vector, Quaternion

from .util import asdasd

if True:
    return

scene = bpy.context.scene

obj = bpy.context.active_object

anim_data = obj.animation_data

# TODO: do we want to export animation data blocks or animations used in the NLA tracks from the active armature?

action = anim_data.action

# for bone in obj.data.bones:
#     print(bone.name, bone.matrix_local)

# TODO: is this the range we should use?

animation = {}

# TODO: have a flat list of channels

for frame in range(math.floor(action.frame_range[0]), math.ceil(action.frame_range[1]) + 1):
    scene.frame_set(frame)

    for bone in obj.pose.bones:
        bone_animation = animation.setdefault(bone.name, {})

        T, R, S = bone.matrix_basis.decompose()
        bone_animation.setdefault('Translation', []).append(T)
        bone_animation.setdefault('Rotation', []).append(R)
        bone_animation.setdefault('Scale', []).append(S)
        for k, v in bone.items():
            bone_animation.setdefault(bone.name, []).append(v)

dead_bones = []
for bone, bone_animation in animation.items():
    if all(all(v == channel[0] for v in channel) for channel in bone_animation.values()):
        dead_bones.append(bone)
for dead_bone in dead_bones:
    del animation[dead_bone]

with open(action.name + '.ani', 'wb') as f:
    f.write(json.dumps({'Duration': int((action.frame_range[1] - action.frame_range[0]) * 0.01 * 1e9), 'Bones': animation}, indent='\t', default=asdasd).encode('utf-8'))
