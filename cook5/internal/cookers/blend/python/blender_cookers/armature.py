import json


import bpyutil
import util


def deps(context, obj, dset):
    dset.add_product((context.path_for_datablock(obj), 'Object', obj.name))


def cook(context, obj):
    armature = obj.data

    # TODO: joints should be sorted by hierarchy relationship (parents should
    # appear before their children) and then also by name.

    joints = []

    for bone in armature.bones:
        joints.append({
            'Name':     bone.name,
            'Parent':   next((i for i, parent in enumerate(armature.bones) if parent == bone.parent), -1),
            'BindPose': bone.matrix_local,
        })

    joints = bpyutil.fixupdict(joints)

    with context.create(context.path_for_datablock(obj)) as f:
        json.dump(joints, util.UTF8Writer(f), default=bpyutil.asdasd, indent='\t')
